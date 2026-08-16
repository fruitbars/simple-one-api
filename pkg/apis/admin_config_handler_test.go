package apis

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"simple-one-api/pkg/config"
)

func TestAdminConfigDraftMasksAndRestoresSecrets(t *testing.T) {
	previous := *config.CurrentConfiguration()
	previousPath := config.CurrentConfigPath()
	t.Cleanup(func() { _ = config.ApplyConfiguration(previous, previousPath) })

	current := config.Configuration{
		APIKey:        "main-secret",
		LoadBalancing: "first",
		Extensions: map[string]interface{}{
			"future_token": "top-level-extension-secret",
		},
		Proxy: config.ProxyConf{
			HTTPProxy:  "http://proxy-user:proxy-pass@proxy.example:8080",
			HTTPSProxy: "http://proxy-user:tls-pass@secure-proxy.example:8443",
		},
		APIKeys: []config.APIKeyConfig{{
			APIKey:          "scoped-secret",
			SupportedModels: map[string][]string{"openai": {"gpt-test"}},
		}},
		Services: map[string][]config.ServiceModel{
			"openai": {{
				Provider: "openai",
				Enabled:  true,
				Models:   []string{"gpt-test"},
				Credentials: map[string]interface{}{
					"api_key":  "provider-secret",
					"nested":   map[string]interface{}{"access_token": "nested-secret"},
					"accounts": []interface{}{map[string]interface{}{"token": "array-secret"}},
				},
				CredentialList: []map[string]interface{}{
					{"authorization": "credential-list-secret"},
				},
				Extensions: map[string]interface{}{
					"future_access_key": "provider-extension-secret",
				},
				Limit: config.Limit{Extensions: map[string]interface{}{
					"burst_token": "limit-extension-secret",
				}},
			}},
		},
	}
	if err := config.ApplyConfiguration(current, "test.json"); err != nil {
		t.Fatalf("apply current config: %v", err)
	}

	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/admin/config/draft", nil)
	AdminConfigDraftHandler(context)
	if response.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", response.Code)
	}

	var draft AdminConfigDraft
	if err := json.Unmarshal(response.Body.Bytes(), &draft); err != nil {
		t.Fatalf("decode draft: %v", err)
	}
	if !isSecretPlaceholder(draft.Config.APIKey) || !isSecretPlaceholder(draft.Config.APIKeys[0].APIKey) {
		t.Fatalf("top-level secrets were not masked: %#v", draft.Config)
	}
	if draft.Config.Proxy.HTTPProxy == current.Proxy.HTTPProxy || draft.Config.Proxy.HTTPSProxy == current.Proxy.HTTPSProxy {
		t.Fatalf("proxy credentials were not masked: %#v", draft.Config.Proxy)
	}
	service := draft.Config.Services["openai"][0]
	if value, ok := service.Credentials["api_key"].(string); !ok || !isSecretPlaceholder(value) {
		t.Fatalf("provider secret was not masked: %#v", service.Credentials)
	}
	nested := service.Credentials["nested"].(map[string]interface{})
	if value, ok := nested["access_token"].(string); !ok || !isSecretPlaceholder(value) {
		t.Fatalf("nested secret was not masked: %#v", nested)
	}
	account := service.Credentials["accounts"].([]interface{})[0].(map[string]interface{})
	if value, ok := account["token"].(string); !ok || !isSecretPlaceholder(value) {
		t.Fatalf("array secret was not masked: %#v", account)
	}
	if value, ok := service.CredentialList[0]["authorization"].(string); !ok || !isSecretPlaceholder(value) {
		t.Fatalf("credential-list secret was not masked: %#v", service.CredentialList)
	}
	if value, ok := draft.Config.Extensions["future_token"].(string); !ok || !isSecretPlaceholder(value) {
		t.Fatalf("top-level extension secret was not masked: %#v", draft.Config.Extensions)
	}
	if value, ok := service.Extensions["future_access_key"].(string); !ok || !isSecretPlaceholder(value) {
		t.Fatalf("provider extension secret was not masked: %#v", service.Extensions)
	}
	if value, ok := service.Limit.Extensions["burst_token"].(string); !ok || !isSecretPlaceholder(value) {
		t.Fatalf("limit extension secret was not masked: %#v", service.Limit.Extensions)
	}
	draft.Config.Proxy.HTTPProxy = strings.Replace(draft.Config.Proxy.HTTPProxy, "proxy.example", "proxy-new.example", 1)

	if err := restoreConfigurationSecrets(&draft.Config, &current); err != nil {
		t.Fatalf("restore secrets: %v", err)
	}
	restored := draft.Config.Services["openai"][0]
	if draft.Config.APIKey != "main-secret" || draft.Config.APIKeys[0].APIKey != "scoped-secret" {
		t.Fatalf("top-level secrets were not restored: %#v", draft.Config)
	}
	if draft.Config.Proxy.HTTPProxy != "http://proxy-user:proxy-pass@proxy-new.example:8080" || draft.Config.Proxy.HTTPSProxy != current.Proxy.HTTPSProxy {
		t.Fatalf("proxy credentials were not restored: %#v", draft.Config.Proxy)
	}
	if restored.Credentials["api_key"] != "provider-secret" {
		t.Fatalf("provider secret was not restored: %#v", restored.Credentials)
	}
	restoredNested := restored.Credentials["nested"].(map[string]interface{})
	if restoredNested["access_token"] != "nested-secret" {
		t.Fatalf("nested secret was not restored: %#v", restoredNested)
	}
	restoredAccount := restored.Credentials["accounts"].([]interface{})[0].(map[string]interface{})
	if restoredAccount["token"] != "array-secret" {
		t.Fatalf("array secret was not restored: %#v", restoredAccount)
	}
	if restored.CredentialList[0]["authorization"] != "credential-list-secret" {
		t.Fatalf("credential-list secret was not restored: %#v", restored.CredentialList)
	}
	if draft.Config.Extensions["future_token"] != "top-level-extension-secret" {
		t.Fatalf("top-level extension secret was not restored: %#v", draft.Config.Extensions)
	}
	if restored.Extensions["future_access_key"] != "provider-extension-secret" {
		t.Fatalf("provider extension secret was not restored: %#v", restored.Extensions)
	}
	if restored.Limit.Extensions["burst_token"] != "limit-extension-secret" {
		t.Fatalf("limit extension secret was not restored: %#v", restored.Limit.Extensions)
	}
}

func TestAdminSecretPlaceholdersSurviveModelReordering(t *testing.T) {
	current := config.Configuration{Services: map[string][]config.ServiceModel{
		"openai": {
			{Models: []string{"model-a"}, Credentials: map[string]interface{}{"api_key": "secret-a"}},
			{Models: []string{"model-b"}, Credentials: map[string]interface{}{"api_key": "secret-b"}},
		},
	}}
	draft := cloneConfiguration(current)
	maskConfigurationSecrets(&draft)
	draft.Services["openai"][0], draft.Services["openai"][1] = draft.Services["openai"][1], draft.Services["openai"][0]

	if err := restoreConfigurationSecrets(&draft, &current); err != nil {
		t.Fatalf("restore reordered secrets: %v", err)
	}
	if got := draft.Services["openai"][0].Credentials["api_key"]; got != "secret-b" {
		t.Fatalf("model-b received the wrong secret: %v", got)
	}
	if got := draft.Services["openai"][1].Credentials["api_key"]; got != "secret-a" {
		t.Fatalf("model-a received the wrong secret: %v", got)
	}
}

func TestRestartRequiredFieldsCompareEffectiveDefaults(t *testing.T) {
	previous := config.Configuration{
		ServerPort: ":9090",
		LogLevel:   "info",
		Proxy:      config.ProxyConf{Strategy: "disabled", Timeout: 30},
	}
	if fields := restartRequiredFields(previous, config.Configuration{}); len(fields) != 0 {
		t.Fatalf("equivalent default values must not require restart: %#v", fields)
	}
}
