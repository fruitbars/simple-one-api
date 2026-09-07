package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	openaisdk "github.com/sashabaranov/go-openai"
	"simple-one-api/pkg/config"
)

func TestResponsesEndpointNormalizesKnownSuffixes(t *testing.T) {
	cases := map[string]string{
		"":                                  "https://api.openai.com/v1/responses",
		"https://example.test/v1":           "https://example.test/v1/responses",
		"https://example.test/v1/responses": "https://example.test/v1/responses",
		"https://example.test/v1/chat/completions": "https://example.test/v1/responses",
	}
	for input, expected := range cases {
		if got := responsesEndpoint(input); got != expected {
			t.Errorf("responsesEndpoint(%q) = %q, want %q", input, got, expected)
		}
	}
}

func TestResponsesPayloadMapsChatMessagesAndTools(t *testing.T) {
	req := &openaisdk.ChatCompletionRequest{
		Model: "model-a",
		Messages: []openaisdk.ChatCompletionMessage{
			{Role: "user", Content: "hello"},
			{Role: "tool", ToolCallID: "call-1", Content: "done"},
		},
		Tools: []openaisdk.Tool{{Type: openaisdk.ToolTypeFunction, Function: &openaisdk.FunctionDefinition{Name: "lookup", Parameters: map[string]any{"type": "object"}}}},
	}
	payload := responsesPayload(req)
	if payload["model"] != "model-a" {
		t.Fatalf("model = %#v", payload["model"])
	}
	input := payload["input"].([]any)
	if len(input) != 2 || input[1].(map[string]any)["type"] != "function_call_output" {
		t.Fatalf("input = %#v", input)
	}
	tools := payload["tools"].([]any)
	if len(tools) != 1 || tools[0].(map[string]any)["name"] != "lookup" {
		t.Fatalf("tools = %#v", tools)
	}
}

func TestResponsesPayloadMapsMaxReasoningToHigh(t *testing.T) {
	req := &openaisdk.ChatCompletionRequest{Model: "model-a", ReasoningEffort: "max"}
	payload := responsesPayload(req)
	reasoning, ok := payload["reasoning"].(map[string]any)
	if !ok || reasoning["effort"] != "high" {
		t.Fatalf("reasoning = %#v, want high", payload["reasoning"])
	}
}

func TestResponsesToOpenAIResponseMapsTextAndUsage(t *testing.T) {
	body := []byte(`{"id":"resp_1","created_at":12,"output":[{"type":"message","content":[{"type":"output_text","text":"hello"}]}],"usage":{"input_tokens":3,"output_tokens":2,"total_tokens":5}}`)
	response, err := responsesToOpenAIResponse(body, "public-model")
	if err != nil {
		t.Fatal(err)
	}
	if response.Model != "public-model" || response.Choices[0].Message.Content != "hello" || response.Usage.TotalTokens != 5 {
		t.Fatalf("response = %#v", response)
	}
}

func TestStreamResponsesAsChatMapsToolCallsAndUsage(t *testing.T) {
	upstream := strings.Join([]string{
		`data: {"type":"response.created","response":{"id":"resp_1","status":"in_progress"}}`,
		`data: {"type":"response.output_item.added","output_index":1,"item":{"id":"call_1","call_id":"call_1","type":"function_call","name":"get_weather","arguments":""}}`,
		`data: {"type":"response.function_call_arguments.delta","item_id":"call_1","output_index":1,"delta":"{\"city\":"}`,
		`data: {"type":"response.function_call_arguments.delta","item_id":"call_1","output_index":1,"delta":"\"北京\"}"}`,
		`data: {"type":"response.completed","response":{"id":"resp_1","status":"completed","usage":{"input_tokens":164,"output_tokens":40,"total_tokens":204,"output_tokens_details":{"reasoning_tokens":28}}}}`,
	}, "\n\n") + "\n\n"
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	response := &http.Response{Body: io.NopCloser(strings.NewReader(upstream))}
	if err := streamResponsesAsChat(context, response, "public-model"); err != nil {
		t.Fatal(err)
	}
	body := recorder.Body.String()
	for _, expected := range []string{`"id":"call_1"`, `"name":"get_weather"`, `"arguments":"{\"city\":"`, `"finish_reason":"tool_calls"`, `"prompt_tokens":164`, `"reasoning_tokens":28`} {
		if !strings.Contains(body, expected) {
			t.Fatalf("stream does not contain %q:\n%s", expected, body)
		}
	}
}

func TestSendDirectResponsesPreservesPayloadAndStream(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if payload["model"] != "upstream-model" || payload["custom_field"] != "keep" {
			t.Errorf("payload = %#v", payload)
		}
		writer.Header().Set("Content-Type", "text/event-stream")
		_, _ = writer.Write([]byte("event: response.completed\ndata: {}\n\n"))
	}))
	defer upstream.Close()
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	params := &directResponsesParams{
		ctx:          context.Request.Context(),
		modelDetails: &config.ModelDetails{ServiceModel: config.ServiceModel{ServerURL: upstream.URL + "/v1/responses", Timeout: 5}},
		creds:        map[string]interface{}{config.KEYNAME_API_KEY: "secret"},
		mappedModel:  "upstream-model", stream: true,
		body: []byte(`{"model":"public-model","stream":true,"custom_field":"keep"}`),
	}
	if err := sendDirectResponses(context, params); err != nil {
		t.Fatal(err)
	}
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "response.completed") {
		t.Fatalf("response = %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestEstimateResponsesRequestTokensIncludesInputAndOutputReservation(t *testing.T) {
	if got := estimateResponsesRequestTokens([]byte("12345678"), 10); got != 12 {
		t.Fatalf("token estimate = %d, want 12", got)
	}
	if got := estimateResponsesRequestTokens(nil, 0); got != 1 {
		t.Fatalf("empty token estimate = %d, want 1", got)
	}
}

func TestDirectResponsesRetriesNextCredentialAfterUpstream429(t *testing.T) {
	previous := *config.CurrentConfiguration()
	t.Cleanup(func() {
		if err := config.ApplyConfiguration(previous, ""); err != nil {
			t.Errorf("restore configuration: %v", err)
		}
	})
	attempts := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		attempts++
		if request.Header.Get("Authorization") == "Bearer first-key" {
			http.Error(writer, "rate limited", http.StatusTooManyRequests)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"id":"resp_1","status":"completed","output":[]}`))
	}))
	defer upstream.Close()
	conf := config.Configuration{LoadBalancing: "first", Services: map[string][]config.ServiceModel{
		"openai": {{
			ID: "direct-retry-provider", Provider: "openai", Enabled: true,
			UpstreamProtocol: config.UpstreamProtocolResponses, ServerURL: upstream.URL + "/v1/responses",
			Models: []string{"public-model"}, CredentialList: []map[string]interface{}{
				{"id": "first", "api_key": "first-key", "enabled": true},
				{"id": "second", "api_key": "second-key", "enabled": true},
			},
		}},
	}}
	if err := config.ApplyConfiguration(conf, ""); err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	body := []byte(`{"model":"public-model","input":"hello"}`)
	handled, err := tryDirectResponses(context, body, responsesRequest{Model: "public-model"})
	if err != nil || !handled {
		t.Fatalf("handled = %v, error = %v", handled, err)
	}
	if attempts != 2 || recorder.Code != http.StatusOK {
		t.Fatalf("attempts = %d, status = %d, body = %s", attempts, recorder.Code, recorder.Body.String())
	}
}
