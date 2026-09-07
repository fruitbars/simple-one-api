package apis

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"simple-one-api/pkg/config"
)

func applyModelsTestConfiguration(t *testing.T, services map[string][]config.ServiceModel) {
	t.Helper()
	if err := config.ApplyConfiguration(config.Configuration{Services: services}, "models-handler-test"); err != nil {
		t.Fatalf("apply configuration: %v", err)
	}
	t.Cleanup(func() { _ = config.ApplyConfiguration(config.Configuration{}, "models-handler-test-cleanup") })
}

func TestModelsHandlerReturnsStandardList(t *testing.T) {
	applyModelsTestConfiguration(t, map[string][]config.ServiceModel{
		"openai": {{Provider: "openai", Enabled: true, Models: []string{"model-b", "model-a"}}},
	})
	context, response := modelsTestContext(http.MethodGet, "/v1/models")
	ModelsHandler(context)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var payload struct {
		Object string  `json:"object"`
		Data   []Model `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Object != "list" || len(payload.Data) != 2 || payload.Data[0].ID != "model-a" || payload.Data[1].ID != "model-b" {
		t.Fatalf("unexpected model list: %#v", payload)
	}
	if payload.Data[0].OwnedBy != "openai" || payload.Data[0].Object != "model" {
		t.Fatalf("unexpected model metadata: %#v", payload.Data[0])
	}
}

func TestModelsHandlerReturnsEmptyStandardList(t *testing.T) {
	applyModelsTestConfiguration(t, nil)
	context, response := modelsTestContext(http.MethodGet, "/v1/models")
	ModelsHandler(context)
	if response.Code != http.StatusOK || response.Body.String() == "" {
		t.Fatalf("unexpected response: %d %s", response.Code, response.Body.String())
	}
	var payload struct {
		Object string  `json:"object"`
		Data   []Model `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Object != "list" || len(payload.Data) != 0 {
		t.Fatalf("unexpected empty model list: %#v", payload)
	}
}

func TestRetrieveModelHandlerReturnsRequestedModel(t *testing.T) {
	applyModelsTestConfiguration(t, map[string][]config.ServiceModel{
		"openai": {{Provider: "openai", Enabled: true, Models: []string{"model-a"}}},
	})
	context, response := modelsTestContext(http.MethodGet, "/v1/models/model-a")
	context.Params = gin.Params{{Key: "model", Value: "model-a"}}
	RetrieveModelHandler(context)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var model Model
	if err := json.Unmarshal(response.Body.Bytes(), &model); err != nil {
		t.Fatal(err)
	}
	if model.ID != "model-a" {
		t.Fatalf("id = %q, want model-a", model.ID)
	}
}

func modelsTestContext(method, target string) (*gin.Context, *httptest.ResponseRecorder) {
	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Request = httptest.NewRequest(method, target, nil)
	return context, response
}
