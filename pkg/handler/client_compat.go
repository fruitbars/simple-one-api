package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	openaisdk "github.com/sashabaranov/go-openai"
	myopenai "simple-one-api/pkg/openai"
)

type responsesRequest struct {
	Model              string          `json:"model"`
	Input              json.RawMessage `json:"input"`
	Instructions       string          `json:"instructions"`
	Stream             bool            `json:"stream"`
	MaxOutputTokens    int             `json:"max_output_tokens"`
	Temperature        float32         `json:"temperature"`
	Tools              []responsesTool `json:"tools"`
	ToolChoice         any             `json:"tool_choice"`
	ParallelToolCalls  any             `json:"parallel_tool_calls"`
	TopP               float32         `json:"top_p"`
	PreviousResponseID string          `json:"previous_response_id"`
	Reasoning          json.RawMessage `json:"reasoning"`
	Truncation         string          `json:"truncation"`
}

type responsesReasoning struct {
	Effort  string `json:"effort"`
	Summary string `json:"summary"`
}

type responsesTool struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters"`
	Strict      bool   `json:"strict"`
}

type inputItem struct {
	Type      string          `json:"type"`
	Role      string          `json:"role"`
	Content   json.RawMessage `json:"content"`
	CallID    string          `json:"call_id"`
	Name      string          `json:"name"`
	Arguments string          `json:"arguments"`
	Output    any             `json:"output"`
}

type contentBlock struct {
	Type      string                `json:"type"`
	Text      string                `json:"text"`
	Content   any                   `json:"content"`
	ToolUseID string                `json:"tool_use_id"`
	ID        string                `json:"id"`
	Name      string                `json:"name"`
	Input     json.RawMessage       `json:"input"`
	ImageURL  string                `json:"image_url"`
	FileID    string                `json:"file_id"`
	Source    *anthropicImageSource `json:"source"`
}

type anthropicImageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
	URL       string `json:"url"`
}

func ResponsesHandler(c *gin.Context) {
	var request responsesRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		if isRequestTooLarge(err) {
			sendErrorResponse(c, http.StatusRequestEntityTooLarge, "request body too large")
			return
		}
		sendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	chatRequest, err := responsesToChat(request)
	if err != nil {
		sendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	if request.Stream {
		executeCompatibilityStream(c, chatRequest, compatibilityProtocolResponses)
		return
	}
	response, status, body := executeCompatibilityRequest(c, chatRequest)
	if status >= http.StatusBadRequest {
		writeResponsesError(c, status, compatibilityErrorMessage(body, http.StatusText(status)))
		return
	}
	payload := buildResponsesResponse(response)
	c.JSON(http.StatusOK, payload)
}

func responsesToChat(request responsesRequest) (*openaisdk.ChatCompletionRequest, error) {
	if request.PreviousResponseID != "" {
		return nil, errors.New("previous_response_id is not supported because this gateway is stateless; include prior items in input")
	}
	var reasoning responsesReasoning
	if len(request.Reasoning) > 0 && string(request.Reasoning) != "null" {
		if err := json.Unmarshal(request.Reasoning, &reasoning); err != nil {
			return nil, fmt.Errorf("invalid reasoning configuration: %w", err)
		}
		if reasoning.Effort != "" && reasoning.Effort != "low" && reasoning.Effort != "medium" && reasoning.Effort != "high" {
			return nil, fmt.Errorf("unsupported reasoning effort %q", reasoning.Effort)
		}
	}
	messages, err := parseResponsesInput(request.Input)
	if err != nil {
		return nil, err
	}
	if request.Instructions != "" {
		messages = append([]openaisdk.ChatCompletionMessage{{Role: openaisdk.ChatMessageRoleSystem, Content: request.Instructions}}, messages...)
	}
	tools := make([]openaisdk.Tool, 0, len(request.Tools))
	for _, tool := range request.Tools {
		if tool.Type != "function" {
			continue
		}
		tools = append(tools, openaisdk.Tool{Type: openaisdk.ToolTypeFunction, Function: &openaisdk.FunctionDefinition{
			Name: tool.Name, Description: tool.Description, Parameters: tool.Parameters, Strict: tool.Strict,
		}})
	}
	return &openaisdk.ChatCompletionRequest{
		Model: request.Model, Messages: messages, MaxCompletionTokens: request.MaxOutputTokens,
		Temperature: request.Temperature, TopP: request.TopP, Tools: tools, ToolChoice: request.ToolChoice,
		ParallelToolCalls: request.ParallelToolCalls,
		ReasoningEffort:   reasoning.Effort,
	}, nil
}

func parseResponsesInput(raw json.RawMessage) ([]openaisdk.ChatCompletionMessage, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return []openaisdk.ChatCompletionMessage{{Role: openaisdk.ChatMessageRoleUser, Content: text}}, nil
	}
	var items []inputItem
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("input must be a string or an array of input items: %w", err)
	}
	messages := make([]openaisdk.ChatCompletionMessage, 0, len(items))
	for _, item := range items {
		switch item.Type {
		case "function_call_output":
			messages = append(messages, openaisdk.ChatCompletionMessage{Role: openaisdk.ChatMessageRoleTool, ToolCallID: item.CallID, Content: stringifyValue(item.Output)})
		case "function_call":
			messages = append(messages, openaisdk.ChatCompletionMessage{Role: openaisdk.ChatMessageRoleAssistant, ToolCalls: []openaisdk.ToolCall{{
				ID: item.CallID, Type: openaisdk.ToolTypeFunction, Function: openaisdk.FunctionCall{Name: item.Name, Arguments: item.Arguments},
			}}})
		case "message", "":
			content, blocks, err := parseMessageContent(item.Content)
			if err != nil {
				return nil, err
			}
			messages = append(messages, openaisdk.ChatCompletionMessage{Role: item.Role, Content: content, MultiContent: blocks})
		default:
			return nil, fmt.Errorf("unsupported Responses input item type %q", item.Type)
		}
	}
	return messages, nil
}

func parseMessageContent(raw json.RawMessage) (string, []openaisdk.ChatMessagePart, error) {
	var text string
	if len(raw) == 0 {
		return "", nil, nil
	}
	if json.Unmarshal(raw, &text) == nil {
		return text, nil, nil
	}
	var blocks []contentBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return "", nil, fmt.Errorf("invalid message content: %w", err)
	}
	var builder strings.Builder
	parts := make([]openaisdk.ChatMessagePart, 0, len(blocks))
	for _, block := range blocks {
		switch block.Type {
		case "input_text", "output_text", "text":
			builder.WriteString(block.Text)
			parts = append(parts, openaisdk.ChatMessagePart{Type: openaisdk.ChatMessagePartTypeText, Text: block.Text})
		case "input_image", "image":
			imageURL, err := contentBlockImageURL(block)
			if err != nil {
				return "", nil, err
			}
			parts = append(parts, openaisdk.ChatMessagePart{Type: openaisdk.ChatMessagePartTypeImageURL, ImageURL: &openaisdk.ChatMessageImageURL{URL: imageURL}})
		default:
			return "", nil, fmt.Errorf("unsupported content block type %q", block.Type)
		}
	}
	if len(parts) == 0 || len(parts) == 1 && parts[0].Type == openaisdk.ChatMessagePartTypeText {
		return builder.String(), nil, nil
	}
	return "", parts, nil
}

func contentBlockImageURL(block contentBlock) (string, error) {
	if block.FileID != "" {
		return "", errors.New("file_id image inputs are not supported; provide image_url or base64 image data")
	}
	if block.ImageURL != "" {
		return block.ImageURL, nil
	}
	if block.Source == nil {
		return "", errors.New("image content requires image_url or source")
	}
	switch block.Source.Type {
	case "base64":
		if block.Source.MediaType == "" || block.Source.Data == "" {
			return "", errors.New("base64 image source requires media_type and data")
		}
		return "data:" + block.Source.MediaType + ";base64," + block.Source.Data, nil
	case "url":
		if block.Source.URL == "" {
			return "", errors.New("URL image source requires url")
		}
		return block.Source.URL, nil
	default:
		return "", fmt.Errorf("unsupported image source type %q", block.Source.Type)
	}
}

func stringifyValue(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func executeCompatibilityRequest(c *gin.Context, request *openaisdk.ChatCompletionRequest) (*myopenai.OpenAIResponse, int, []byte) {
	request.Stream = false
	writer := newCompatibilityBuffer(c.Writer)
	inner, _ := gin.CreateTestContext(writer)
	inner.Request = c.Request.Clone(c.Request.Context())
	HandleOpenAIRequest(inner, request)
	body := writer.Bytes()
	if writer.Status() >= http.StatusBadRequest {
		return nil, writer.Status(), body
	}
	var response myopenai.OpenAIResponse
	if err := json.Unmarshal(body, &response); err != nil {
		encoded, _ := json.Marshal(map[string]any{"error": "invalid internal response: " + err.Error()})
		return nil, http.StatusBadGateway, encoded
	}
	return &response, http.StatusOK, body
}

func buildResponsesResponse(response *myopenai.OpenAIResponse) map[string]any {
	now := time.Now().Unix()
	responseID := "resp_" + response.ID
	output := make([]any, 0)
	if len(response.Choices) > 0 {
		message := response.Choices[0].Message
		if message.Content != "" {
			output = append(output, map[string]any{
				"id": "msg_" + response.ID, "type": "message", "status": "completed", "role": "assistant",
				"content": []any{map[string]any{"type": "output_text", "text": message.Content, "annotations": []any{}}},
			})
		}
		for _, tool := range message.ToolCalls {
			output = append(output, map[string]any{
				"id": "fc_" + tool.ID, "type": "function_call", "status": "completed", "call_id": tool.ID,
				"name": tool.Function.Name, "arguments": tool.Function.Arguments,
			})
		}
	}
	usage := map[string]any{"input_tokens": 0, "output_tokens": 0, "total_tokens": 0}
	if response.Usage != nil {
		usage = map[string]any{"input_tokens": response.Usage.PromptTokens, "output_tokens": response.Usage.CompletionTokens, "total_tokens": response.Usage.TotalTokens}
	}
	return map[string]any{
		"id": responseID, "object": "response", "created_at": now, "status": "completed", "model": response.Model,
		"output": output, "parallel_tool_calls": true, "tool_choice": "auto", "tools": []any{}, "usage": usage,
		"error": nil, "incomplete_details": nil,
	}
}

func writeSSE(c *gin.Context, event string, payload any) {
	encoded, _ := json.Marshal(payload)
	_, _ = fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, encoded)
	c.Writer.Flush()
}

func writeResponsesStream(c *gin.Context, response map[string]any) {
	c.Header("Content-Type", "text/event-stream")
	created := make(map[string]any, len(response))
	for key, value := range response {
		created[key] = value
	}
	created["status"] = "in_progress"
	created["output"] = []any{}
	created["usage"] = nil
	writeSSE(c, "response.created", map[string]any{"type": "response.created", "response": created, "sequence_number": 0})
	sequence := 1
	for index, item := range response["output"].([]any) {
		writeSSE(c, "response.output_item.added", map[string]any{"type": "response.output_item.added", "output_index": index, "item": item, "sequence_number": sequence})
		sequence++
		itemMap := item.(map[string]any)
		if itemMap["type"] == "message" {
			content := itemMap["content"].([]any)[0].(map[string]any)
			writeSSE(c, "response.content_part.added", map[string]any{"type": "response.content_part.added", "output_index": index, "content_index": 0, "part": map[string]any{"type": "output_text", "text": "", "annotations": []any{}}, "sequence_number": sequence})
			sequence++
			writeSSE(c, "response.output_text.delta", map[string]any{"type": "response.output_text.delta", "output_index": index, "content_index": 0, "delta": content["text"], "sequence_number": sequence})
			sequence++
			writeSSE(c, "response.output_text.done", map[string]any{"type": "response.output_text.done", "output_index": index, "content_index": 0, "text": content["text"], "sequence_number": sequence})
			sequence++
			writeSSE(c, "response.content_part.done", map[string]any{"type": "response.content_part.done", "output_index": index, "content_index": 0, "part": content, "sequence_number": sequence})
			sequence++
		} else if itemMap["type"] == "function_call" {
			writeSSE(c, "response.function_call_arguments.delta", map[string]any{"type": "response.function_call_arguments.delta", "output_index": index, "item_id": itemMap["id"], "delta": itemMap["arguments"], "sequence_number": sequence})
			sequence++
			writeSSE(c, "response.function_call_arguments.done", map[string]any{"type": "response.function_call_arguments.done", "output_index": index, "item_id": itemMap["id"], "arguments": itemMap["arguments"], "sequence_number": sequence})
			sequence++
		}
		writeSSE(c, "response.output_item.done", map[string]any{"type": "response.output_item.done", "output_index": index, "item": item, "sequence_number": sequence})
		sequence++
	}
	writeSSE(c, "response.completed", map[string]any{"type": "response.completed", "response": response, "sequence_number": sequence})
}

type anthropicRequest struct {
	Model         string             `json:"model"`
	System        json.RawMessage    `json:"system"`
	Messages      []anthropicMessage `json:"messages"`
	MaxTokens     int                `json:"max_tokens"`
	Temperature   float32            `json:"temperature"`
	Stream        bool               `json:"stream"`
	Tools         []anthropicTool    `json:"tools"`
	ToolChoice    any                `json:"tool_choice"`
	TopP          float32            `json:"top_p"`
	StopSequences []string           `json:"stop_sequences"`
}

type anthropicMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type anthropicTool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"input_schema"`
}

func AnthropicMessagesHandler(c *gin.Context) {
	var request anthropicRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		if isRequestTooLarge(err) {
			writeAnthropicError(c, http.StatusRequestEntityTooLarge, "request_too_large", "request body too large")
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"type": "error", "error": gin.H{"type": "invalid_request_error", "message": err.Error()}})
		return
	}
	chatRequest, err := anthropicToChat(request)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "error", "error": gin.H{"type": "invalid_request_error", "message": err.Error()}})
		return
	}
	if request.Stream {
		executeCompatibilityStream(c, chatRequest, compatibilityProtocolAnthropic)
		return
	}
	response, status, body := executeCompatibilityRequest(c, chatRequest)
	if status >= http.StatusBadRequest {
		writeAnthropicError(c, status, anthropicErrorType(status), compatibilityErrorMessage(body, http.StatusText(status)))
		return
	}
	payload := buildAnthropicResponse(response)
	c.JSON(http.StatusOK, payload)
}

func anthropicToChat(request anthropicRequest) (*openaisdk.ChatCompletionRequest, error) {
	messages := make([]openaisdk.ChatCompletionMessage, 0, len(request.Messages)+1)
	if len(request.System) > 0 {
		text, _, err := parseMessageContent(request.System)
		if err != nil {
			return nil, err
		}
		if text != "" {
			messages = append(messages, openaisdk.ChatCompletionMessage{Role: openaisdk.ChatMessageRoleSystem, Content: text})
		}
	}
	for _, message := range request.Messages {
		var text string
		if json.Unmarshal(message.Content, &text) == nil {
			messages = append(messages, openaisdk.ChatCompletionMessage{Role: message.Role, Content: text})
			continue
		}
		var blocks []contentBlock
		if err := json.Unmarshal(message.Content, &blocks); err != nil {
			return nil, fmt.Errorf("invalid Anthropic message content: %w", err)
		}
		var builder strings.Builder
		var toolCalls []openaisdk.ToolCall
		for _, block := range blocks {
			switch block.Type {
			case "text":
				builder.WriteString(block.Text)
			case "tool_use":
				toolCalls = append(toolCalls, openaisdk.ToolCall{ID: block.ID, Type: openaisdk.ToolTypeFunction, Function: openaisdk.FunctionCall{Name: block.Name, Arguments: string(block.Input)}})
			case "tool_result":
				messages = append(messages, openaisdk.ChatCompletionMessage{Role: openaisdk.ChatMessageRoleTool, ToolCallID: block.ToolUseID, Content: stringifyValue(block.Content)})
			case "image":
				imageURL, err := contentBlockImageURL(block)
				if err != nil {
					return nil, err
				}
				messages = append(messages, openaisdk.ChatCompletionMessage{Role: message.Role, MultiContent: []openaisdk.ChatMessagePart{{Type: openaisdk.ChatMessagePartTypeImageURL, ImageURL: &openaisdk.ChatMessageImageURL{URL: imageURL}}}})
			default:
				return nil, fmt.Errorf("unsupported Anthropic content block type %q", block.Type)
			}
		}
		if builder.Len() > 0 || len(toolCalls) > 0 {
			messages = append(messages, openaisdk.ChatCompletionMessage{Role: message.Role, Content: builder.String(), ToolCalls: toolCalls})
		}
	}
	tools := make([]openaisdk.Tool, 0, len(request.Tools))
	for _, tool := range request.Tools {
		tools = append(tools, openaisdk.Tool{Type: openaisdk.ToolTypeFunction, Function: &openaisdk.FunctionDefinition{Name: tool.Name, Description: tool.Description, Parameters: tool.InputSchema}})
	}
	return &openaisdk.ChatCompletionRequest{Model: request.Model, Messages: messages, MaxTokens: request.MaxTokens, Temperature: request.Temperature, TopP: request.TopP, Stop: request.StopSequences, Tools: tools, ToolChoice: anthropicToolChoice(request.ToolChoice)}, nil
}

func anthropicToolChoice(value any) any {
	choice, ok := value.(map[string]any)
	if !ok {
		return value
	}
	switch choice["type"] {
	case "auto":
		return "auto"
	case "any":
		return "required"
	case "tool":
		return map[string]any{"type": "function", "function": map[string]any{"name": choice["name"]}}
	default:
		return value
	}
}

func compatibilityErrorMessage(body []byte, fallback string) string {
	var payload map[string]any
	if json.Unmarshal(body, &payload) == nil {
		switch value := payload["error"].(type) {
		case string:
			return value
		case map[string]any:
			if message, ok := value["message"].(string); ok {
				return message
			}
		}
	}
	if text := strings.TrimSpace(string(body)); text != "" {
		return text
	}
	return fallback
}

func anthropicErrorType(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "invalid_request_error"
	case http.StatusUnauthorized:
		return "authentication_error"
	case http.StatusForbidden:
		return "permission_error"
	case http.StatusNotFound:
		return "not_found_error"
	case http.StatusTooManyRequests:
		return "rate_limit_error"
	default:
		return "api_error"
	}
}

func writeAnthropicError(c *gin.Context, status int, errorType, message string) {
	c.JSON(status, gin.H{"type": "error", "error": gin.H{"type": errorType, "message": message}})
}

func writeResponsesError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": gin.H{"type": "invalid_request_error", "message": message}})
}

func isRequestTooLarge(err error) bool {
	var maxBytesError *http.MaxBytesError
	return errors.As(err, &maxBytesError)
}

func buildAnthropicResponse(response *myopenai.OpenAIResponse) map[string]any {
	content := make([]any, 0)
	stopReason := "end_turn"
	if len(response.Choices) > 0 {
		message := response.Choices[0].Message
		if message.Content != "" {
			content = append(content, map[string]any{"type": "text", "text": message.Content})
		}
		for _, tool := range message.ToolCalls {
			var input any = map[string]any{}
			_ = json.Unmarshal([]byte(tool.Function.Arguments), &input)
			content = append(content, map[string]any{"type": "tool_use", "id": tool.ID, "name": tool.Function.Name, "input": input})
			stopReason = "tool_use"
		}
	}
	usage := map[string]any{"input_tokens": 0, "output_tokens": 0}
	if response.Usage != nil {
		usage = map[string]any{"input_tokens": response.Usage.PromptTokens, "output_tokens": response.Usage.CompletionTokens}
	}
	return map[string]any{"id": response.ID, "type": "message", "role": "assistant", "model": response.Model, "content": content, "stop_reason": stopReason, "stop_sequence": nil, "usage": usage}
}

func writeAnthropicStream(c *gin.Context, response map[string]any) {
	c.Header("Content-Type", "text/event-stream")
	content := response["content"].([]any)
	start := map[string]any{}
	for key, value := range response {
		start[key] = value
	}
	start["content"] = []any{}
	start["stop_reason"] = nil
	writeSSE(c, "message_start", map[string]any{"type": "message_start", "message": start})
	for index, item := range content {
		block := item.(map[string]any)
		blockStart := block
		if block["type"] == "text" {
			blockStart = map[string]any{"type": "text", "text": ""}
		} else if block["type"] == "tool_use" {
			blockStart = map[string]any{"type": "tool_use", "id": block["id"], "name": block["name"], "input": map[string]any{}}
		}
		writeSSE(c, "content_block_start", map[string]any{"type": "content_block_start", "index": index, "content_block": blockStart})
		if block["type"] == "text" {
			writeSSE(c, "content_block_delta", map[string]any{"type": "content_block_delta", "index": index, "delta": map[string]any{"type": "text_delta", "text": block["text"]}})
		} else if block["type"] == "tool_use" {
			input, _ := json.Marshal(block["input"])
			writeSSE(c, "content_block_delta", map[string]any{"type": "content_block_delta", "index": index, "delta": map[string]any{"type": "input_json_delta", "partial_json": string(input)}})
		}
		writeSSE(c, "content_block_stop", map[string]any{"type": "content_block_stop", "index": index})
	}
	writeSSE(c, "message_delta", map[string]any{"type": "message_delta", "delta": map[string]any{"stop_reason": response["stop_reason"], "stop_sequence": nil}, "usage": response["usage"]})
	writeSSE(c, "message_stop", map[string]any{"type": "message_stop"})
}
