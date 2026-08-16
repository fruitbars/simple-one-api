package config

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestInitConfigRejectsInvalidJSON(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "broken.json")
	if err := os.WriteFile(configPath, []byte(`{"services":`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := InitConfig(configPath); err == nil {
		t.Fatal("invalid JSON must return an error")
	}
}

func TestInitConfigUsesBuiltInDefaultsWhenDefaultFileIsMissing(t *testing.T) {
	previous := *CurrentConfiguration()
	previousPath := CurrentConfigPath()
	t.Cleanup(func() { _ = ApplyConfiguration(previous, previousPath) })

	directory := t.TempDir()
	previousWorkingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(directory); err != nil {
		t.Fatalf("change working directory: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(previousWorkingDirectory) })

	if err := InitConfig("config.json"); err != nil {
		t.Fatalf("init missing default config: %v", err)
	}
	expectedPath := filepath.Join(directory, "config.json")
	if !sameFilesystemPath(t, CurrentConfigPath(), expectedPath) {
		t.Fatalf("config path = %q, want %q", CurrentConfigPath(), expectedPath)
	}
	if got := CurrentServerPort(); got != ":9090" {
		t.Fatalf("server port = %q, want built-in default", got)
	}
	if _, err := os.Stat(filepath.Join(directory, "config.json")); !os.IsNotExist(err) {
		t.Fatalf("missing config startup must not create config.json, stat err=%v", err)
	}
}

func sameFilesystemPath(t *testing.T, first, second string) bool {
	t.Helper()
	firstEval, firstErr := filepath.EvalSymlinks(filepath.Dir(first))
	secondEval, secondErr := filepath.EvalSymlinks(filepath.Dir(second))
	if firstErr != nil || secondErr != nil {
		return filepath.Clean(first) == filepath.Clean(second)
	}
	return filepath.Join(firstEval, filepath.Base(first)) == filepath.Join(secondEval, filepath.Base(second))
}

func TestInitConfigRejectsMissingExplicitConfiguration(t *testing.T) {
	if err := InitConfig(filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Fatal("missing explicit configuration must return an error")
	}
}

func TestBundledSampleConfigurationsRemainValid(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("..", "..", "samples", "*.json"))
	if err != nil {
		t.Fatalf("list sample configurations: %v", err)
	}
	if len(paths) == 0 {
		t.Fatal("no bundled sample configurations found")
	}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			payload, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read sample: %v", err)
			}
			var conf Configuration
			if err := json.Unmarshal(payload, &conf); err != nil {
				t.Fatalf("decode sample: %v", err)
			}
			if issues := ValidateConfiguration(conf); len(issues) > 0 {
				t.Fatalf("sample is no longer valid: %#v", issues)
			}
		})
	}
}

func TestHTTPSProxySelection(t *testing.T) {
	transport, err := getHttpProxyTransport("http://proxy-http.example:8080", "http://proxy-https.example:8443", 30)
	if err != nil {
		t.Fatalf("create proxy transport: %v", err)
	}
	httpRequest, err := http.NewRequest(http.MethodGet, "http://upstream.example/v1/models", nil)
	if err != nil {
		t.Fatalf("create http request: %v", err)
	}
	httpProxy, err := transport.Proxy(httpRequest)
	if err != nil {
		t.Fatalf("http proxy lookup: %v", err)
	}
	if httpProxy.String() != "http://proxy-http.example:8080" {
		t.Fatalf("unexpected http proxy: %s", httpProxy)
	}

	httpsRequest, err := http.NewRequest(http.MethodGet, "https://upstream.example/v1/models", nil)
	if err != nil {
		t.Fatalf("create https request: %v", err)
	}
	httpsProxy, err := transport.Proxy(httpsRequest)
	if err != nil {
		t.Fatalf("https proxy lookup: %v", err)
	}
	if httpsProxy.String() != "http://proxy-https.example:8443" {
		t.Fatalf("unexpected https proxy: %s", httpsProxy)
	}
}

func TestRandomModelWithEmptyPoolReturnsError(t *testing.T) {
	previous := *CurrentConfiguration()
	previousPath := CurrentConfigPath()
	t.Cleanup(func() { _ = ApplyConfiguration(previous, previousPath) })

	if err := ApplyConfiguration(Configuration{Services: map[string][]ServiceModel{}}, "test.json"); err != nil {
		t.Fatalf("apply empty config: %v", err)
	}
	if _, err := GetRandomEnabledModelDetails(); err == nil {
		t.Fatal("empty random model pool must return an error")
	}
	if _, _, err := GetRandomEnabledModelDetailsV1(); err == nil {
		t.Fatal("empty random model pool V1 must return an error")
	}
}

func TestApplyConfigurationNormalizesLoadBalancing(t *testing.T) {
	previous := *CurrentConfiguration()
	previousPath := CurrentConfigPath()
	t.Cleanup(func() { _ = ApplyConfiguration(previous, previousPath) })

	if err := ApplyConfiguration(Configuration{LoadBalancing: " FIRST "}, "test.json"); err != nil {
		t.Fatalf("apply configuration: %v", err)
	}
	if got := CurrentLoadBalancing(); got != "first" {
		t.Fatalf("load balancing was not normalized: %q", got)
	}
}

func TestRuntimeAccessorsReturnCopies(t *testing.T) {
	previous := *CurrentConfiguration()
	previousPath := CurrentConfigPath()
	t.Cleanup(func() { _ = ApplyConfiguration(previous, previousPath) })

	conf := Configuration{
		APIKey: "runtime-secret",
		Proxy:  ProxyConf{HTTPProxy: "http://proxy.example:8080"},
		Services: map[string][]ServiceModel{
			"openai": {{Enabled: true, Models: []string{"model-a"}}},
		},
	}
	if err := ApplyConfiguration(conf, "test.json"); err != nil {
		t.Fatalf("apply configuration: %v", err)
	}

	CurrentConfiguration().APIKey = "mutated"
	delete(CurrentSupportModels(), "model-a")
	delete(CurrentModelToService(), "model-a")
	CurrentProxy().HTTPProxy = "http://mutated.example"
	GSOAConf.APIKey = "legacy-mutated"
	delete(SupportModels, "model-a")
	delete(ModelToService, "model-a")
	GProxyConf.HTTPProxy = "http://legacy-mutated.example"
	if CurrentAPIKey() != "runtime-secret" {
		t.Fatalf("configuration accessor mutated runtime API key: %q", CurrentAPIKey())
	}
	if _, ok := CurrentSupportModels()["model-a"]; !ok {
		t.Fatal("support-model accessor mutated runtime snapshot")
	}
	if _, ok := CurrentModelToService()["model-a"]; !ok {
		t.Fatal("model-service accessor mutated runtime snapshot")
	}
	if CurrentProxy().HTTPProxy != "http://proxy.example:8080" {
		t.Fatalf("proxy accessor mutated runtime snapshot: %q", CurrentProxy().HTTPProxy)
	}
}

func TestPrepareConfigurationGeneratesStableIDsAndNormalizesModels(t *testing.T) {
	conf := Configuration{Services: map[string][]ServiceModel{
		"openai": {{
			Enabled:         true,
			Models:          []string{" model-a ", "", "model-a", "model-b"},
			EmbeddingModels: []string{" embedding-a ", "embedding-a"},
		}},
	}}
	prepared, err := PrepareConfiguration(conf, "test.json")
	if err != nil {
		t.Fatalf("prepare configuration: %v", err)
	}
	normalized := prepared.Configuration()
	service := normalized.Services["openai"][0]
	if service.ID != "openai-1" {
		t.Fatalf("generated service ID = %q, want openai-1", service.ID)
	}
	if service.Provider != "openai" {
		t.Fatalf("normalized provider = %q, want openai", service.Provider)
	}
	if got := strings.Join(service.Models, ","); got != "model-a,model-b" {
		t.Fatalf("normalized models = %q", got)
	}
	if got := strings.Join(service.EmbeddingModels, ","); got != "embedding-a" {
		t.Fatalf("normalized embedding models = %q", got)
	}

	prepared.Publish()
	firstID := CurrentModelToService()["model-a"][0].ServiceID
	if err := ApplyConfiguration(normalized, "test.json"); err != nil {
		t.Fatalf("reapply normalized configuration: %v", err)
	}
	if secondID := CurrentModelToService()["model-a"][0].ServiceID; secondID != firstID {
		t.Fatalf("service ID changed after reapply: %q -> %q", firstID, secondID)
	}
}

func TestExplicitServiceIDsSurviveProviderReordering(t *testing.T) {
	conf := Configuration{Services: map[string][]ServiceModel{
		"openai": {
			{ID: "primary", Enabled: true, Models: []string{"model-a"}},
			{ID: "fallback", Enabled: true, Models: []string{"model-b"}},
		},
	}}
	if err := ApplyConfiguration(conf, "test.json"); err != nil {
		t.Fatalf("apply configuration: %v", err)
	}
	if got := CurrentModelToService()["model-a"][0].ServiceID; got != "primary" {
		t.Fatalf("model-a service ID = %q", got)
	}

	conf.Services["openai"] = []ServiceModel{conf.Services["openai"][1], conf.Services["openai"][0]}
	if err := ApplyConfiguration(conf, "test.json"); err != nil {
		t.Fatalf("apply reordered configuration: %v", err)
	}
	if got := CurrentModelToService()["model-a"][0].ServiceID; got != "primary" {
		t.Fatalf("model-a service ID changed after reorder: %q", got)
	}
	if got := CurrentModelToService()["model-b"][0].ServiceID; got != "fallback" {
		t.Fatalf("model-b service ID changed after reorder: %q", got)
	}
}

func TestValidateConfigurationRejectsInvalidManagementDrafts(t *testing.T) {
	tests := []struct {
		name string
		conf Configuration
		path string
	}{
		{
			name: "duplicate service IDs",
			conf: Configuration{Services: map[string][]ServiceModel{"openai": {
				{ID: "duplicate", Enabled: true, Models: []string{"model-a"}},
				{ID: "duplicate", Enabled: true, Models: []string{"model-b"}},
			}}},
			path: "services.openai.1.id",
		},
		{
			name: "unsupported service",
			conf: Configuration{Services: map[string][]ServiceModel{"unknown": {{Enabled: true, Models: []string{"model-a"}}}}},
			path: "services.unknown",
		},
		{
			name: "retired AgentBuilder service",
			conf: Configuration{Services: map[string][]ServiceModel{"agentbuilder": {{ID: "legacy", Enabled: true, Models: []string{"bot"}}}}},
			path: "services.agentbuilder",
		},
		{
			name: "retired Coze service",
			conf: Configuration{Services: map[string][]ServiceModel{"cozecn": {{ID: "legacy", Enabled: true, Models: []string{"bot"}, ServerURL: "https://api.coze.cn/v3/chat"}}}},
			path: "services.cozecn",
		},
		{
			name: "port outside range",
			conf: Configuration{ServerPort: ":70000"},
			path: "server_port",
		},
		{
			name: "enabled proxy missing type",
			conf: Configuration{Proxy: ProxyConf{Strategy: PROXY_STRATEGY_DEFAULT}},
			path: "proxy.type",
		},
		{
			name: "HTTP proxy missing address",
			conf: Configuration{Proxy: ProxyConf{Strategy: PROXY_STRATEGY_DEFAULT, Type: ProxyTypeHTTP}},
			path: "proxy.http_proxy",
		},
		{
			name: "negative service limit",
			conf: Configuration{Services: map[string][]ServiceModel{"openai": {{Enabled: true, Models: []string{"model-a"}, Limit: Limit{QPS: -1}}}}},
			path: "services.openai.0.limit",
		},
		{
			name: "enabled OpenAI service missing models",
			conf: Configuration{Services: map[string][]ServiceModel{"openai": {{Enabled: true}}}},
			path: "services.openai.0.models",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			issues := ValidateConfiguration(test.conf)
			if !containsValidationPath(issues, test.path) {
				t.Fatalf("validation issues = %#v, want path %q", issues, test.path)
			}
		})
	}
}

func TestValidateConfigurationAllowsProviderDefaultModels(t *testing.T) {
	issues := ValidateConfiguration(Configuration{Services: map[string][]ServiceModel{
		"qianfan": {{Enabled: true}},
	}})
	if containsValidationPath(issues, "services.qianfan.0.models") {
		t.Fatalf("qianfan default models should be accepted: %#v", issues)
	}
}

func TestConfigurationPreservesUnknownJSONFields(t *testing.T) {
	payload := []byte(`{
		"server_port": ":9090",
		"future_top_level": {"enabled": true},
		"proxy": {"strategy": "disabled", "future_proxy_mode": "tunnel"},
		"services": {"openai": [{
			"enabled": true,
			"models": ["model-a"],
			"future_provider_option": ["alpha", "beta"],
			"limit": {"qps": 2, "future_burst": 5}
		}]}
	}`)
	var conf Configuration
	if err := json.Unmarshal(payload, &conf); err != nil {
		t.Fatalf("unmarshal configuration: %v", err)
	}
	prepared, err := PrepareConfiguration(conf, "test.json")
	if err != nil {
		t.Fatalf("prepare configuration: %v", err)
	}
	roundTrip, err := json.Marshal(prepared.Configuration())
	if err != nil {
		t.Fatalf("marshal configuration: %v", err)
	}
	var document map[string]interface{}
	if err := json.Unmarshal(roundTrip, &document); err != nil {
		t.Fatalf("decode round trip: %v", err)
	}
	if _, ok := document["future_top_level"]; !ok {
		t.Fatalf("top-level extension was lost: %s", roundTrip)
	}
	proxy := document["proxy"].(map[string]interface{})
	if proxy["future_proxy_mode"] != "tunnel" {
		t.Fatalf("proxy extension was lost: %#v", proxy)
	}
	service := document["services"].(map[string]interface{})["openai"].([]interface{})[0].(map[string]interface{})
	if _, ok := service["future_provider_option"]; !ok {
		t.Fatalf("service extension was lost: %#v", service)
	}
	limit := service["limit"].(map[string]interface{})
	if limit["future_burst"] != float64(5) {
		t.Fatalf("limit extension was lost: %#v", limit)
	}
}

func TestConfigurationPreservesUnknownYAMLFields(t *testing.T) {
	payload := []byte("server_port: :9090\nfuture_top_level: enabled\nservices:\n  openai:\n    - enabled: true\n      models: [model-a]\n      future_provider_option: keep\n")
	var conf Configuration
	if err := yaml.Unmarshal(payload, &conf); err != nil {
		t.Fatalf("unmarshal YAML configuration: %v", err)
	}
	if conf.Extensions["future_top_level"] != "enabled" {
		t.Fatalf("top-level YAML extension was lost: %#v", conf.Extensions)
	}
	if conf.Services["openai"][0].Extensions["future_provider_option"] != "keep" {
		t.Fatalf("service YAML extension was lost: %#v", conf.Services["openai"][0].Extensions)
	}
}

func containsValidationPath(issues []ValidationIssue, path string) bool {
	for _, issue := range issues {
		if issue.Path == path {
			return true
		}
	}
	return false
}
