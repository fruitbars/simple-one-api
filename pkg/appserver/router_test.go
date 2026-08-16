package appserver

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/mylog"
)

func TestEmbeddedWebRoutes(t *testing.T) {
	router := NewRouterWithOptions(Options{EnableWeb: true})
	for _, target := range []string{"/", "/settings"} {
		request := httptest.NewRequest(http.MethodGet, target, nil)
		request.Header.Set("Accept", "text/html")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		if response.Code != http.StatusOK {
			t.Fatalf("%s returned %d", target, response.Code)
		}
		if !strings.Contains(response.Body.String(), `<div id="root"></div>`) {
			t.Fatalf("%s did not serve the embedded app: %q", target, response.Body.String())
		}
	}
}

func TestWebCanBeDisabled(t *testing.T) {
	router := NewRouterWithOptions(Options{EnableWeb: false})
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("unexpected status: %d", response.Code)
	}
}

func TestWildcardCORSDoesNotAllowCredentials(t *testing.T) {
	router := NewRouterWithOptions(Options{})
	request := httptest.NewRequest(http.MethodOptions, "/v1/models", nil)
	request.Header.Set("Origin", "https://example.test")
	request.Header.Set("Access-Control-Request-Method", http.MethodGet)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Header().Get("Access-Control-Allow-Credentials") != "" {
		t.Fatal("wildcard CORS must not enable credentials")
	}
}

func TestAdminRequiresConfiguredBearerKey(t *testing.T) {
	router := NewRouterWithOptions(Options{})
	t.Setenv(adminBootstrapEnvironment, "first-run-secret")

	setTestConfiguration(t, config.Configuration{})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/admin/status", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected bootstrap challenge, got %d", response.Code)
	}
	remoteRequest := httptest.NewRequest(http.MethodGet, "http://admin.example.test/api/admin/status", nil)
	remoteRequest.Header.Set("Authorization", "Bearer first-run-secret")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, remoteRequest)
	if response.Code != http.StatusOK {
		t.Fatalf("expected remote bootstrap token to unlock admin, got %d: %s", response.Code, response.Body.String())
	}
	crossOriginRemoteRequest := httptest.NewRequest(http.MethodGet, "http://admin.example.test/api/admin/status", nil)
	crossOriginRemoteRequest.Header.Set("Authorization", "Bearer first-run-secret")
	crossOriginRemoteRequest.Header.Set("Origin", "https://attacker.example")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, crossOriginRemoteRequest)
	if response.Code != http.StatusForbidden {
		t.Fatalf("expected cross-origin remote bootstrap to be forbidden, got %d", response.Code)
	}
	localRequest := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/api/admin/status", nil)
	localRequest.RemoteAddr = "127.0.0.1:12345"
	response = httptest.NewRecorder()
	router.ServeHTTP(response, localRequest)
	if response.Code != http.StatusOK {
		t.Fatalf("expected local bootstrap admin, got %d", response.Code)
	}
	crossOriginRequest := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/api/admin/status", nil)
	crossOriginRequest.RemoteAddr = "127.0.0.1:12345"
	crossOriginRequest.Header.Set("Origin", "https://attacker.example")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, crossOriginRequest)
	if response.Code != http.StatusForbidden {
		t.Fatalf("expected cross-origin bootstrap admin to be forbidden, got %d", response.Code)
	}

	setTestConfiguration(t, config.Configuration{APIKey: "admin-secret"})
	request := httptest.NewRequest(http.MethodGet, "/api/admin/status", nil)
	request.Header.Set("Authorization", "Bearer wrong")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized, got %d", response.Code)
	}
}

func TestTrustedLocalAdminBootstrapOnlyAppliesWhenNoAPIKeyIsConfigured(t *testing.T) {
	setTestConfiguration(t, config.Configuration{})
	router := NewRouterWithOptions(Options{TrustedLocalAdminBootstrap: true})
	remoteRequest := httptest.NewRequest(http.MethodGet, "http://desktop.local/api/admin/status", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, remoteRequest)
	if response.Code != http.StatusOK {
		t.Fatalf("expected trusted first-run admin, got %d: %s", response.Code, response.Body.String())
	}

	setTestConfiguration(t, config.Configuration{APIKey: "admin-secret"})
	remoteRequest = httptest.NewRequest(http.MethodGet, "http://desktop.local/api/admin/status", nil)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, remoteRequest)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected configured admin key to still be required, got %d", response.Code)
	}
}

func TestModelsRequireConfiguredBearerKey(t *testing.T) {
	setTestConfiguration(t, config.Configuration{
		APIKey: "client-secret",
		Services: map[string][]config.ServiceModel{
			"openai": {{Provider: "openai", Enabled: true, Models: []string{"test-model"}}},
		},
	})
	router := NewRouterWithOptions(Options{})

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/models", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized, got %d", response.Code)
	}

	request := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	request.Header.Set("Authorization", "Bearer client-secret")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected authorized request, got %d", response.Code)
	}
}

func TestCompatibilityRoutesAcceptAPIKeyHeader(t *testing.T) {
	setTestConfiguration(t, config.Configuration{APIKey: "client-secret"})
	router := NewRouterWithOptions(Options{})

	for _, target := range []string{"/v1/responses", "/v1/messages"} {
		request := httptest.NewRequest(http.MethodPost, target, strings.NewReader("{"))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("x-api-key", "client-secret")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("%s returned %d, want request parsing status 400", target, response.Code)
		}
	}
}

func TestCompatibilityRoutesBridgeOpenAIProvider(t *testing.T) {
	previousLogger := mylog.Logger
	mylog.Logger = zap.NewNop()
	t.Cleanup(func() { mylog.Logger = previousLogger })
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if !strings.HasSuffix(request.URL.Path, "/chat/completions") {
			http.NotFound(w, request)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"chatcmpl-1","object":"chat.completion","created":1,"model":"model-a","choices":[{"index":0,"message":{"role":"assistant","content":"hello"},"finish_reason":"stop"}],"usage":{"prompt_tokens":2,"completion_tokens":1,"total_tokens":3}}`))
	}))
	defer upstream.Close()

	setTestConfiguration(t, config.Configuration{Services: map[string][]config.ServiceModel{
		"openai": {{ID: "compat", Provider: "openai", Enabled: true, Models: []string{"model-a"}, ServerURL: upstream.URL + "/v1", Credentials: map[string]interface{}{"api_key": "upstream-key"}}},
	}})
	router := NewRouterWithOptions(Options{})

	tests := []struct {
		path string
		body string
		want string
	}{
		{path: "/v1/responses", body: `{"model":"model-a","input":"hi"}`, want: "response"},
		{path: "/v1/messages", body: `{"model":"model-a","max_tokens":32,"messages":[{"role":"user","content":"hi"}]}`, want: "message"},
	}
	for _, test := range tests {
		request := httptest.NewRequest(http.MethodPost, test.path, strings.NewReader(test.body))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("%s returned %d: %s", test.path, response.Code, response.Body.String())
		}
		var payload map[string]any
		if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		if payload["object"] != test.want && payload["type"] != test.want {
			t.Fatalf("%s payload = %#v", test.path, payload)
		}
	}
}

func TestCompatibilityRequestBodyLimit(t *testing.T) {
	setTestConfiguration(t, config.Configuration{})
	router := NewRouterWithOptions(Options{})
	body := `{"model":"model-a","input":"` + strings.Repeat("x", int(maxAPIRequestBodyBytes)) + `"}`
	request := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413; body=%s", response.Code, response.Body.String())
	}
}

func TestAnthropicCompatibilityUsesNativeErrorShape(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"upstream busy"}}`))
	}))
	defer upstream.Close()
	setTestConfiguration(t, config.Configuration{Services: map[string][]config.ServiceModel{
		"openai": {{ID: "compat-error", Provider: "openai", Enabled: true, Models: []string{"model-a"}, ServerURL: upstream.URL + "/v1", Credentials: map[string]interface{}{"api_key": "unused"}}},
	}})
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"model":"model-a","max_tokens":32,"messages":[{"role":"user","content":"hi"}]}`))
	request.Header.Set("Content-Type", "application/json")
	NewRouterWithOptions(Options{}).ServeHTTP(response, request)
	if response.Code < 400 {
		t.Fatalf("status = %d, want error", response.Code)
	}
	var payload struct {
		Type  string `json:"type"`
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Type != "error" || payload.Error.Type == "" || !strings.Contains(payload.Error.Message, "upstream busy") {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestResponsesCompatibilityStreamsBeforeUpstreamCompletes(t *testing.T) {
	release := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("upstream response writer cannot flush")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"id\":\"chat-1\",\"model\":\"model-a\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"hello\"}}]}\n\n"))
		flusher.Flush()
		select {
		case <-release:
		case <-request.Context().Done():
			return
		}
		_, _ = w.Write([]byte("data: {\"id\":\"chat-1\",\"model\":\"model-a\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n"))
		flusher.Flush()
	}))
	defer upstream.Close()
	setTestConfiguration(t, config.Configuration{Services: map[string][]config.ServiceModel{
		"openai": {{ID: "compat-stream", Provider: "openai", Enabled: true, Models: []string{"model-a"}, ServerURL: upstream.URL + "/v1", Credentials: map[string]interface{}{"api_key": "unused"}}},
	}})
	gateway := httptest.NewServer(NewRouterWithOptions(Options{}))
	defer gateway.Close()
	request, _ := http.NewRequest(http.MethodPost, gateway.URL+"/v1/responses", strings.NewReader(`{"model":"model-a","input":"hi","stream":true}`))
	request.Header.Set("Content-Type", "application/json")
	response, err := gateway.Client().Do(request)
	if err != nil {
		close(release)
		t.Fatal(err)
	}
	defer response.Body.Close()
	reader := bufio.NewReader(response.Body)
	var received strings.Builder
	for !strings.Contains(received.String(), "response.output_text.delta") || !strings.Contains(received.String(), "hello") {
		line, readErr := reader.ReadString('\n')
		if readErr != nil {
			close(release)
			t.Fatalf("read first stream event: %v", readErr)
		}
		received.WriteString(line)
	}
	close(release)
	rest, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(rest), "response.completed") {
		t.Fatalf("terminal events = %s", rest)
	}
}

func TestCompatibilityStreamCancellationReachesUpstream(t *testing.T) {
	canceled := make(chan struct{}, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"id\":\"chat-cancel\",\"model\":\"model-a\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"ready\"}}]}\n\n"))
		w.(http.Flusher).Flush()
		<-request.Context().Done()
		canceled <- struct{}{}
	}))
	defer upstream.Close()
	setTestConfiguration(t, config.Configuration{Services: map[string][]config.ServiceModel{
		"openai": {{ID: "compat-cancel", Provider: "openai", Enabled: true, Models: []string{"model-a"}, ServerURL: upstream.URL + "/v1", Credentials: map[string]interface{}{"api_key": "unused"}}},
	}})
	gateway := httptest.NewServer(NewRouterWithOptions(Options{}))
	defer gateway.Close()
	requestContext, cancel := context.WithCancel(context.Background())
	request, _ := http.NewRequestWithContext(requestContext, http.MethodPost, gateway.URL+"/v1/responses", strings.NewReader(`{"model":"model-a","input":"hi","stream":true}`))
	request.Header.Set("Content-Type", "application/json")
	response, err := gateway.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	reader := bufio.NewReader(response.Body)
	for {
		line, readErr := reader.ReadString('\n')
		if readErr != nil {
			t.Fatal(readErr)
		}
		if strings.Contains(line, "ready") {
			break
		}
	}
	cancel()
	_ = response.Body.Close()
	select {
	case <-canceled:
	case <-time.After(2 * time.Second):
		t.Fatal("upstream request was not canceled")
	}
}

func TestLegacyWebSocketOnlyMountedWithWeb(t *testing.T) {
	setTestConfiguration(t, config.Configuration{APIKey: "client-secret"})

	disabled := httptest.NewRecorder()
	NewRouterWithOptions(Options{EnableWeb: false}).ServeHTTP(
		disabled,
		httptest.NewRequest(http.MethodGet, "/multimodelcall", nil),
	)
	if disabled.Code != http.StatusNotFound {
		t.Fatalf("expected disabled legacy WebSocket route, got %d", disabled.Code)
	}

	enabled := httptest.NewRecorder()
	NewRouterWithOptions(Options{EnableWeb: true}).ServeHTTP(
		enabled,
		httptest.NewRequest(http.MethodGet, "/multimodelcall", nil),
	)
	if enabled.Code != http.StatusUnauthorized {
		t.Fatalf("expected authenticated legacy WebSocket route, got %d", enabled.Code)
	}
}

func setTestConfiguration(t *testing.T, conf config.Configuration) {
	t.Helper()
	previousLogger := mylog.Logger
	if mylog.Logger == nil {
		mylog.Logger = zap.NewNop()
	}
	previous := *config.CurrentConfiguration()
	previousPath := config.CurrentConfigPath()
	t.Cleanup(func() {
		_ = config.ApplyConfiguration(previous, previousPath)
		mylog.Logger = previousLogger
	})
	if err := config.ApplyConfiguration(conf, "test.json"); err != nil {
		t.Fatalf("apply test configuration: %v", err)
	}
}
