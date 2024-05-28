package adapter

import (
	"encoding/json"
	"simple-one-api/pkg/llm/minimax"
	"simple-one-api/pkg/openai"
	"strings"
)

func OpenAIRequestToMinimaxRequest(openAIReq openai.OpenAIRequest) *minimax.MinimaxRequest {
	var req minimax.MinimaxRequest

	req.Model = openAIReq.Model

	botName := "BOT"

	botSetting := struct {
		BotName string `json:"bot_name"` // 机器人的名字
		Content string `json:"content"`  // 具体机器人的设定
	}{BotName: botName, Content: botName}

	if len(openAIReq.Messages) > 0 && strings.ToUpper(openAIReq.Messages[0].Role) == "SYSTEM" {
		botSetting.Content = openAIReq.Messages[0].Content
		openAIReq.Messages = openAIReq.Messages[1:]
	}

	req.BotSetting = append(req.BotSetting, botSetting)
	// 将 OpenAIRequest 的 Messages 转换为 SparkChatRequest 的 Message
	for _, msg := range openAIReq.Messages {
		role := strings.ToUpper(msg.Role)
		if strings.ToUpper(msg.Role) == "ASSISTANT" {
			role = "BOT"
		}
		req.Messages = append(req.Messages, struct {
			SenderType string `json:"sender_type"` // 发送者类型
			SenderName string `json:"sender_name"` // 发送者名称
			Text       string `json:"text"`
		}{
			SenderType: role,
			SenderName: strings.ToUpper(role),
			Text:       msg.Content,
		})
	}

	req.ReplyConstraints.SenderType = botName
	req.ReplyConstraints.SenderName = botName

	if openAIReq.Stream != nil {
		req.Stream = *openAIReq.Stream
	}

	if openAIReq.TopP != nil {
		req.TopP = *openAIReq.TopP
	}

	if openAIReq.Temperature != nil {
		req.Temperature = *openAIReq.Temperature
	}

	if openAIReq.MaxTokens != nil {
		req.TokensToGenerate = int64(*openAIReq.MaxTokens)
	}

	return &req
}

func MinimaxResponseToOpenAIStreamResponse(minimaxResp *minimax.MinimaxResponse) *openai.OpenAIStreamResponse {
	openAIResp := &openai.OpenAIStreamResponse{
		ID:      minimaxResp.ID,
		Object:  "chat.completion.chunk",
		Created: int(minimaxResp.Created), // 使用当前 Unix 时间戳
		Model:   minimaxResp.Model,
	}

	// 转换 Usage
	openAIResp.Usage = &openai.Usage{
		TotalTokens: int(minimaxResp.Usage.TotalTokens),
	}

	for _, choice := range minimaxResp.Choices {
		for _, msg := range choice.Messages {
			openAIChoice := struct {
				Index int `json:"index,omitempty"`
				Delta struct {
					Role    string `json:"role,omitempty"`
					Content string `json:"content,omitempty"`
				} `json:"delta,omitempty"`
				Logprobs     any `json:"logprobs,omitempty"`
				FinishReason any `json:"finish_reason,omitempty"`
			}{
				Index: int(choice.Index),
				Delta: struct {
					Role    string `json:"role,omitempty"`
					Content string `json:"content,omitempty"`
				}{
					Role:    "assistant",
					Content: msg.Text,
				},
				FinishReason: choice.FinishReason,
			}
			openAIResp.Choices = append(openAIResp.Choices, openAIChoice)
		}
	}

	return openAIResp
}

// MinimaxResponseToOpenAIResponse 将 MinimaxResponse 转换为 OpenAIResponse
func MinimaxResponseToOpenAIResponse(minimaxResp *minimax.MinimaxResponse) *openai.OpenAIResponse {
	if minimaxResp == nil {
		return nil
	}

	// 转换 Choices
	var choices []openai.Choice
	for _, minimaxChoice := range minimaxResp.Choices {
		var messages []openai.ResponseMessage
		for _, msg := range minimaxChoice.Messages {
			messages = append(messages, openai.ResponseMessage{
				Role:    "assistant",
				Content: msg.Text,
			})
		}
		var logProbs json.RawMessage // 如果需要，可以处理 logProbs

		choices = append(choices, openai.Choice{
			Index:        int(minimaxChoice.Index),
			Message:      messages[0], // 假设只取第一个消息，如果有多个消息需要处理，请调整此处逻辑
			LogProbs:     &logProbs,
			FinishReason: minimaxChoice.FinishReason,
		})
	}

	// 转换 Usage
	usage := &openai.Usage{
		TotalTokens: int(minimaxResp.Usage.TotalTokens),
		// PromptTokens 和 CompletionTokens 需要额外处理
	}

	// 转换 Error
	var errorDetail *openai.ErrorDetail
	if minimaxResp.BaseResp.StatusCode != 0 {
		errorDetail = &openai.ErrorDetail{
			Message: minimaxResp.BaseResp.StatusMsg,
			Code:    minimaxResp.BaseResp.StatusCode,
		}
	}

	return &openai.OpenAIResponse{
		ID:      minimaxResp.ID,
		Created: minimaxResp.Created,
		Model:   minimaxResp.Model,
		Choices: choices,
		Usage:   usage,
		Error:   errorDetail,
	}
}
