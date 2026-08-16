package apis

import (
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"simple-one-api/pkg/config"
)

type PublicServiceModel struct {
	Provider        string            `json:"provider"`
	EmbeddingModels []string          `json:"embedding_models"`
	EmbeddingLimit  PublicLimit       `json:"embedding_limit"`
	Models          []string          `json:"models"`
	ReasoningModels map[string]string `json:"reasoning_models"`
	Enabled         bool              `json:"enabled"`
	ServerURL       string            `json:"server_url"`
	ModelMap        map[string]string `json:"model_map"`
	ModelRedirect   map[string]string `json:"model_redirect"`
	Limit           PublicLimit       `json:"limit"`
	Timeout         int               `json:"timeout"`
	CredentialCount int               `json:"credential_count"`
}

type PublicLimit struct {
	QPS         float64 `json:"qps"`
	QPM         float64 `json:"qpm"`
	RPM         float64 `json:"rpm"`
	Concurrency float64 `json:"concurrency"`
	Timeout     int     `json:"timeout"`
}

type PublicTranslation struct {
	Enable         bool   `json:"enable"`
	PromptTemplate string `json:"promptTemplate"`
	Retry          int    `json:"retry"`
	Concurrency    int    `json:"concurrency"`
}

type PublicRange struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

type PublicModelParams struct {
	TemperatureRange PublicRange `json:"temperatureRange"`
	TopPRange        PublicRange `json:"topPRange"`
	MaxTokens        int         `json:"maxTokens"`
}

type PublicAPIKeyConfig struct {
	APIKey          string              `json:"api_key"`
	SupportedModels map[string][]string `json:"supported_models"`
}

type PublicProxyConfig struct {
	Strategy    string `json:"strategy"`
	Type        string `json:"type"`
	HTTPProxy   string `json:"http_proxy"`
	HTTPSProxy  string `json:"https_proxy"`
	Socks5Proxy string `json:"socks5_proxy"`
	Timeout     int    `json:"timeout"`
}

type PublicConfig struct {
	ServerPort         string                          `json:"server_port"`
	Debug              bool                            `json:"debug"`
	LogLevel           string                          `json:"log_level"`
	LoadBalancing      string                          `json:"load_balancing"`
	EnableWeb          bool                            `json:"enable_web"`
	Proxy              PublicProxyConfig               `json:"proxy"`
	Translation        PublicTranslation               `json:"translation"`
	MultiContentModels []string                        `json:"multi_content_models"`
	ModelRedirect      map[string]string               `json:"model_redirect"`
	ParamsRange        map[string]PublicModelParams    `json:"params_range"`
	Services           map[string][]PublicServiceModel `json:"services"`
	APIKey             string                          `json:"api_key"`
	APIKeys            []PublicAPIKeyConfig            `json:"api_keys"`
	ConfigPath         string                          `json:"config_path"`
}

type AdminStatus struct {
	StartedAt    time.Time `json:"started_at"`
	Uptime       string    `json:"uptime"`
	GoVersion    string    `json:"go_version"`
	GOOS         string    `json:"goos"`
	GOARCH       string    `json:"goarch"`
	ConfigPath   string    `json:"config_path"`
	ModelCount   int       `json:"model_count"`
	ServiceCount int       `json:"service_count"`
	APIKeyCount  int       `json:"api_key_count"`
	EnableWeb    bool      `json:"enable_web"`
}

var startedAt = time.Now()

func AdminStatusHandler(c *gin.Context) {
	conf := config.CurrentConfiguration()
	c.JSON(http.StatusOK, AdminStatus{
		StartedAt:    startedAt,
		Uptime:       time.Since(startedAt).Round(time.Second).String(),
		GoVersion:    runtime.Version(),
		GOOS:         runtime.GOOS,
		GOARCH:       runtime.GOARCH,
		ConfigPath:   config.CurrentConfigPath(),
		ModelCount:   len(config.CurrentSupportModels()),
		ServiceCount: len(config.CurrentModelToService()),
		APIKeyCount:  len(conf.APIKeys),
		EnableWeb:    conf.EnableWeb,
	})
}

func AdminConfigHandler(c *gin.Context) {
	conf := config.CurrentConfiguration()
	pub := PublicConfig{
		ServerPort:    conf.ServerPort,
		Debug:         conf.Debug,
		LogLevel:      conf.LogLevel,
		LoadBalancing: conf.LoadBalancing,
		EnableWeb:     conf.EnableWeb,
		Proxy:         publicProxyConfig(conf.Proxy),
		Translation: PublicTranslation{
			Enable: conf.Translation.Enable, PromptTemplate: conf.Translation.PromptTemplate,
			Retry: conf.Translation.Retry, Concurrency: conf.Translation.Concurrency,
		},
		MultiContentModels: append([]string(nil), conf.MultiContentModels...),
		ModelRedirect:      cloneStringMap(conf.ModelRedirect),
		ParamsRange:        publicParamsRange(conf.ParamsRange),
		Services:           make(map[string][]PublicServiceModel, len(conf.Services)),
		APIKey:             redactSecret(conf.APIKey),
		APIKeys:            make([]PublicAPIKeyConfig, 0, len(conf.APIKeys)),
		ConfigPath:         config.CurrentConfigPath(),
	}

	for serviceName, serviceModels := range conf.Services {
		items := make([]PublicServiceModel, 0, len(serviceModels))
		for _, serviceModel := range serviceModels {
			credentialCount := len(serviceModel.CredentialList)
			if len(serviceModel.Credentials) > 0 {
				credentialCount++
			}
			items = append(items, PublicServiceModel{
				Provider:        serviceModel.Provider,
				EmbeddingModels: append([]string(nil), serviceModel.EmbeddingModels...),
				EmbeddingLimit:  publicLimit(serviceModel.EmbeddingLimit),
				Models:          append([]string(nil), serviceModel.Models...),
				ReasoningModels: cloneStringMap(serviceModel.ReasoningModels),
				Enabled:         serviceModel.Enabled,
				ServerURL:       serviceModel.ServerURL,
				ModelMap:        cloneStringMap(serviceModel.ModelMap),
				ModelRedirect:   cloneStringMap(serviceModel.ModelRedirect),
				Limit:           publicLimit(serviceModel.Limit),
				Timeout:         serviceModel.Timeout,
				CredentialCount: credentialCount,
			})
		}
		pub.Services[serviceName] = items
	}

	for _, apiKey := range conf.APIKeys {
		pub.APIKeys = append(pub.APIKeys, PublicAPIKeyConfig{
			APIKey:          redactSecret(apiKey.APIKey),
			SupportedModels: apiKey.SupportedModels,
		})
	}

	c.JSON(http.StatusOK, pub)
}

func publicLimit(limit config.Limit) PublicLimit {
	return PublicLimit{QPS: limit.QPS, QPM: limit.QPM, RPM: limit.RPM, Concurrency: limit.Concurrency, Timeout: limit.Timeout}
}

func publicParamsRange(values map[string]config.ModelParams) map[string]PublicModelParams {
	if values == nil {
		return nil
	}
	result := make(map[string]PublicModelParams, len(values))
	for name, params := range values {
		result[name] = PublicModelParams{
			TemperatureRange: PublicRange{Min: params.TemperatureRange.Min, Max: params.TemperatureRange.Max},
			TopPRange:        PublicRange{Min: params.TopPRange.Min, Max: params.TopPRange.Max},
			MaxTokens:        params.MaxTokens,
		}
	}
	return result
}

func cloneStringMap(src map[string]string) map[string]string {
	if src == nil {
		return nil
	}
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func redactSecret(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) <= 8 {
		return strings.Repeat("*", len(value))
	}
	return value[:4] + strings.Repeat("*", len(value)-8) + value[len(value)-4:]
}

func publicProxyConfig(proxy config.ProxyConf) PublicProxyConfig {
	return PublicProxyConfig{
		Strategy:    proxy.Strategy,
		Type:        proxy.Type,
		HTTPProxy:   redactProxyURL(proxy.HTTPProxy),
		HTTPSProxy:  redactProxyURL(proxy.HTTPSProxy),
		Socks5Proxy: redactProxyURL(proxy.Socks5Proxy),
		Timeout:     proxy.Timeout,
	}
}

func redactProxyURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return "[redacted]"
	}
	if parsed.User != nil {
		parsed.User = url.User("redacted")
	}
	return parsed.String()
}
