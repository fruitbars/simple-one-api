package apis

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/mycommon"
)

func TestAdminConfigRedactsProxyCredentials(t *testing.T) {
	previous := *config.CurrentConfiguration()
	t.Cleanup(func() { _ = config.ApplyConfiguration(previous, "test.json") })
	conf := config.Configuration{
		Proxy: config.ProxyConf{
			HTTPProxy:  "http://alice:secret@proxy.example:8080",
			HTTPSProxy: "https://proxy.example:8443",
		},
		Services:    map[string][]config.ServiceModel{},
		Translation: config.Translation{Extensions: map[string]interface{}{"future_token": "translation-secret"}},
		ParamsRange: map[string]config.ModelParams{"default": {
			Extensions: map[string]interface{}{"future_secret": "params-secret"},
		}},
	}
	conf.Services["openai"] = []config.ServiceModel{{
		Enabled: true, Models: []string{"model-a"}, Limit: config.Limit{Extensions: map[string]interface{}{"access_key": "limit-secret"}},
	}}
	if err := config.ApplyConfiguration(conf, "test.json"); err != nil {
		t.Fatalf("apply test config: %v", err)
	}

	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/admin/config", nil)
	AdminConfigHandler(context)

	if response.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", response.Code)
	}
	for _, secret := range []string{"alice", "translation-secret", "params-secret", "limit-secret"} {
		if strings.Contains(response.Body.String(), secret) {
			t.Fatalf("configuration secret %q leaked: %s", secret, response.Body.String())
		}
	}
	var payload PublicConfig
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Proxy.HTTPProxy != "http://redacted@proxy.example:8080" {
		t.Fatalf("unexpected redacted proxy: %s", payload.Proxy.HTTPProxy)
	}
	if payload.Proxy.HTTPSProxy != "https://proxy.example:8443" {
		t.Fatalf("credential-free proxy changed: %s", payload.Proxy.HTTPSProxy)
	}
}

func TestAdminCapacityReportsRuntimeUsageWithoutSecrets(t *testing.T) {
	previous := *config.CurrentConfiguration()
	t.Cleanup(func() { _ = config.ApplyConfiguration(previous, "test.json") })
	credential := map[string]interface{}{
		"id": "capacity-key", "name": "Capacity Key", "enabled": true,
		"api_key": "upstream-secret", "limit": map[string]interface{}{"tpm": 1000},
	}
	conf := config.Configuration{
		APIKey: "admin-secret",
		Services: map[string][]config.ServiceModel{"openai": {{
			ID: "capacity-provider", Provider: "openai", Enabled: true,
			Models: []string{"model-a"}, CredentialList: []map[string]interface{}{credential},
		}}},
	}
	if err := config.ApplyConfiguration(conf, "test.json"); err != nil {
		t.Fatalf("apply test config: %v", err)
	}
	mycommon.ReserveCredentialCapacity(credential, "capacity-provider:capacity-key", "model-a", 275)

	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/admin/capacity", nil)
	AdminCapacityHandler(context)

	if response.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", response.Code)
	}
	if strings.Contains(response.Body.String(), "upstream-secret") {
		t.Fatalf("capacity response leaked upstream secret: %s", response.Body.String())
	}
	var payload struct {
		Data []CredentialCapacityStatus `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Data) != 1 {
		t.Fatalf("expected one capacity row, got %d", len(payload.Data))
	}
	row := payload.Data[0]
	if row.ReservedTokens != 275 || row.RemainingTokens != 725 || !row.Available {
		t.Fatalf("unexpected capacity row: %+v", row)
	}
}
