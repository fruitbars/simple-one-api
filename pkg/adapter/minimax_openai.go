package adapter

import (
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
