package adapter

import (
	"encoding/json"
	"github.com/sashabaranov/go-openai"
	"log"
	"simple-one-api/pkg/llm/minimax"
	myopenai "simple-one-api/pkg/openai"
	"strings"
)

func OpenAIRequestToMinimaxRequest(openAIReq *openai.ChatCompletionRequest) *minimax.MinimaxRequest {
	var req minimax.MinimaxRequest

	req.Model = openAIReq.Model

	botName := "BOT"

	botSetting := struct {
		BotName string `json:"bot_name"` // 机器人的名字
		Content string `json:"content"`  // 具体机器人的设定
	}{BotName: botName, Content: botName}

	if len(openAIReq.Messages) > 0 && strings.ToUpper(openAIReq.Messages[0].Role) == "SYSTEM" {
		botSetting.Content = openAIReq.Messages[0].Content

		if len(openAIReq.Messages) > 1 {
			openAIReq.Messages = openAIReq.Messages[1:]
		} else {
			log.Println("message only has a SYSTEM message")
			// 处理数组长度不足的情况，例如可以清空或给出错误提示
			openAIReq.Messages = nil // 或其他适当的错误处理
		}
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

	req.Stream = openAIReq.Stream

	req.TopP = openAIReq.TopP

	req.Temperature = openAIReq.Temperature

	req.TokensToGenerate = int64(openAIReq.MaxTokens)

	if req.Model == "abab6-chat" {
		req.TokensToGenerate = 8192
	}

	return &req
}

func MinimaxResponseToOpenAIStreamResponse(minimaxResp *minimax.MinimaxResponse) *myopenai.OpenAIStreamResponse {
	openAIResp := &myopenai.OpenAIStreamResponse{
		ID:      minimaxResp.ID,
		Object:  "chat.completion.chunk",
		Created: minimaxResp.Created, // 使用当前 Unix 时间戳
		Model:   minimaxResp.Model,
	}

	// 转换 Usage
	openAIResp.Usage = &myopenai.Usage{
		TotalTokens: int(minimaxResp.Usage.TotalTokens),
	}

	for _, choice := range minimaxResp.Choices {
		for _, msg := range choice.Messages {
			openAIChoice := struct {
				Index        int                    `json:"index"`
				Delta        myopenai.ResponseDelta `json:"delta,omitempty"`
				Logprobs     any                    `json:"logprobs,omitempty"`
				FinishReason any                    `json:"finish_reason,omitempty"`
			}{
				Index: int(choice.Index),
				Delta: myopenai.ResponseDelta{
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
func MinimaxResponseToOpenAIResponse(minimaxResp *minimax.MinimaxResponse) *myopenai.OpenAIResponse {
	if minimaxResp == nil {
		return nil
	}

	// 转换 Choices
	var choices []myopenai.Choice
	for _, minimaxChoice := range minimaxResp.Choices {
		var messages []myopenai.ResponseMessage
		for _, msg := range minimaxChoice.Messages {
			messages = append(messages, myopenai.ResponseMessage{
				Role:    "assistant",
				Content: msg.Text,
			})
		}
		var logProbs json.RawMessage // 如果需要，可以处理 logProbs

		choices = append(choices, myopenai.Choice{
			Index:        int(minimaxChoice.Index),
			Message:      messages[0], // 假设只取第一个消息，如果有多个消息需要处理，请调整此处逻辑
			LogProbs:     &logProbs,
			FinishReason: minimaxChoice.FinishReason,
		})
	}

	// 转换 Usage
	usage := &myopenai.Usage{
		TotalTokens: int(minimaxResp.Usage.TotalTokens),
		// PromptTokens 和 CompletionTokens 需要额外处理
	}

	// 转换 Error
	var errorDetail *myopenai.ErrorDetail
	if minimaxResp.BaseResp.StatusCode != 0 {
		errorDetail = &myopenai.ErrorDetail{
			Message: minimaxResp.BaseResp.StatusMsg,
			Code:    minimaxResp.BaseResp.StatusCode,
		}
	}

	return &myopenai.OpenAIResponse{
		ID:      minimaxResp.ID,
		Created: minimaxResp.Created,
		Model:   minimaxResp.Model,
		Choices: choices,
		Usage:   usage,
		Error:   errorDetail,
	}
}
