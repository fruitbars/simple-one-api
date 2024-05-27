package adapter

import (
	"log"
	baidu_qianfan "simple-one-api/pkg/llm/baidu-qianfan"
	"simple-one-api/pkg/openai"
)

// 转换函数
func OpenAIRequestToQianFanRequest(oaiReq openai.OpenAIRequest) *baidu_qianfan.QianFanRequest {
	var req baidu_qianfan.QianFanRequest

	req.Messages = oaiReq.Messages
	req.Stream = oaiReq.Stream
	req.Stop = oaiReq.Stop
	req.MaxOutputTokens = oaiReq.MaxTokens

	if oaiReq.TopP != nil {
		topP := float64(*oaiReq.TopP) // 将 *float32 转换为 float64
		req.TopP = &topP
	}

	if oaiReq.Temperature != nil {
		temperature := float64(*oaiReq.Temperature) // 将 *float32 转换为 float64
		req.Temperature = &temperature
	}
	if oaiReq.Stream != nil {
		req.Stream = oaiReq.Stream
	}
	// 处理系统名称或描述等可能需要自定义的转换
	if oaiReq.User != nil {
		req.UserID = oaiReq.User
	}

	if oaiReq.Messages[0].Role == "system" {
		req.Messages = req.Messages[1:]
		req.System = &oaiReq.Messages[0].Content
	}

	// 将FrequencyPenalty 转换为 PenaltyScore
	if oaiReq.FrequencyPenalty != nil {
		req.PenaltyScore = new(float64)
		frequencyPenalty := float64(*oaiReq.FrequencyPenalty)
		*req.PenaltyScore = frequencyPenalty
		if *oaiReq.FrequencyPenalty < 1.0 {
			*req.PenaltyScore = 1.0
		} else if *oaiReq.FrequencyPenalty > 2.0 {
			*req.PenaltyScore = 2.0
		}
	}

	return &req
}

// 转换函数
func QianFanResponseToOpenAIResponse(qfResp *baidu_qianfan.QianFanResponse) *openai.OpenAIResponse {
	// 创建一个 OpenAIResponse 实例
	if qfResp.ErrorCode != 0 && len(qfResp.ErrorMsg) > 0 {
		return &openai.OpenAIResponse{
			ID: qfResp.ID,
			Error: &openai.ErrorDetail{
				Message: qfResp.ErrorMsg,
				Code:    qfResp.ErrorCode,
			},
		}

	}

	oaResp := openai.OpenAIResponse{
		ID:                qfResp.ID,
		Object:            qfResp.Object,
		Created:           int64(qfResp.Created),
		Model:             "", // 假定使用的模型
		SystemFingerprint: "", // 假定一个系统指纹
		Usage: &openai.Usage{
			PromptTokens:     qfResp.Usage.PromptTokens,
			CompletionTokens: qfResp.Usage.CompletionTokens,
			TotalTokens:      qfResp.Usage.TotalTokens,
		},
	}

	// 根据结果和是否结束设置 Choices
	choice := openai.Choice{
		Index: 0,
		Message: openai.ResponseMessage{
			Role:    "assistant", // 默认设置为助手回复
			Content: qfResp.Result,
		},
		FinishReason: "completed", // 默认完成原因，可以根据QianFanResponse字段进一步定制
	}

	// 如果 QianFanResponse 中有 IsEnd 且为 true，则认为对话结束
	if qfResp.IsEnd != nil && *qfResp.IsEnd {
		choice.FinishReason = "stop"
	}

	// 将 Choice 添加到 Choices 数组
	oaResp.Choices = append(oaResp.Choices, choice)

	log.Println(oaResp)

	return &oaResp
}

func QianFanResponseToOpenAIStreamResponse(qfResp *baidu_qianfan.QianFanResponse) *openai.OpenAIStreamResponse {
	// 创建一个 OpenAIResponse 实例
	if qfResp.ErrorCode != 0 && len(qfResp.ErrorMsg) > 0 {

		log.Println("something error")
		return &openai.OpenAIStreamResponse{
			//ID: qfResp.ID,
			Error: &openai.ErrorDetail{
				Message: qfResp.ErrorMsg,
				Type:    "invalid_request_error",
				Code:    qfResp.ErrorCode,
			},
		}

	}

	oaResp := openai.OpenAIStreamResponse{
		ID:                qfResp.ID,
		Object:            "chat.completion.chunk",
		Created:           qfResp.Created,
		Model:             "", // 假定使用的模型
		SystemFingerprint: "", // 假定一个系统指纹
		Usage: &openai.Usage{
			PromptTokens:     qfResp.Usage.PromptTokens,
			CompletionTokens: qfResp.Usage.CompletionTokens,
			TotalTokens:      qfResp.Usage.TotalTokens,
		},
	}

	// 根据结果和是否结束设置 Choices
	choice := struct {
		Index int `json:"index,omitempty"`
		Delta struct {
			Role    string `json:"role,omitempty"`
			Content string `json:"content,omitempty"`
		} `json:"delta,omitempty"`
		Logprobs     any `json:"logprobs,omitempty"`
		FinishReason any `json:"finish_reason,omitempty"`
	}{}

	choice.Index = 0
	choice.Delta.Role = "assistant" // 假设角色为 assistant
	choice.Delta.Content = qfResp.Result

	// 如果 QianFanResponse 中有 IsEnd 且为 true，则认为对话结束
	if qfResp.IsEnd != nil && *qfResp.IsEnd {
		choice.FinishReason = "stop"
	}

	oaResp.Choices = append(oaResp.Choices, choice)

	log.Println(oaResp)

	return &oaResp
}
