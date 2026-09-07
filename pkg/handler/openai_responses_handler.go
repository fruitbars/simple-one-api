package handler

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	openaisdk "github.com/sashabaranov/go-openai"
	"simple-one-api/pkg/config"
	myopenai "simple-one-api/pkg/openai"
	"simple-one-api/pkg/utils"
)

// OpenAI2ResponsesHandler sends the normalized chat request to an upstream
// OpenAI Responses endpoint and converts its response back to chat format.
func OpenAI2ResponsesHandler(c *gin.Context, params *OAIRequestParam) error {
	apiKey, _ := utils.GetStringFromMap(params.creds, config.KEYNAME_API_KEY)
	endpoint := responsesEndpoint(params.modelDetails.ServerURL)
	payload := responsesPayload(params.chatCompletionReq)
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(params.ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	if params.chatCompletionReq.Stream {
		req.Header.Set("Accept", "text/event-stream")
	}
	client := utils.NewHTTPClient(params.httpTransport, responseTimeout(params.modelDetails.Timeout))
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errorBody, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
		return utils.NewHTTPStatusError(resp.StatusCode, resp.Status, strings.TrimSpace(string(errorBody)))
	}
	if params.chatCompletionReq.Stream {
		return streamResponsesAsChat(c, resp, params.ClientModel)
	}
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	converted, err := responsesToOpenAIResponse(responseBody, params.ClientModel)
	if err != nil {
		return err
	}
	c.JSON(http.StatusOK, converted)
	return nil
}

func responseTimeout(seconds int) time.Duration {
	if seconds <= 0 {
		seconds = 120
	}
	return time.Duration(seconds) * time.Second
}

func responsesEndpoint(serverURL string) string {
	serverURL = strings.TrimRight(strings.TrimSpace(serverURL), "/")
	if serverURL == "" {
		return "https://api.openai.com/v1/responses"
	}
	if strings.HasSuffix(serverURL, "/responses") {
		return serverURL
	}
	if strings.HasSuffix(serverURL, "/chat/completions") {
		return strings.TrimSuffix(serverURL, "/chat/completions") + "/responses"
	}
	return serverURL + "/responses"
}

func responsesPayload(req *openaisdk.ChatCompletionRequest) map[string]any {
	input := make([]any, 0, len(req.Messages))
	for _, message := range req.Messages {
		role := message.Role
		if role == "tool" {
			input = append(input, map[string]any{"type": "function_call_output", "call_id": message.ToolCallID, "output": message.Content})
			continue
		}
		if len(message.ToolCalls) > 0 {
			for _, call := range message.ToolCalls {
				input = append(input, map[string]any{"type": "function_call", "call_id": call.ID, "name": call.Function.Name, "arguments": call.Function.Arguments})
			}
		}
		content := make([]any, 0, len(message.MultiContent)+1)
		if message.Content != "" {
			content = append(content, map[string]any{"type": "input_text", "text": message.Content})
		}
		for _, part := range message.MultiContent {
			if part.Type == openaisdk.ChatMessagePartTypeText {
				content = append(content, map[string]any{"type": "input_text", "text": part.Text})
			} else if part.ImageURL != nil {
				content = append(content, map[string]any{"type": "input_image", "image_url": part.ImageURL.URL})
			}
		}
		if len(content) == 0 {
			content = append(content, map[string]any{"type": "input_text", "text": ""})
		}
		input = append(input, map[string]any{"type": "message", "role": role, "content": content})
	}
	payload := map[string]any{"model": req.Model, "input": input, "stream": req.Stream}
	maxOutputTokens := req.MaxCompletionTokens
	if maxOutputTokens <= 0 {
		maxOutputTokens = req.MaxTokens
	}
	if maxOutputTokens > 0 {
		payload["max_output_tokens"] = maxOutputTokens
	}
	if req.Temperature != 0 {
		payload["temperature"] = req.Temperature
	}
	if req.TopP != 0 {
		payload["top_p"] = req.TopP
	}
	if len(req.Tools) > 0 {
		tools := make([]any, 0, len(req.Tools))
		for _, tool := range req.Tools {
			if tool.Function == nil {
				continue
			}
			tools = append(tools, map[string]any{"type": "function", "name": tool.Function.Name, "description": tool.Function.Description, "parameters": tool.Function.Parameters, "strict": tool.Function.Strict})
		}
		payload["tools"] = tools
	}
	if req.ToolChoice != nil {
		payload["tool_choice"] = req.ToolChoice
	}
	if req.ReasoningEffort != "" {
		if effort, err := normalizeReasoningEffort(req.ReasoningEffort); err == nil {
			payload["reasoning"] = map[string]any{"effort": effort}
		} else {
			// Preserve the existing request path's error handling for unknown
			// values while avoiding provider-specific aliases such as "max".
			payload["reasoning"] = map[string]any{"effort": req.ReasoningEffort}
		}
	}
	return payload
}

func responsesToOpenAIResponse(body []byte, clientModel string) (*myopenai.OpenAIResponse, error) {
	var payload struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Created int64  `json:"created_at"`
		Output  []struct {
			Type    string `json:"type"`
			Role    string `json:"role"`
			Status  string `json:"status"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
			CallID    string `json:"call_id"`
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		} `json:"output"`
		Usage *struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
			TotalTokens  int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("invalid Responses response: %w", err)
	}
	message := myopenai.ResponseMessage{Role: "assistant"}
	for _, item := range payload.Output {
		if item.Type == "message" {
			for _, content := range item.Content {
				if content.Type == "output_text" || content.Type == "text" {
					message.Content += content.Text
				}
			}
		} else if item.Type == "function_call" {
			message.ToolCalls = append(message.ToolCalls, myopenai.ToolCall{ID: item.CallID, Type: myopenai.ToolType("function"), Function: myopenai.FunctionCall{Name: item.Name, Arguments: item.Arguments}})
		}
	}
	if payload.ID == "" {
		payload.ID = fmt.Sprintf("resp_%d", time.Now().UnixNano())
	}
	result := &myopenai.OpenAIResponse{ID: payload.ID, Object: "chat.completion", Created: payload.Created, Model: clientModel, Choices: []myopenai.Choice{{Index: 0, Message: message, FinishReason: "stop"}}}
	if payload.Usage != nil {
		result.Usage = &myopenai.Usage{PromptTokens: payload.Usage.InputTokens, CompletionTokens: payload.Usage.OutputTokens, TotalTokens: payload.Usage.TotalTokens}
	}
	return result, nil
}

func streamResponsesAsChat(c *gin.Context, resp *http.Response, clientModel string) error {
	utils.SetEventStreamHeaders(c)
	id := "chatcmpl_" + fmt.Sprint(time.Now().UnixNano())
	created := time.Now().Unix()
	toolIndexes := make(map[string]int)
	nextToolIndex := 0
	hasTools := false
	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if strings.HasPrefix(line, "data:") {
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if data == "[DONE]" || data == "" {
				continue
			}
			var event responsesUpstreamStreamEvent
			if json.Unmarshal([]byte(data), &event) == nil {
				if event.Response != nil {
					if event.Response.ID != "" {
						id = event.Response.ID
					}
				}
				switch event.Type {
				case "response.output_text.delta":
					if event.Delta != "" {
						if writeErr := writeResponsesChatChunk(c, id, clientModel, created, map[string]any{"role": "assistant", "content": event.Delta}, nil, nil); writeErr != nil {
							return writeErr
						}
					}
				case "response.output_item.added":
					if event.Item.Type == "function_call" {
						hasTools = true
						key := responsesToolEventKey(event.Item.ID, event.Item.CallID, event.OutputIndex)
						index, exists := toolIndexes[key]
						if !exists {
							index = nextToolIndex
							nextToolIndex++
							toolIndexes[key] = index
						}
						callID := event.Item.CallID
						if callID == "" {
							callID = event.Item.ID
						}
						delta := map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{
							"index": index, "id": callID, "type": "function",
							"function": map[string]any{"name": event.Item.Name, "arguments": event.Item.Arguments},
						}}}
						if writeErr := writeResponsesChatChunk(c, id, clientModel, created, delta, nil, nil); writeErr != nil {
							return writeErr
						}
					}
				case "response.function_call_arguments.delta":
					hasTools = true
					key := responsesToolEventKey(event.ItemID, "", event.OutputIndex)
					index, exists := toolIndexes[key]
					if !exists {
						index = nextToolIndex
						nextToolIndex++
						toolIndexes[key] = index
					}
					delta := map[string]any{"tool_calls": []any{map[string]any{
						"index": index, "function": map[string]any{"arguments": event.Delta},
					}}}
					if writeErr := writeResponsesChatChunk(c, id, clientModel, created, delta, nil, nil); writeErr != nil {
						return writeErr
					}
				case "response.completed", "response.incomplete":
					finishReason := "stop"
					if hasTools {
						finishReason = "tool_calls"
					} else if event.Type == "response.incomplete" || event.Response != nil && event.Response.Status == "incomplete" {
						finishReason = "length"
					}
					var usage any
					if event.Response != nil && event.Response.Usage != nil {
						usage = map[string]any{
							"prompt_tokens": event.Response.Usage.InputTokens, "completion_tokens": event.Response.Usage.OutputTokens,
							"total_tokens":              event.Response.Usage.TotalTokens,
							"completion_tokens_details": map[string]any{"reasoning_tokens": event.Response.Usage.OutputTokensDetails.ReasoningTokens},
						}
					}
					if writeErr := writeResponsesChatChunk(c, id, clientModel, created, map[string]any{}, finishReason, usage); writeErr != nil {
						return writeErr
					}
				case "error", "response.failed":
					message := event.Error.Message
					if message == "" && event.Response != nil {
						message = event.Response.Error.Message
					}
					if message == "" {
						message = "upstream Responses stream failed"
					}
					return errors.New(message)
				}
			}
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

type responsesUpstreamStreamEvent struct {
	Type        string `json:"type"`
	Delta       string `json:"delta"`
	ItemID      string `json:"item_id"`
	OutputIndex int    `json:"output_index"`
	Item        struct {
		Type      string `json:"type"`
		ID        string `json:"id"`
		CallID    string `json:"call_id"`
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"item"`
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
	Response *struct {
		ID     string `json:"id"`
		Status string `json:"status"`
		Error  struct {
			Message string `json:"message"`
		} `json:"error"`
		Usage *struct {
			InputTokens         int `json:"input_tokens"`
			OutputTokens        int `json:"output_tokens"`
			TotalTokens         int `json:"total_tokens"`
			OutputTokensDetails struct {
				ReasoningTokens int `json:"reasoning_tokens"`
			} `json:"output_tokens_details"`
		} `json:"usage"`
	} `json:"response"`
}

func responsesToolEventKey(id, callID string, outputIndex int) string {
	if id != "" {
		return id
	}
	if callID != "" {
		return callID
	}
	return fmt.Sprintf("output:%d", outputIndex)
}

func writeResponsesChatChunk(c *gin.Context, id, model string, created int64, delta map[string]any, finishReason any, usage any) error {
	chunk := map[string]any{
		"id": id, "object": "chat.completion.chunk", "created": created, "model": model,
		"choices": []any{map[string]any{"index": 0, "delta": delta, "finish_reason": finishReason}},
	}
	if usage != nil {
		chunk["usage"] = usage
	}
	encoded, err := json.Marshal(chunk)
	if err != nil {
		return err
	}
	if _, err := c.Writer.WriteString("data: " + string(encoded) + "\n\n"); err != nil {
		return err
	}
	if flusher, ok := c.Writer.(http.Flusher); ok {
		flusher.Flush()
	}
	return nil
}
