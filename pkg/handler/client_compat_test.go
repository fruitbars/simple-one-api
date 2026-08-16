package handler

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	openaisdk "github.com/sashabaranov/go-openai"
	myopenai "simple-one-api/pkg/openai"
)

func TestResponsesToChatMapsToolsAndToolOutputs(t *testing.T) {
	request := responsesRequest{
		Model:        "model-a",
		Instructions: "be concise",
		Input: json.RawMessage(`[
      {"type":"message","role":"user","content":[{"type":"input_text","text":"list files"}]},
      {"type":"function_call_output","call_id":"call_1","output":"README.md"}
    ]`),
		Tools: []responsesTool{{Type: "function", Name: "shell", Description: "run command", Parameters: map[string]any{"type": "object"}}},
	}
	chat, err := responsesToChat(request)
	if err != nil {
		t.Fatal(err)
	}
	if len(chat.Messages) != 3 || chat.Messages[0].Role != "system" || chat.Messages[1].Content != "list files" {
		t.Fatalf("messages = %#v", chat.Messages)
	}
	if chat.Messages[2].Role != "tool" || chat.Messages[2].ToolCallID != "call_1" {
		t.Fatalf("tool output = %#v", chat.Messages[2])
	}
	if len(chat.Tools) != 1 || chat.Tools[0].Function.Name != "shell" {
		t.Fatalf("tools = %#v", chat.Tools)
	}
}

func TestResponsesToChatMapsReasoningAndImages(t *testing.T) {
	request := responsesRequest{
		Model:     "model-a",
		Reasoning: json.RawMessage(`{"effort":"high","summary":"auto"}`),
		Input:     json.RawMessage(`[{"type":"message","role":"user","content":[{"type":"input_text","text":"inspect"},{"type":"input_image","image_url":"https://example.test/image.png"}]}]`),
	}
	chat, err := responsesToChat(request)
	if err != nil {
		t.Fatal(err)
	}
	if chat.ReasoningEffort != "high" || len(chat.Messages) != 1 || len(chat.Messages[0].MultiContent) != 2 {
		t.Fatalf("chat request = %#v", chat)
	}
}

func TestCompatibilityRejectsStatefulOrUnknownInput(t *testing.T) {
	if _, err := responsesToChat(responsesRequest{PreviousResponseID: "resp_1"}); err == nil {
		t.Fatal("previous_response_id must not be silently ignored")
	}
	if _, err := responsesToChat(responsesRequest{Input: json.RawMessage(`[{"type":"computer_call"}]`)}); err == nil {
		t.Fatal("unknown input type must not be silently ignored")
	}
}

func TestAnthropicToChatMapsBase64ImageAndToolChoice(t *testing.T) {
	request := anthropicRequest{
		Model: "model-a", MaxTokens: 32,
		Messages:   []anthropicMessage{{Role: "user", Content: json.RawMessage(`[{"type":"image","source":{"type":"base64","media_type":"image/png","data":"AAAA"}}]`)}},
		ToolChoice: map[string]any{"type": "tool", "name": "shell"},
	}
	chat, err := anthropicToChat(request)
	if err != nil {
		t.Fatal(err)
	}
	if len(chat.Messages) != 1 || len(chat.Messages[0].MultiContent) != 1 || chat.Messages[0].MultiContent[0].ImageURL.URL != "data:image/png;base64,AAAA" {
		t.Fatalf("messages = %#v", chat.Messages)
	}
	if chat.ToolChoice == nil {
		t.Fatal("tool choice was not mapped")
	}
}

func TestCompatibilityResponsesPreserveToolCalls(t *testing.T) {
	response := &myopenai.OpenAIResponse{
		ID: "chat-1", Model: "model-a",
		Choices: []myopenai.Choice{
			{Message: myopenai.ResponseMessage{
				ToolCalls: []myopenai.ToolCall{
					{ID: "call-1", Type: "function", Function: myopenai.FunctionCall{Name: "shell", Arguments: `{"cmd":"pwd"}`}},
				},
			}},
		},
	}
	responsesPayload := buildResponsesResponse(response)
	output := responsesPayload["output"].([]any)
	if len(output) != 1 || output[0].(map[string]any)["type"] != "function_call" {
		t.Fatalf("Responses output = %#v", output)
	}
	anthropicPayload := buildAnthropicResponse(response)
	content := anthropicPayload["content"].([]any)
	if len(content) != 1 || content[0].(map[string]any)["type"] != "tool_use" || anthropicPayload["stop_reason"] != "tool_use" {
		t.Fatalf("Anthropic content = %#v", anthropicPayload)
	}
}

func TestCompatibilityStreamsEmitTerminalEvents(t *testing.T) {
	response := &myopenai.OpenAIResponse{ID: "chat-1", Model: "model-a", Choices: []myopenai.Choice{{Message: myopenai.ResponseMessage{Content: "done"}}}}

	responsesRecorder := httptest.NewRecorder()
	responsesContext, _ := gin.CreateTestContext(responsesRecorder)
	writeResponsesStream(responsesContext, buildResponsesResponse(response))
	if body := responsesRecorder.Body.String(); !strings.Contains(body, "event: response.output_text.delta") || !strings.Contains(body, "event: response.completed") {
		t.Fatalf("Responses stream = %s", body)
	}

	anthropicRecorder := httptest.NewRecorder()
	anthropicContext, _ := gin.CreateTestContext(anthropicRecorder)
	writeAnthropicStream(anthropicContext, buildAnthropicResponse(response))
	if body := anthropicRecorder.Body.String(); !strings.Contains(body, "event: content_block_delta") || !strings.Contains(body, "event: message_stop") {
		t.Fatalf("Anthropic stream = %s", body)
	}
}

func TestAnthropicCompatibilityEmitterStreamsToolArguments(t *testing.T) {
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	emitter := newCompatibilityStreamEmitter(context, compatibilityProtocolAnthropic, "model-a")
	toolIndex := 0
	emitter.emit(openaisdk.ChatCompletionStreamResponse{
		ID: "chat-tool", Model: "model-a",
		Choices: []openaisdk.ChatCompletionStreamChoice{{Delta: openaisdk.ChatCompletionStreamChoiceDelta{ToolCalls: []openaisdk.ToolCall{{
			Index: &toolIndex, ID: "call-1", Type: openaisdk.ToolTypeFunction, Function: openaisdk.FunctionCall{Name: "shell", Arguments: `{"cmd":`},
		}}}}},
	})
	emitter.emit(openaisdk.ChatCompletionStreamResponse{
		ID: "chat-tool", Model: "model-a",
		Choices: []openaisdk.ChatCompletionStreamChoice{{Delta: openaisdk.ChatCompletionStreamChoiceDelta{ToolCalls: []openaisdk.ToolCall{{
			Index: &toolIndex, Function: openaisdk.FunctionCall{Arguments: `"pwd"}`},
		}}}}},
	})
	emitter.finish()
	body := recorder.Body.String()
	for _, expected := range []string{"content_block_start", "input_json_delta", `\"pwd\"`, "content_block_stop", "message_stop"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("Anthropic stream missing %q: %s", expected, body)
		}
	}
}
