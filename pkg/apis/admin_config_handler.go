package apis

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/configstore"
	"simple-one-api/pkg/initializer"
)

const redactedValue = "__SIMPLE_ONE_REDACTED__"

type AdminConfigDraft struct {
	Config       config.Configuration  `json:"config"`
	DatabasePath string                `json:"database_path"`
	Revision     *configstore.Revision `json:"revision,omitempty"`
}

type AdminPublishRequest struct {
	Config config.Configuration `json:"config"`
	Note   string               `json:"note"`
}

type AdminPublishResponse struct {
	Revision        configstore.Revision `json:"revision"`
	RestartRequired bool                 `json:"restart_required"`
	RestartFields   []string             `json:"restart_fields"`
	AuthChanged     bool                 `json:"auth_changed"`
}

func AdminConfigDraftHandler(c *gin.Context) {
	conf := cloneConfiguration(*config.CurrentConfiguration())
	maskConfigurationSecrets(&conf)
	response := AdminConfigDraft{Config: conf}
	if store := initializer.ConfigStore(); store != nil {
		response.DatabasePath = store.Path()
		if revision, _, err := store.Active(c.Request.Context()); err == nil {
			response.Revision = &revision
		}
	}
	c.JSON(http.StatusOK, response)
}

func AdminConfigValidateHandler(c *gin.Context) {
	var request AdminPublishRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid configuration JSON", "detail": err.Error()})
		return
	}
	if err := restoreConfigurationSecrets(&request.Config, config.CurrentConfiguration()); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "restore configuration secrets", "detail": err.Error()})
		return
	}
	issues := config.ValidateConfiguration(request.Config)
	c.JSON(http.StatusOK, gin.H{"valid": len(issues) == 0, "issues": issues})
}

func AdminConfigPublishHandler(c *gin.Context) {
	var request AdminPublishRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid configuration JSON", "detail": err.Error()})
		return
	}
	previous := config.CurrentConfiguration()
	if err := restoreConfigurationSecrets(&request.Config, previous); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "restore configuration secrets", "detail": err.Error()})
		return
	}
	authChanged := previous.APIKey != request.Config.APIKey
	restartFields := restartRequiredFields(*previous, request.Config)
	revision, err := initializer.PublishConfiguration(c.Request.Context(), request.Config, "admin", strings.TrimSpace(request.Note))
	if err != nil {
		var validationError *initializer.ConfigurationValidationError
		if errors.As(err, &validationError) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "configuration validation failed", "issues": validationError.Issues})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "publish configuration", "detail": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, AdminPublishResponse{
		Revision: revision, RestartRequired: len(restartFields) > 0, RestartFields: restartFields, AuthChanged: authChanged,
	})
}

func AdminConfigRevisionsHandler(c *gin.Context) {
	store := initializer.ConfigStore()
	if store == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "SQLite configuration repository is unavailable"})
		return
	}
	revisions, err := store.List(c.Request.Context(), 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list configuration revisions", "detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": revisions})
}

func AdminConfigActivateHandler(c *gin.Context) {
	var path struct {
		ID int64 `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&path); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid revision id"})
		return
	}
	store := initializer.ConfigStore()
	if store == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "SQLite configuration repository is unavailable"})
		return
	}
	_, payload, err := store.Revision(c.Request.Context(), path.ID)
	if errors.Is(err, sql.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "configuration revision not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "read configuration revision", "detail": err.Error()})
		return
	}
	var target config.Configuration
	if err := json.Unmarshal(payload, &target); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "decode configuration revision"})
		return
	}
	restartFields := restartRequiredFields(*config.CurrentConfiguration(), target)
	authChanged := config.CurrentAPIKey() != target.APIKey
	revision, err := initializer.ActivateConfiguration(c.Request.Context(), path.ID)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "activate configuration revision", "detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, AdminPublishResponse{
		Revision: revision, RestartRequired: len(restartFields) > 0, RestartFields: restartFields, AuthChanged: authChanged,
	})
}

func restartRequiredFields(previous, next config.Configuration) []string {
	if prepared, err := config.PrepareConfiguration(previous, ""); err == nil {
		previous = prepared.Configuration()
	}
	if prepared, err := config.PrepareConfiguration(next, ""); err == nil {
		next = prepared.Configuration()
	}
	fields := make([]string, 0, 3)
	if previous.ServerPort != next.ServerPort {
		fields = append(fields, "server_port")
	}
	if previous.EnableWeb != next.EnableWeb {
		fields = append(fields, "enable_web")
	}
	if previous.LogLevel != next.LogLevel || previous.Debug != next.Debug {
		fields = append(fields, "log_level")
	}
	return fields
}

func cloneConfiguration(conf config.Configuration) config.Configuration {
	payload, _ := json.Marshal(conf)
	var clone config.Configuration
	_ = json.Unmarshal(payload, &clone)
	return clone
}

func maskConfigurationSecrets(conf *config.Configuration) {
	if conf.APIKey != "" {
		conf.APIKey = secretPlaceholder("api_key")
	}
	conf.Proxy.HTTPProxy = maskProxyCredentials(conf.Proxy.HTTPProxy, "proxy.http_proxy")
	conf.Proxy.HTTPSProxy = maskProxyCredentials(conf.Proxy.HTTPSProxy, "proxy.https_proxy")
	conf.Proxy.Socks5Proxy = maskProxyCredentials(conf.Proxy.Socks5Proxy, "proxy.socks5_proxy")
	for index := range conf.APIKeys {
		if conf.APIKeys[index].APIKey != "" {
			conf.APIKeys[index].APIKey = secretPlaceholder(fmt.Sprintf("api_keys.%d.api_key", index))
		}
	}
	for serviceName := range conf.Services {
		for index := range conf.Services[serviceName] {
			model := &conf.Services[serviceName][index]
			base := fmt.Sprintf("services.%s.%d", serviceName, index)
			maskSecretMap(model.Credentials, base+".credentials")
			for credentialIndex := range model.CredentialList {
				maskSecretMap(model.CredentialList[credentialIndex], fmt.Sprintf("%s.credential_list.%d", base, credentialIndex))
			}
		}
	}
	maskExtensionSecrets(conf)
}

func maskExtensionSecrets(conf *config.Configuration) {
	maskSecretMap(conf.Extensions, "configuration")
	maskSecretMap(conf.Proxy.Extensions, "proxy")
	maskSecretMap(conf.Translation.Extensions, "translation")
	for index := range conf.APIKeys {
		maskSecretMap(conf.APIKeys[index].Extensions, fmt.Sprintf("api_keys.%d", index))
	}
	for name, params := range conf.ParamsRange {
		maskSecretMap(params.Extensions, "params_range."+name)
		maskSecretMap(params.TemperatureRange.Extensions, "params_range."+name+".temperatureRange")
		maskSecretMap(params.TopPRange.Extensions, "params_range."+name+".topPRange")
	}
	for serviceName := range conf.Services {
		for index := range conf.Services[serviceName] {
			model := &conf.Services[serviceName][index]
			base := fmt.Sprintf("services.%s.%d", serviceName, index)
			maskSecretMap(model.Extensions, base)
			maskSecretMap(model.Limit.Extensions, base+".limit")
			maskSecretMap(model.EmbeddingLimit.Extensions, base+".embedding_limit")
		}
	}
}

func maskSecretMap(values map[string]interface{}, path string) {
	for key, value := range values {
		if isSecretField(key) {
			if value != nil && value != "" {
				values[key] = secretPlaceholder(path + "." + key)
			}
			continue
		}
		if nested, ok := value.(map[string]interface{}); ok {
			maskSecretMap(nested, path+"."+key)
			continue
		}
		if items, ok := value.([]interface{}); ok {
			maskSecretSlice(items, path+"."+key)
		}
	}
}

func maskSecretSlice(items []interface{}, path string) {
	for index, value := range items {
		switch nested := value.(type) {
		case map[string]interface{}:
			maskSecretMap(nested, fmt.Sprintf("%s.%d", path, index))
		case []interface{}:
			maskSecretSlice(nested, fmt.Sprintf("%s.%d", path, index))
		}
	}
}

func restoreConfigurationSecrets(candidate *config.Configuration, current *config.Configuration) error {
	secrets := collectConfigurationSecrets(current)
	if isSecretPlaceholder(candidate.APIKey) {
		value, found := secrets[candidate.APIKey]
		if !found {
			return errors.New("unknown api_key placeholder; reload the draft")
		}
		candidate.APIKey = value
	}
	var err error
	if candidate.Proxy.HTTPProxy, err = restoreProxyCredentials(candidate.Proxy.HTTPProxy, secrets); err != nil {
		return err
	}
	if candidate.Proxy.HTTPSProxy, err = restoreProxyCredentials(candidate.Proxy.HTTPSProxy, secrets); err != nil {
		return err
	}
	if candidate.Proxy.Socks5Proxy, err = restoreProxyCredentials(candidate.Proxy.Socks5Proxy, secrets); err != nil {
		return err
	}
	for index := range candidate.APIKeys {
		if isSecretPlaceholder(candidate.APIKeys[index].APIKey) {
			value, found := secrets[candidate.APIKeys[index].APIKey]
			if !found {
				return fmt.Errorf("unknown api_keys.%d.api_key placeholder; reload the draft", index)
			}
			candidate.APIKeys[index].APIKey = value
		}
	}
	for serviceName := range candidate.Services {
		for index := range candidate.Services[serviceName] {
			model := &candidate.Services[serviceName][index]
			if err := restoreSecretMap(model.Credentials, secrets); err != nil {
				return err
			}
			for credentialIndex := range model.CredentialList {
				if err := restoreSecretMap(model.CredentialList[credentialIndex], secrets); err != nil {
					return err
				}
			}
		}
	}
	if err := restoreExtensionSecrets(candidate, secrets); err != nil {
		return err
	}
	return nil
}

func restoreExtensionSecrets(conf *config.Configuration, secrets map[string]string) error {
	maps := []map[string]interface{}{conf.Extensions, conf.Proxy.Extensions, conf.Translation.Extensions}
	for index := range conf.APIKeys {
		maps = append(maps, conf.APIKeys[index].Extensions)
	}
	for _, params := range conf.ParamsRange {
		maps = append(maps, params.Extensions, params.TemperatureRange.Extensions, params.TopPRange.Extensions)
	}
	for serviceName := range conf.Services {
		for index := range conf.Services[serviceName] {
			model := &conf.Services[serviceName][index]
			maps = append(maps, model.Extensions, model.Limit.Extensions, model.EmbeddingLimit.Extensions)
		}
	}
	for _, values := range maps {
		if err := restoreSecretMap(values, secrets); err != nil {
			return err
		}
	}
	return nil
}

func collectConfigurationSecrets(current *config.Configuration) map[string]string {
	masked := cloneConfiguration(*current)
	secrets := make(map[string]string)
	collect := func(placeholder, value string) {
		if value != "" {
			secrets[placeholder] = value
		}
	}
	collect(secretPlaceholder("api_key"), current.APIKey)
	collect(secretPlaceholder("proxy.http_proxy"), current.Proxy.HTTPProxy)
	collect(secretPlaceholder("proxy.https_proxy"), current.Proxy.HTTPSProxy)
	collect(secretPlaceholder("proxy.socks5_proxy"), current.Proxy.Socks5Proxy)
	for index := range current.APIKeys {
		collect(secretPlaceholder(fmt.Sprintf("api_keys.%d.api_key", index)), current.APIKeys[index].APIKey)
	}
	maskConfigurationSecrets(&masked)
	collectExtensionSecrets(&masked, current, secrets)
	for serviceName, models := range masked.Services {
		for index, model := range models {
			collectSecretMap(model.Credentials, current.Services[serviceName][index].Credentials, secrets)
			for credentialIndex := range model.CredentialList {
				collectSecretMap(model.CredentialList[credentialIndex], current.Services[serviceName][index].CredentialList[credentialIndex], secrets)
			}
		}
	}
	return secrets
}

func collectExtensionSecrets(masked, current *config.Configuration, secrets map[string]string) {
	collectSecretMap(masked.Extensions, current.Extensions, secrets)
	collectSecretMap(masked.Proxy.Extensions, current.Proxy.Extensions, secrets)
	collectSecretMap(masked.Translation.Extensions, current.Translation.Extensions, secrets)
	for index := range masked.APIKeys {
		if index < len(current.APIKeys) {
			collectSecretMap(masked.APIKeys[index].Extensions, current.APIKeys[index].Extensions, secrets)
		}
	}
	for name, maskedParams := range masked.ParamsRange {
		currentParams, exists := current.ParamsRange[name]
		if !exists {
			continue
		}
		collectSecretMap(maskedParams.Extensions, currentParams.Extensions, secrets)
		collectSecretMap(maskedParams.TemperatureRange.Extensions, currentParams.TemperatureRange.Extensions, secrets)
		collectSecretMap(maskedParams.TopPRange.Extensions, currentParams.TopPRange.Extensions, secrets)
	}
	for serviceName, models := range masked.Services {
		currentModels := current.Services[serviceName]
		for index, model := range models {
			if index >= len(currentModels) {
				continue
			}
			collectSecretMap(model.Extensions, currentModels[index].Extensions, secrets)
			collectSecretMap(model.Limit.Extensions, currentModels[index].Limit.Extensions, secrets)
			collectSecretMap(model.EmbeddingLimit.Extensions, currentModels[index].EmbeddingLimit.Extensions, secrets)
		}
	}
}

func collectSecretMap(masked, current map[string]interface{}, secrets map[string]string) {
	for key, maskedValue := range masked {
		if placeholder, ok := maskedValue.(string); ok && isSecretPlaceholder(placeholder) {
			if value, ok := current[key].(string); ok {
				secrets[placeholder] = value
			}
			continue
		}
		maskedNested, maskedOK := maskedValue.(map[string]interface{})
		currentNested, currentOK := current[key].(map[string]interface{})
		if maskedOK && currentOK {
			collectSecretMap(maskedNested, currentNested, secrets)
			continue
		}
		maskedSlice, maskedOK := maskedValue.([]interface{})
		currentSlice, currentOK := current[key].([]interface{})
		if maskedOK && currentOK {
			collectSecretSlice(maskedSlice, currentSlice, secrets)
		}
	}
}

func collectSecretSlice(masked, current []interface{}, secrets map[string]string) {
	for index, maskedValue := range masked {
		if index >= len(current) {
			return
		}
		switch nested := maskedValue.(type) {
		case map[string]interface{}:
			if currentNested, ok := current[index].(map[string]interface{}); ok {
				collectSecretMap(nested, currentNested, secrets)
			}
		case []interface{}:
			if currentNested, ok := current[index].([]interface{}); ok {
				collectSecretSlice(nested, currentNested, secrets)
			}
		}
	}
}

func restoreSecretMap(candidate map[string]interface{}, secrets map[string]string) error {
	for key, value := range candidate {
		if placeholder, ok := value.(string); ok && isSecretPlaceholder(placeholder) {
			original, exists := secrets[placeholder]
			if !exists {
				return fmt.Errorf("unknown secret placeholder for %s; reload the draft", key)
			}
			candidate[key] = original
			continue
		}
		if nestedCandidate, ok := value.(map[string]interface{}); ok {
			if err := restoreSecretMap(nestedCandidate, secrets); err != nil {
				return err
			}
			continue
		}
		if items, ok := value.([]interface{}); ok {
			if err := restoreSecretSlice(items, secrets); err != nil {
				return err
			}
		}
	}
	return nil
}

func restoreSecretSlice(items []interface{}, secrets map[string]string) error {
	for index, value := range items {
		if placeholder, ok := value.(string); ok && isSecretPlaceholder(placeholder) {
			original, exists := secrets[placeholder]
			if !exists {
				return fmt.Errorf("unknown secret placeholder at index %d; reload the draft", index)
			}
			items[index] = original
			continue
		}
		switch nested := value.(type) {
		case map[string]interface{}:
			if err := restoreSecretMap(nested, secrets); err != nil {
				return err
			}
		case []interface{}:
			if err := restoreSecretSlice(nested, secrets); err != nil {
				return err
			}
		}
	}
	return nil
}

func secretPlaceholder(path string) string { return redactedValue + ":" + path }

func isSecretPlaceholder(value string) bool {
	return value == redactedValue || strings.HasPrefix(value, redactedValue+":")
}

func isSecretField(key string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(key, "-", "_"), " ", "_"))
	for _, fragment := range []string{"api_key", "apikey", "secret", "password", "token", "authorization", "access_key"} {
		if strings.Contains(normalized, fragment) {
			return true
		}
	}
	return false
}

func maskProxyCredentials(value, path string) string {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.User == nil {
		return value
	}
	parsed.User = url.User(secretPlaceholder(path))
	return parsed.String()
}

func restoreProxyCredentials(candidate string, secrets map[string]string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(candidate))
	if err != nil || parsed.User == nil {
		return candidate, nil
	}
	placeholder := parsed.User.Username()
	if !isSecretPlaceholder(placeholder) {
		return candidate, nil
	}
	original, found := secrets[placeholder]
	if !found {
		return "", fmt.Errorf("unknown proxy placeholder for %s; reload the draft", placeholder)
	}
	originalURL, err := url.Parse(original)
	if err != nil || originalURL.User == nil {
		return "", fmt.Errorf("stored proxy credentials for %s are invalid", placeholder)
	}
	parsed.User = originalURL.User
	return parsed.String(), nil
}
