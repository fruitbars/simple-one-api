package adapter

import (
	"github.com/fruitbars/gosparkclient"
	"simple-one-api/pkg/openai"
	"time"
)

func OpenAIRequestToXingHuoRequest(openAIReq openai.OpenAIRequest) *gosparkclient.SparkChatRequest {
	var sparkChatReq gosparkclient.SparkChatRequest

	// 将 OpenAIRequest 的 Messages 转换为 SparkChatRequest 的 Message
	for _, msg := range openAIReq.Messages {
		sparkChatReq.Message = append(sparkChatReq.Message, struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	if openAIReq.TopP != nil {
		sparkChatReq.Topk = int(*openAIReq.TopP)
	}

	if openAIReq.Temperature != nil {
		sparkChatReq.Temperature = float64(*openAIReq.Temperature)
	}

	if openAIReq.MaxTokens != nil {
		sparkChatReq.Maxtokens = *openAIReq.MaxTokens
	}

	return &sparkChatReq
}

// 转换函数
func XingHuoResponseToOpenAIResponse(qfResp *gosparkclient.SparkAPIResponse) *openai.OpenAIResponse {
	openAIResp := &openai.OpenAIResponse{
		ID:                qfResp.Header.Sid,
		Object:            "text_completion",
		SystemFingerprint: qfResp.Header.Message,
	}

	// 转换 Choices
	for _, choice := range qfResp.Payload.Choices.Text {
		openAIResp.Choices = append(openAIResp.Choices, openai.Choice{
			Index: choice.Index,
			Message: openai.ResponseMessage{
				Role:    choice.Role,
				Content: choice.Content,
			},
		})
	}

	// 转换 Usage
	openAIResp.Usage = &openai.Usage{
		PromptTokens:     qfResp.Payload.Usage.Text.PromptTokens,
		CompletionTokens: qfResp.Payload.Usage.Text.CompletionTokens,
		TotalTokens:      qfResp.Payload.Usage.Text.TotalTokens,
	}

	// 设置 Created 时间为当前 Unix 时间戳（如果需要的话）
	openAIResp.Created = time.Now().Unix() // 你可以使用 time.Now().Unix() 设置为当前时间戳

	return openAIResp
}

func XingHuoResponseToOpenAIStreamResponse(qfResp *gosparkclient.SparkAPIResponse) *openai.OpenAIStreamResponse {
	openAIResp := &openai.OpenAIStreamResponse{
		ID:                qfResp.Header.Sid,
		Object:            "chat.completion.chunk",
		Created:           int(time.Now().Unix()), // 使用当前 Unix 时间戳
		SystemFingerprint: qfResp.Header.Message,
	}

	// 转换 Choices
	for _, choice := range qfResp.Payload.Choices.Text {
		openAIResp.Choices = append(openAIResp.Choices, struct {
			Index int `json:"index,omitempty"`
			Delta struct {
				Role    string `json:"role,omitempty"`
				Content string `json:"content,omitempty"`
			} `json:"delta,omitempty"`
			Logprobs     interface{} `json:"logprobs,omitempty"`
			FinishReason interface{} `json:"finish_reason,omitempty"`
		}{
			Index: choice.Index,
			Delta: struct {
				Role    string `json:"role,omitempty"`
				Content string `json:"content,omitempty"`
			}{
				Role:    choice.Role,
				Content: choice.Content,
			},
		})
	}

	// 转换 Usage
	openAIResp.Usage = &openai.Usage{
		PromptTokens:     qfResp.Payload.Usage.Text.PromptTokens,
		CompletionTokens: qfResp.Payload.Usage.Text.CompletionTokens,
		TotalTokens:      qfResp.Payload.Usage.Text.TotalTokens,
	}

	return openAIResp
}
