package adapter

import (
	"github.com/sashabaranov/go-openai"
	"simple-one-api/pkg/llm/claude"
	"simple-one-api/pkg/mycommon"
	myopenai "simple-one-api/pkg/openai"
	"time"
)

// OpenAIRequestToClaudeRequest 将 OpenAI 的 ChatCompletionRequest 转换为 Claude 的 RequestBody
func OpenAIRequestToClaudeRequest(oaiReq *openai.ChatCompletionRequest) *claude.RequestBody {
	claudeMessages := make([]claude.Message, len(oaiReq.Messages))

	for i, oaiMsg := range oaiReq.Messages {
		var content string
		var multiContent []claude.ContentBlock

		if oaiMsg.Content != "" {
			content = oaiMsg.Content
		} else {
			for _, part := range oaiMsg.MultiContent {
				cb := claude.ContentBlock{
					Type: string(part.Type),
					Text: part.Text,
				}

				if part.ImageURL != nil {
					imgData, mType, _ := mycommon.GetImageURLData(part.ImageURL.URL)
					cb.Image = &claude.Image{
						Source: claude.ImageSource{
							Type:      "base64",
							MediaType: mType, // Assuming media type, this might need adjustment
							Data:      imgData,
						},
					}
				}
				multiContent = append(multiContent, cb)
			}
		}

		claudeMessages[i] = claude.Message{
			Role:         oaiMsg.Role,
			Content:      content,
			MultiContent: multiContent,
		}
	}

	var metadata *claude.Metadata

	if oaiReq.User != "" {
		metadata = &claude.Metadata{UserID: oaiReq.User}
	}

	maxTokens := 4096
	if oaiReq.MaxTokens >= 0 {
		maxTokens = oaiReq.MaxTokens
	}

	return &claude.RequestBody{
		Model:         oaiReq.Model,
		Messages:      claudeMessages,
		MaxTokens:     maxTokens,
		StopSequences: oaiReq.Stop,
		Stream:        oaiReq.Stream,
		Temperature:   oaiReq.Temperature,
		TopK:          int(oaiReq.TopP), // Assuming TopP maps to TopK, adjust if needed
		TopP:          oaiReq.TopP,
		ToolChoice:    convertToolChoice(oaiReq.ToolChoice),
		Tools:         convertTools(oaiReq.Tools),
		Metadata:      metadata,
	}
}

func convertToolChoice(oaiToolChoice interface{}) *claude.ToolChoice {
	if oaiToolChoice == nil {
		return nil
	}

	switch tc := oaiToolChoice.(type) {
	case openai.ToolChoice:
		return &claude.ToolChoice{
			Type: string(tc.Type),
			Name: tc.Function.Name,
		}
	case string:
		return &claude.ToolChoice{
			Type: "tool",
			Name: tc,
		}
	default:
		return nil
	}
}

func convertTools(oaiTools []openai.Tool) []claude.Tool {
	claudeTools := make([]claude.Tool, len(oaiTools))

	for i, oaiTool := range oaiTools {
		claudeTools[i] = claude.Tool{
			Name:        oaiTool.Function.Name,
			Description: oaiTool.Function.Description,
			InputSchema: claude.ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"parameters": oaiTool.Function.Parameters,
				},
				Required: []string{"parameters"},
			},
		}
	}

	return claudeTools
}

// stopReasonToFinishReason 将 stop_reason 转换为 finish_reason
func claudeStopReasonToFinishReason(stopReason string) string {
	switch stopReason {
	case "end_turn":
		return "stop"
	case "max_tokens":
		return "length"
	case "stop_sequence":
		return "stop"
	case "tool_use":
		return "tool_calls"
	default:
		return "unknown"
	}
}

// ClaudeReponseToOpenAIResponse 将 claude.ResponseBody 转换为 myopenai.OpenAIResponse
func ClaudeReponseToOpenAIResponse(resp *claude.ResponseBody) *myopenai.OpenAIResponse {
	if resp == nil {
		return nil
	}

	choices := make([]myopenai.Choice, len(resp.Content))
	for i, content := range resp.Content {
		choices[i] = myopenai.Choice{
			Index: i,
			Message: myopenai.ResponseMessage{
				Role:    resp.Role,
				Content: content.Text, // 假设 RespContent 有一个 Text 字段
			},
			FinishReason: claudeStopReasonToFinishReason(resp.StopReason),
		}
	}

	usage := &myopenai.Usage{
		PromptTokens:     resp.Usage.InputTokens,
		CompletionTokens: resp.Usage.OutputTokens,
		TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
	}

	openAIResponse := &myopenai.OpenAIResponse{
		ID:      resp.ID,
		Object:  resp.Type,
		Created: time.Now().Unix(), // 假设我们在转换时设置当前时间戳
		Model:   resp.Model,
		Choices: choices,
		Usage:   usage,
	}

	return openAIResponse
}
