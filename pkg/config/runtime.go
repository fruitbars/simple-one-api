package config

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
)

type ValidationIssue struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

type runtimeSnapshot struct {
	configuration      *Configuration
	configPath         string
	modelToService     map[string][]ModelDetails
	supportModels      map[string]string
	loadBalancing      string
	serverPort         string
	apiKey             string
	debug              bool
	logLevel           string
	globalRedirect     map[string]string
	multiContentModels []string
	proxy              *ProxyConf
	translation        *Translation
	apiKeys            map[string]APIKeyConfig
}

var activeSnapshot atomic.Pointer[runtimeSnapshot]
var serviceIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
var unsafeServiceIDCharacters = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

type PreparedConfiguration struct {
	snapshot *runtimeSnapshot
}

func ValidateConfiguration(conf Configuration) []ValidationIssue {
	payload, err := json.Marshal(conf)
	if err != nil {
		return []ValidationIssue{{Path: "configuration", Message: err.Error()}}
	}
	var normalized Configuration
	if err := json.Unmarshal(payload, &normalized); err != nil {
		return []ValidationIssue{{Path: "configuration", Message: err.Error()}}
	}
	normalizeConfiguration(&normalized)
	return validateNormalizedConfiguration(normalized)
}

func validateNormalizedConfiguration(conf Configuration) []ValidationIssue {
	issues := make([]ValidationIssue, 0)
	if conf.ServerPort != "" {
		_, port, err := net.SplitHostPort(conf.ServerPort)
		portNumber, numberErr := strconv.Atoi(port)
		if err != nil || numberErr != nil || portNumber < 1 || portNumber > 65535 {
			issues = append(issues, ValidationIssue{Path: "server_port", Message: "must be in :9090 or host:port format"})
		}
	}
	if value := strings.ToLower(strings.TrimSpace(conf.LoadBalancing)); value != "" {
		allowed := map[string]bool{"first": true, "random": true, "rand": true, "round_robin": true, "rr": true, "hash": true}
		if !allowed[value] {
			issues = append(issues, ValidationIssue{Path: "load_balancing", Message: "supported values: first, random, round_robin, hash"})
		}
	}
	if value := strings.ToLower(strings.TrimSpace(conf.LogLevel)); value != "" {
		allowed := map[string]bool{"debug": true, "info": true, "warn": true, "warning": true, "error": true, "prod": true, "production": true, "prodj": true, "prodjson": true, "productionjson": true, "dev": true, "development": true}
		if !allowed[value] {
			issues = append(issues, ValidationIssue{Path: "log_level", Message: "unsupported log level"})
		}
	}
	if conf.Proxy.Timeout < 0 {
		issues = append(issues, ValidationIssue{Path: "proxy.timeout", Message: "must not be negative"})
	}
	proxyStrategy := strings.ToLower(strings.TrimSpace(conf.Proxy.Strategy))
	if _, allowed := map[string]bool{"disabled": true, "default": true, "all": true, "force_all": true}[proxyStrategy]; !allowed {
		issues = append(issues, ValidationIssue{Path: "proxy.strategy", Message: "supported values: disabled, default, all, force_all"})
	}
	proxyType := strings.ToLower(strings.TrimSpace(conf.Proxy.Type))
	if proxyType != "" && proxyType != ProxyTypeHTTP && proxyType != ProxyTypeSOCKS5 {
		issues = append(issues, ValidationIssue{Path: "proxy.type", Message: "supported values: http, socks5"})
	}
	if proxyStrategy != PROXY_STRATEGY_DISABLED {
		switch proxyType {
		case ProxyTypeHTTP:
			if strings.TrimSpace(conf.Proxy.HTTPProxy) == "" && strings.TrimSpace(conf.Proxy.HTTPSProxy) == "" {
				issues = append(issues, ValidationIssue{Path: "proxy.http_proxy", Message: "an HTTP or HTTPS proxy URL is required"})
			}
		case ProxyTypeSOCKS5:
			if strings.TrimSpace(conf.Proxy.Socks5Proxy) == "" {
				issues = append(issues, ValidationIssue{Path: "proxy.socks5_proxy", Message: "a SOCKS5 proxy address is required"})
			}
		default:
			issues = append(issues, ValidationIssue{Path: "proxy.type", Message: "proxy type is required when proxying is enabled"})
		}
	}
	for field, value := range map[string]string{
		"proxy.http_proxy": conf.Proxy.HTTPProxy, "proxy.https_proxy": conf.Proxy.HTTPSProxy,
	} {
		if value == "" {
			continue
		}
		parsed, err := url.Parse(value)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			issues = append(issues, ValidationIssue{Path: field, Message: "must be an absolute proxy URL"})
		}
	}
	seenKeys := make(map[string]struct{}, len(conf.APIKeys))
	for index, item := range conf.APIKeys {
		if item.APIKey == "" {
			issues = append(issues, ValidationIssue{Path: fmt.Sprintf("api_keys.%d.api_key", index), Message: "must not be empty"})
			continue
		}
		if _, exists := seenKeys[item.APIKey]; exists {
			issues = append(issues, ValidationIssue{Path: fmt.Sprintf("api_keys.%d.api_key", index), Message: "duplicate access key"})
		}
		seenKeys[item.APIKey] = struct{}{}
	}
	if conf.Translation.Retry < 0 || conf.Translation.Concurrency < 0 {
		issues = append(issues, ValidationIssue{Path: "translation", Message: "retry and concurrency must not be negative"})
	}
	if conf.CircuitBreaker.FailureThreshold < 0 || conf.CircuitBreaker.RecoveryTimeoutSeconds < 0 || conf.CircuitBreaker.HalfOpenMaxRequests < 0 {
		issues = append(issues, ValidationIssue{Path: "circuit_breaker", Message: "thresholds and timeouts must not be negative"})
	}
	seenServiceIDs := make(map[string]struct{})
	for serviceName, models := range conf.Services {
		if _, supported := SupportedServiceTypes[serviceName]; !supported {
			issues = append(issues, ValidationIssue{Path: "services." + serviceName, Message: "unsupported service type"})
		}
		for index, model := range models {
			base := fmt.Sprintf("services.%s.%d", serviceName, index)
			if model.ID == "" || !serviceIDPattern.MatchString(model.ID) {
				issues = append(issues, ValidationIssue{Path: base + ".id", Message: "must be a stable identifier using letters, numbers, dot, underscore, or dash"})
			} else if _, exists := seenServiceIDs[model.ID]; exists {
				issues = append(issues, ValidationIssue{Path: base + ".id", Message: "duplicate service id"})
			}
			seenServiceIDs[model.ID] = struct{}{}
			if model.Timeout < 0 || hasNegativeLimit(model.Limit) || hasNegativeLimit(model.EmbeddingLimit) {
				issues = append(issues, ValidationIssue{Path: base + ".limit", Message: "limits and timeouts must not be negative"})
			}
			if model.Enabled && len(model.Models) == 0 && len(model.EmbeddingModels) == 0 {
				if _, hasDefaults := DefaultSupportModelMap[serviceName]; !hasDefaults {
					issues = append(issues, ValidationIssue{Path: base + ".models", Message: "at least one chat or embedding model is required"})
				}
			}
			if model.ServerURL != "" {
				parsed, err := url.Parse(model.ServerURL)
				if err != nil || parsed.Host == "" || !map[string]bool{"http": true, "https": true, "ws": true, "wss": true}[parsed.Scheme] {
					issues = append(issues, ValidationIssue{Path: base + ".server_url", Message: "must be an absolute HTTP(S) or WebSocket URL"})
				}
			}
			for modelIndex, name := range append(append([]string(nil), model.Models...), model.EmbeddingModels...) {
				if strings.TrimSpace(name) == "" {
					issues = append(issues, ValidationIssue{Path: fmt.Sprintf("%s.models.%d", base, modelIndex), Message: "model name must not be empty"})
				}
			}
		}
	}
	sort.SliceStable(issues, func(i, j int) bool { return issues[i].Path < issues[j].Path })
	return issues
}

func hasNegativeLimit(limit Limit) bool {
	return limit.QPS < 0 || limit.QPM < 0 || limit.RPM < 0 || limit.Concurrency < 0 || limit.Timeout < 0
}

func PrepareConfiguration(conf Configuration, configPath string) (*PreparedConfiguration, error) {
	data, err := json.Marshal(conf)
	if err != nil {
		return nil, err
	}
	var immutable Configuration
	if err := json.Unmarshal(data, &immutable); err != nil {
		return nil, err
	}
	normalizeConfiguration(&immutable)
	if issues := validateNormalizedConfiguration(immutable); len(issues) > 0 {
		return nil, fmt.Errorf("configuration validation failed: %s: %s", issues[0].Path, issues[0].Message)
	}
	modelToService, supportModels := createModelToServiceMap(immutable)
	keyMap := make(map[string]APIKeyConfig, len(immutable.APIKeys))
	for _, item := range immutable.APIKeys {
		keyMap[item.APIKey] = item
	}
	multiContent := append([]string(nil), defaultMultiContentModels...)
	multiContent = append(multiContent, immutable.MultiContentModels...)
	return &PreparedConfiguration{snapshot: &runtimeSnapshot{
		configuration:      &immutable,
		configPath:         configPath,
		modelToService:     modelToService,
		supportModels:      supportModels,
		loadBalancing:      immutable.LoadBalancing,
		serverPort:         immutable.ServerPort,
		apiKey:             immutable.APIKey,
		debug:              immutable.Debug,
		logLevel:           immutable.LogLevel,
		globalRedirect:     immutable.ModelRedirect,
		multiContentModels: multiContent,
		proxy:              &immutable.Proxy,
		translation:        &immutable.Translation,
		apiKeys:            keyMap,
	}}, nil
}

func normalizeConfiguration(immutable *Configuration) {
	immutable.LoadBalancing = strings.ToLower(strings.TrimSpace(immutable.LoadBalancing))
	if immutable.CircuitBreaker.FailureThreshold == 0 {
		immutable.CircuitBreaker.FailureThreshold = 5
	}
	if immutable.CircuitBreaker.RecoveryTimeoutSeconds == 0 {
		immutable.CircuitBreaker.RecoveryTimeoutSeconds = 30
	}
	if immutable.CircuitBreaker.HalfOpenMaxRequests == 0 {
		immutable.CircuitBreaker.HalfOpenMaxRequests = 1
	}
	if immutable.LoadBalancing == "" {
		immutable.LoadBalancing = "random"
	}
	immutable.ServerPort = strings.TrimSpace(immutable.ServerPort)
	if immutable.ServerPort == "" {
		immutable.ServerPort = ":9090"
	}
	immutable.LogLevel = strings.ToLower(strings.TrimSpace(immutable.LogLevel))
	if immutable.LogLevel == "" {
		immutable.LogLevel = "info"
	}
	immutable.Proxy.Strategy = strings.ToLower(strings.TrimSpace(immutable.Proxy.Strategy))
	if immutable.Proxy.Strategy == "" {
		immutable.Proxy.Strategy = PROXY_STRATEGY_DISABLED
	}
	immutable.Proxy.Type = strings.ToLower(strings.TrimSpace(immutable.Proxy.Type))
	if immutable.Proxy.Timeout <= 0 {
		immutable.Proxy.Timeout = 30
	}
	for serviceName, models := range immutable.Services {
		for index := range models {
			model := &models[index]
			model.ID = strings.TrimSpace(model.ID)
			if model.ID == "" {
				model.ID = stableServiceID(serviceName, index)
			}
			model.Provider = strings.TrimSpace(model.Provider)
			if model.Provider == "" {
				model.Provider = serviceName
			}
			model.ServerURL = strings.TrimSpace(model.ServerURL)
			model.Models = normalizeStringList(model.Models)
			model.EmbeddingModels = normalizeStringList(model.EmbeddingModels)
		}
		immutable.Services[serviceName] = models
	}
	immutable.MultiContentModels = normalizeStringList(immutable.MultiContentModels)
}

func stableServiceID(serviceName string, index int) string {
	clean := unsafeServiceIDCharacters.ReplaceAllString(strings.TrimSpace(serviceName), "-")
	clean = strings.Trim(clean, "-._")
	if clean == "" {
		clean = "service"
	}
	return fmt.Sprintf("%s-%d", clean, index+1)
}

func normalizeStringList(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func (prepared *PreparedConfiguration) Publish() {
	if prepared == nil || prepared.snapshot == nil {
		return
	}
	publishSnapshot(prepared.snapshot)
}

func (prepared *PreparedConfiguration) Configuration() Configuration {
	if prepared == nil || prepared.snapshot == nil {
		return Configuration{}
	}
	return cloneJSONValue(*prepared.snapshot.configuration)
}

func ApplyConfiguration(conf Configuration, configPath string) error {
	prepared, err := PrepareConfiguration(conf, configPath)
	if err != nil {
		return err
	}
	prepared.Publish()
	return nil
}

func publishSnapshot(snapshot *runtimeSnapshot) {
	activeSnapshot.Store(snapshot)

	// Compatibility aliases are detached copies so legacy callers cannot mutate
	// the active snapshot. New code should use the Current* accessors.
	compatibilityConfig := cloneJSONValue(*snapshot.configuration)
	GSOAConf = &compatibilityConfig
	ConfigFilePath = snapshot.configPath
	ModelToService = cloneJSONValue(snapshot.modelToService)
	SupportModels = cloneStringMap(snapshot.supportModels)
	LoadBalancingStrategy = snapshot.loadBalancing
	ServerPort = snapshot.serverPort
	APIKey = snapshot.apiKey
	Debug = snapshot.debug
	LogLevel = snapshot.logLevel
	GlobalModelRedirect = cloneStringMap(snapshot.globalRedirect)
	SupportMultiContentModels = append([]string(nil), snapshot.multiContentModels...)
	GProxyConf = &compatibilityConfig.Proxy
	GTranslation = &compatibilityConfig.Translation
	apiKeyMap = cloneJSONValue(snapshot.apiKeys)
}

func currentSnapshot() *runtimeSnapshot {
	if snapshot := activeSnapshot.Load(); snapshot != nil {
		return snapshot
	}
	return &runtimeSnapshot{
		configuration:      &Configuration{},
		modelToService:     map[string][]ModelDetails{},
		supportModels:      map[string]string{},
		loadBalancing:      "random",
		serverPort:         ":9090",
		logLevel:           "info",
		globalRedirect:     map[string]string{},
		multiContentModels: append([]string(nil), defaultMultiContentModels...),
		proxy:              &ProxyConf{Strategy: PROXY_STRATEGY_DISABLED, Timeout: 30},
		translation:        &Translation{},
		apiKeys:            map[string]APIKeyConfig{},
	}
}

func CurrentConfiguration() *Configuration {
	clone := cloneJSONValue(*currentSnapshot().configuration)
	return &clone
}
func CurrentConfigPath() string { return currentSnapshot().configPath }
func CurrentModelToService() map[string][]ModelDetails {
	return cloneJSONValue(currentSnapshot().modelToService)
}
func CurrentSupportModels() map[string]string {
	return cloneStringMap(currentSnapshot().supportModels)
}
func cloneStringMap(current map[string]string) map[string]string {
	clone := make(map[string]string, len(current))
	for key, value := range current {
		clone[key] = value
	}
	return clone
}
func CurrentLoadBalancing() string { return currentSnapshot().loadBalancing }
func CurrentServerPort() string    { return currentSnapshot().serverPort }
func CurrentAPIKey() string        { return currentSnapshot().apiKey }
func CurrentDebug() bool           { return currentSnapshot().debug }
func CurrentLogLevel() string      { return currentSnapshot().logLevel }
func CurrentProxy() *ProxyConf {
	clone := *currentSnapshot().proxy
	return &clone
}
func CurrentTranslation() *Translation {
	clone := *currentSnapshot().translation
	return &clone
}

func cloneJSONValue[T any](value T) T {
	payload, err := json.Marshal(value)
	if err != nil {
		return value
	}
	var clone T
	if err := json.Unmarshal(payload, &clone); err != nil {
		return value
	}
	return clone
}
