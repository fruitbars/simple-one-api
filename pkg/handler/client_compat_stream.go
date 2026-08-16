package handler

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	openaisdk "github.com/sashabaranov/go-openai"
)

type compatibilityBuffer struct {
	gin.ResponseWriter
	header http.Header
	status int
	buffer bytes.Buffer
}

func newCompatibilityBuffer(base gin.ResponseWriter) *compatibilityBuffer {
	return &compatibilityBuffer{ResponseWriter: base, header: make(http.Header), status: http.StatusOK}
}

func (writer *compatibilityBuffer) Header() http.Header            { return writer.header }
func (writer *compatibilityBuffer) WriteHeader(status int)         { writer.status = status }
func (writer *compatibilityBuffer) WriteHeaderNow()                {}
func (writer *compatibilityBuffer) Status() int                    { return writer.status }
func (writer *compatibilityBuffer) Size() int                      { return writer.buffer.Len() }
func (writer *compatibilityBuffer) Written() bool                  { return writer.buffer.Len() > 0 }
func (writer *compatibilityBuffer) Write(data []byte) (int, error) { return writer.buffer.Write(data) }
func (writer *compatibilityBuffer) WriteString(data string) (int, error) {
	return writer.buffer.WriteString(data)
}
func (writer *compatibilityBuffer) Bytes() []byte {
	return append([]byte(nil), writer.buffer.Bytes()...)
}

type compatibilityProtocol int

const (
	compatibilityProtocolResponses compatibilityProtocol = iota
	compatibilityProtocolAnthropic
)

type compatibilityStreamWriter struct {
	gin.ResponseWriter
	header  http.Header
	status  int
	size    int
	buffer  strings.Builder
	emitter *compatibilityStreamEmitter
}

func executeCompatibilityStream(c *gin.Context, request *openaisdk.ChatCompletionRequest, protocol compatibilityProtocol) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	emitter := newCompatibilityStreamEmitter(c, protocol, request.Model)
	writer := &compatibilityStreamWriter{ResponseWriter: c.Writer, header: make(http.Header), status: http.StatusOK, emitter: emitter}
	inner, _ := gin.CreateTestContext(writer)
	inner.Request = c.Request.Clone(c.Request.Context())
	request.Stream = true
	request.StreamOptions = &openaisdk.StreamOptions{IncludeUsage: true}
	HandleOpenAIRequest(inner, request)
	writer.finish()
	if writer.status >= http.StatusBadRequest && !emitter.started {
		message := compatibilityErrorMessage([]byte(writer.buffer.String()), http.StatusText(writer.status))
		c.Header("Content-Type", "application/json")
		if protocol == compatibilityProtocolAnthropic {
			writeAnthropicError(c, writer.status, anthropicErrorType(writer.status), message)
		} else {
			writeResponsesError(c, writer.status, message)
		}
		return
	}
	if writer.status >= http.StatusBadRequest {
		emitter.fail(compatibilityErrorMessage([]byte(writer.buffer.String()), http.StatusText(writer.status)))
		return
	}
	emitter.finish()
}

func (writer *compatibilityStreamWriter) Header() http.Header    { return writer.header }
func (writer *compatibilityStreamWriter) WriteHeader(status int) { writer.status = status }
func (writer *compatibilityStreamWriter) WriteHeaderNow()        {}
func (writer *compatibilityStreamWriter) Status() int            { return writer.status }
func (writer *compatibilityStreamWriter) Size() int              { return writer.size }
func (writer *compatibilityStreamWriter) Written() bool          { return writer.size > 0 }

func (writer *compatibilityStreamWriter) Write(data []byte) (int, error) {
	writer.size += len(data)
	writer.buffer.Write(data)
	if writer.status < http.StatusBadRequest {
		writer.consumeLines(false)
	}
	return len(data), nil
}

func (writer *compatibilityStreamWriter) WriteString(data string) (int, error) {
	return writer.Write([]byte(data))
}

func (writer *compatibilityStreamWriter) Flush() {
	writer.consumeLines(false)
}

func (writer *compatibilityStreamWriter) finish() {
	writer.consumeLines(true)
}

func (writer *compatibilityStreamWriter) consumeLines(final bool) {
	content := writer.buffer.String()
	lastNewline := strings.LastIndexByte(content, '\n')
	if final {
		lastNewline = len(content) - 1
	}
	if lastNewline < 0 {
		return
	}
	consumed := content[:lastNewline+1]
	remainder := content[lastNewline+1:]
	writer.buffer.Reset()
	writer.buffer.WriteString(remainder)
	scanner := bufio.NewScanner(strings.NewReader(consumed))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" || payload == "[DONE]" {
			continue
		}
		var chunk openaisdk.ChatCompletionStreamResponse
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			writer.emitter.fail("invalid upstream stream event: " + err.Error())
			continue
		}
		writer.emitter.emit(chunk)
	}
}

type compatibilityToolState struct {
	index       int
	id          string
	name        string
	arguments   strings.Builder
	outputIndex int
	blockIndex  int
	started     bool
}

type compatibilityStreamEmitter struct {
	context     *gin.Context
	protocol    compatibilityProtocol
	model       string
	id          string
	started     bool
	finished    bool
	sequence    int
	textStarted bool
	text        strings.Builder
	tools       map[int]*compatibilityToolState
	usage       *openaisdk.Usage
}

func newCompatibilityStreamEmitter(context *gin.Context, protocol compatibilityProtocol, model string) *compatibilityStreamEmitter {
	return &compatibilityStreamEmitter{context: context, protocol: protocol, model: model, tools: make(map[int]*compatibilityToolState)}
}

func (emitter *compatibilityStreamEmitter) ensureStarted(chunk openaisdk.ChatCompletionStreamResponse) {
	if emitter.started {
		return
	}
	emitter.started = true
	emitter.id = chunk.ID
	if emitter.id == "" {
		emitter.id = fmt.Sprintf("compat_%d", time.Now().UnixNano())
	}
	if chunk.Model != "" {
		emitter.model = chunk.Model
	}
	if emitter.protocol == compatibilityProtocolResponses {
		response := map[string]any{"id": "resp_" + emitter.id, "object": "response", "created_at": time.Now().Unix(), "status": "in_progress", "model": emitter.model, "output": []any{}, "usage": nil}
		emitter.responsesEvent("response.created", map[string]any{"response": response})
	} else {
		message := map[string]any{"id": emitter.id, "type": "message", "role": "assistant", "model": emitter.model, "content": []any{}, "stop_reason": nil, "stop_sequence": nil, "usage": map[string]any{"input_tokens": 0, "output_tokens": 0}}
		writeSSE(emitter.context, "message_start", map[string]any{"type": "message_start", "message": message})
	}
}

func (emitter *compatibilityStreamEmitter) emit(chunk openaisdk.ChatCompletionStreamResponse) {
	emitter.ensureStarted(chunk)
	if chunk.Usage != nil {
		emitter.usage = chunk.Usage
	}
	for _, choice := range chunk.Choices {
		delta := choice.Delta
		text := delta.Content
		if text == "" {
			text = delta.Refusal
		}
		if text != "" {
			emitter.emitText(text)
		}
		for _, call := range delta.ToolCalls {
			emitter.emitTool(call)
		}
	}
}

func (emitter *compatibilityStreamEmitter) emitText(delta string) {
	if !emitter.textStarted {
		emitter.textStarted = true
		if emitter.protocol == compatibilityProtocolResponses {
			item := map[string]any{"id": "msg_" + emitter.id, "type": "message", "status": "in_progress", "role": "assistant", "content": []any{}}
			emitter.responsesEvent("response.output_item.added", map[string]any{"output_index": 0, "item": item})
			emitter.responsesEvent("response.content_part.added", map[string]any{"output_index": 0, "content_index": 0, "part": map[string]any{"type": "output_text", "text": "", "annotations": []any{}}})
		} else {
			writeSSE(emitter.context, "content_block_start", map[string]any{"type": "content_block_start", "index": 0, "content_block": map[string]any{"type": "text", "text": ""}})
		}
	}
	emitter.text.WriteString(delta)
	if emitter.protocol == compatibilityProtocolResponses {
		emitter.responsesEvent("response.output_text.delta", map[string]any{"output_index": 0, "content_index": 0, "delta": delta})
	} else {
		writeSSE(emitter.context, "content_block_delta", map[string]any{"type": "content_block_delta", "index": 0, "delta": map[string]any{"type": "text_delta", "text": delta}})
	}
}

func (emitter *compatibilityStreamEmitter) emitTool(call openaisdk.ToolCall) {
	index := 0
	if call.Index != nil {
		index = *call.Index
	}
	state := emitter.tools[index]
	if state == nil {
		state = &compatibilityToolState{index: index, outputIndex: len(emitter.tools), blockIndex: len(emitter.tools)}
		if emitter.textStarted {
			state.outputIndex++
			state.blockIndex++
		}
		emitter.tools[index] = state
	}
	if call.ID != "" {
		state.id = call.ID
	}
	if call.Function.Name != "" {
		state.name = call.Function.Name
	}
	if !state.started {
		state.started = true
		if state.id == "" {
			state.id = fmt.Sprintf("call_%d_%d", time.Now().UnixNano(), index)
		}
		if emitter.protocol == compatibilityProtocolResponses {
			item := map[string]any{"id": "fc_" + state.id, "type": "function_call", "status": "in_progress", "call_id": state.id, "name": state.name, "arguments": ""}
			emitter.responsesEvent("response.output_item.added", map[string]any{"output_index": state.outputIndex, "item": item})
		} else {
			writeSSE(emitter.context, "content_block_start", map[string]any{"type": "content_block_start", "index": state.blockIndex, "content_block": map[string]any{"type": "tool_use", "id": state.id, "name": state.name, "input": map[string]any{}}})
		}
	}
	if call.Function.Arguments == "" {
		return
	}
	state.arguments.WriteString(call.Function.Arguments)
	if emitter.protocol == compatibilityProtocolResponses {
		emitter.responsesEvent("response.function_call_arguments.delta", map[string]any{"output_index": state.outputIndex, "item_id": "fc_" + state.id, "delta": call.Function.Arguments})
	} else {
		writeSSE(emitter.context, "content_block_delta", map[string]any{"type": "content_block_delta", "index": state.blockIndex, "delta": map[string]any{"type": "input_json_delta", "partial_json": call.Function.Arguments}})
	}
}

func (emitter *compatibilityStreamEmitter) finish() {
	if emitter.finished {
		return
	}
	emitter.finished = true
	if !emitter.started {
		emitter.ensureStarted(openaisdk.ChatCompletionStreamResponse{Model: emitter.model})
	}
	if emitter.protocol == compatibilityProtocolResponses {
		output := make([]any, 0, 1+len(emitter.tools))
		if emitter.textStarted {
			text := emitter.text.String()
			emitter.responsesEvent("response.output_text.done", map[string]any{"output_index": 0, "content_index": 0, "text": text})
			part := map[string]any{"type": "output_text", "text": text, "annotations": []any{}}
			emitter.responsesEvent("response.content_part.done", map[string]any{"output_index": 0, "content_index": 0, "part": part})
			item := map[string]any{"id": "msg_" + emitter.id, "type": "message", "status": "completed", "role": "assistant", "content": []any{part}}
			emitter.responsesEvent("response.output_item.done", map[string]any{"output_index": 0, "item": item})
			output = append(output, item)
		}
		for _, tool := range emitter.orderedTools() {
			arguments := tool.arguments.String()
			emitter.responsesEvent("response.function_call_arguments.done", map[string]any{"output_index": tool.outputIndex, "item_id": "fc_" + tool.id, "arguments": arguments})
			item := map[string]any{"id": "fc_" + tool.id, "type": "function_call", "status": "completed", "call_id": tool.id, "name": tool.name, "arguments": arguments}
			emitter.responsesEvent("response.output_item.done", map[string]any{"output_index": tool.outputIndex, "item": item})
			output = append(output, item)
		}
		usage := emitter.responsesUsage()
		response := map[string]any{"id": "resp_" + emitter.id, "object": "response", "created_at": time.Now().Unix(), "status": "completed", "model": emitter.model, "output": output, "usage": usage, "error": nil, "incomplete_details": nil}
		emitter.responsesEvent("response.completed", map[string]any{"response": response})
		return
	}
	if emitter.textStarted {
		writeSSE(emitter.context, "content_block_stop", map[string]any{"type": "content_block_stop", "index": 0})
	}
	for _, tool := range emitter.orderedTools() {
		writeSSE(emitter.context, "content_block_stop", map[string]any{"type": "content_block_stop", "index": tool.blockIndex})
	}
	stopReason := "end_turn"
	if len(emitter.tools) > 0 {
		stopReason = "tool_use"
	}
	writeSSE(emitter.context, "message_delta", map[string]any{"type": "message_delta", "delta": map[string]any{"stop_reason": stopReason, "stop_sequence": nil}, "usage": emitter.anthropicUsage()})
	writeSSE(emitter.context, "message_stop", map[string]any{"type": "message_stop"})
}

func (emitter *compatibilityStreamEmitter) fail(message string) {
	if emitter.finished {
		return
	}
	emitter.finished = true
	if emitter.protocol == compatibilityProtocolResponses {
		emitter.responsesEvent("error", map[string]any{"error": map[string]any{"type": "api_error", "message": message}})
	} else {
		writeSSE(emitter.context, "error", map[string]any{"type": "error", "error": map[string]any{"type": "api_error", "message": message}})
	}
}

func (emitter *compatibilityStreamEmitter) responsesEvent(event string, fields map[string]any) {
	fields["type"] = event
	fields["sequence_number"] = emitter.sequence
	emitter.sequence++
	writeSSE(emitter.context, event, fields)
}

func (emitter *compatibilityStreamEmitter) orderedTools() []*compatibilityToolState {
	result := make([]*compatibilityToolState, 0, len(emitter.tools))
	for index := 0; index < len(emitter.tools); index++ {
		if tool := emitter.tools[index]; tool != nil {
			result = append(result, tool)
		}
	}
	return result
}

func (emitter *compatibilityStreamEmitter) responsesUsage() map[string]any {
	if emitter.usage == nil {
		return map[string]any{"input_tokens": 0, "output_tokens": 0, "total_tokens": 0}
	}
	return map[string]any{"input_tokens": emitter.usage.PromptTokens, "output_tokens": emitter.usage.CompletionTokens, "total_tokens": emitter.usage.TotalTokens}
}

func (emitter *compatibilityStreamEmitter) anthropicUsage() map[string]any {
	if emitter.usage == nil {
		return map[string]any{"input_tokens": 0, "output_tokens": 0}
	}
	return map[string]any{"input_tokens": emitter.usage.PromptTokens, "output_tokens": emitter.usage.CompletionTokens}
}
