package adapter

import (
	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	baiduqianfan "simple-one-api/pkg/llm/baidu-qianfan"
	"simple-one-api/pkg/mycommon"
	"simple-one-api/pkg/mylog"
	myopenai "simple-one-api/pkg/openai"
	"strings"
)

// ModelConfig 用于存储每个模型的最小值、最大值和默认值
type qianFanModelConfig struct {
	Min        int
	Max        int
	DefaultMax int
}

// models 是一个映射，用于存储所有模型的配置
var qianFanModelPrefixes = map[string]qianFanModelConfig{
	"ERNIE-4.0-8K": {Min: 2, Max: 2048, DefaultMax: 2048},

	"ERNIE-3.5-8K":   {Min: 2, Max: 2048, DefaultMax: 2048},
	"ERNIE-3.5-128K": {Min: 2, Max: 4096, DefaultMax: 4096},

	"ERNIE-Speed-8K":   {Min: 2, Max: 2048, DefaultMax: 2048},
	"ERNIE-Speed-128K": {Min: 2, Max: 4096, DefaultMax: 4096},

	"ERNIE-Lite-8K":   {Min: 2, Max: 1024, DefaultMax: 1024},
	"ERNIE-Lite-128K": {Min: 2, Max: 2048, DefaultMax: 2048},

	"ERNIE-Tiny-8K": {Min: 2, Max: 2048, DefaultMax: 2048},

	// 添加更多模型配置
}

// validateMaxTokens 校验和调整maxtokens的值
func validateMaxTokens(maxtokens, min, max, defaultMax int) int {
	if maxtokens == 0 {
		return defaultMax
	} else if maxtokens < min {
		return min
	} else if maxtokens > max {
		return max
	}
	return maxtokens
}

// CheckMaxTokens 根据模型名称和maxtokens参数来校验和调整该参数的取值
func qianFanCheckMaxTokens(model string, maxtokens int) int {
	// 通过遍历modelPrefixes，找到匹配的模型前缀配置
	for prefix, config := range qianFanModelPrefixes {
		if strings.HasPrefix(model, prefix) {
			return validateMaxTokens(maxtokens, config.Min, config.Max, config.DefaultMax)
		}
	}
	mylog.Logger.Warn("Unknown model prefix", zap.String("model", model))
	return 0
}

func OpenAIRequestToQianFanRequest(oaiReq *openai.ChatCompletionRequest) *baiduqianfan.QianFanRequest {
	var req baiduqianfan.QianFanRequest

	for _, chatMsg := range req.Messages {
		qianMsg := mycommon.Message{
			Role:    chatMsg.Role,
			Content: chatMsg.Content,
		}
		req.Messages = append(req.Messages, qianMsg)
	}

	req.Stream = &oaiReq.Stream
	req.Stop = oaiReq.Stop

	if oaiReq.MaxTokens > 0 {
		maxTokens := qianFanCheckMaxTokens(oaiReq.Model, oaiReq.MaxTokens)
		req.MaxOutputTokens = &maxTokens
	}

	topP := float64(oaiReq.TopP) // 将 *float32 转换为 float64

	if topP < 0 {
		topP = 0
	}
	if topP > 1.0 {
		topP = 1.0
	}
	req.TopP = &topP

	temperature := float64(oaiReq.Temperature) // 将 *float32 转换为 float64

	if temperature <= 0 {
		temperature = 0.1
	}

	if temperature > 1 {
		temperature = 1
	}
	req.Temperature = &temperature

	req.Stream = &oaiReq.Stream
	// 处理系统名称或描述等可能需要自定义的转换
	req.UserID = &oaiReq.User

	if len(oaiReq.Messages) > 0 && strings.ToUpper(oaiReq.Messages[0].Role) == "SYSTEM" {
		if len(oaiReq.Messages) > 1 {
			oaiReq.Messages = oaiReq.Messages[1:]
		} else {
			// 处理数组长度不足的情况，例如可以清空或给出错误提示
			req.Messages = nil // 或其他适当的错误处理
		}

		req.System = &oaiReq.Messages[0].Content
	}

	for _, msg := range oaiReq.Messages {
		req.Messages = append(req.Messages, struct {
			Role    string `json:"role"`    // 用户或助手的角色
			Content string `json:"content"` // 对话内容
		}{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// 将FrequencyPenalty 转换为 PenaltyScore
	req.PenaltyScore = new(float64)
	frequencyPenalty := float64(oaiReq.FrequencyPenalty)
	*req.PenaltyScore = frequencyPenalty
	if oaiReq.FrequencyPenalty < 1.0 {
		*req.PenaltyScore = 1.0
	} else if oaiReq.FrequencyPenalty > 2.0 {
		*req.PenaltyScore = 2.0
	}

	return &req
}

func QianFanResponseToOpenAIResponse(qfResp *baiduqianfan.QianFanResponse) *myopenai.OpenAIResponse {
	// 创建一个 OpenAIResponse 实例
	if qfResp.ErrorCode != 0 && len(qfResp.ErrorMsg) > 0 {
		return &myopenai.OpenAIResponse{
			ID: qfResp.ID,
			Error: &myopenai.ErrorDetail{
				Message: qfResp.ErrorMsg,
				Code:    qfResp.ErrorCode,
			},
		}

	}

	oaResp := myopenai.OpenAIResponse{
		ID:                qfResp.ID,
		Object:            qfResp.Object,
		Created:           qfResp.Created,
		Model:             "", // 假定使用的模型
		SystemFingerprint: "", // 假定一个系统指纹
		Usage: &myopenai.Usage{
			PromptTokens:     qfResp.Usage.PromptTokens,
			CompletionTokens: qfResp.Usage.CompletionTokens,
			TotalTokens:      qfResp.Usage.TotalTokens,
		},
	}

	// 根据结果和是否结束设置 Choices
	choice := myopenai.Choice{
		Index: 0,
		Message: myopenai.ResponseMessage{
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

	mylog.Logger.Info("resp", zap.Any("oaResp", oaResp))

	return &oaResp
}

func QianFanResponseToOpenAIStreamResponse(qfResp *baiduqianfan.QianFanResponse) *myopenai.OpenAIStreamResponse {
	// 创建一个 OpenAIResponse 实例
	if qfResp.ErrorCode != 0 && len(qfResp.ErrorMsg) > 0 {
		mylog.Logger.Error("something error")
		return &myopenai.OpenAIStreamResponse{
			//ID: qfResp.ID,
			Error: &myopenai.ErrorDetail{
				Message: qfResp.ErrorMsg,
				Type:    "invalid_request_error",
				Code:    qfResp.ErrorCode,
			},
		}

	}

	oaResp := myopenai.OpenAIStreamResponse{
		ID:                qfResp.ID,
		Object:            "chat.completion.chunk",
		Created:           qfResp.Created,
		Model:             "", // 假定使用的模型
		SystemFingerprint: "", // 假定一个系统指纹
		Usage: &myopenai.Usage{
			PromptTokens:     qfResp.Usage.PromptTokens,
			CompletionTokens: qfResp.Usage.CompletionTokens,
			TotalTokens:      qfResp.Usage.TotalTokens,
		},
	}

	// 根据结果和是否结束设置 Choices
	choice := struct {
		Index        int                    `json:"index"`
		Delta        myopenai.ResponseDelta `json:"delta,omitempty"`
		Logprobs     any                    `json:"logprobs,omitempty"`
		FinishReason any                    `json:"finish_reason,omitempty"`
	}{}

	choice.Index = 0
	choice.Delta.Role = "assistant" // 假设角色为 assistant
	choice.Delta.Content = qfResp.Result

	// 如果 QianFanResponse 中有 IsEnd 且为 true，则认为对话结束
	if qfResp.IsEnd != nil && *qfResp.IsEnd {
		choice.FinishReason = "stop"
	}

	oaResp.Choices = append(oaResp.Choices, choice)

	mylog.Logger.Info("resp", zap.Any("resp", oaResp))

	return &oaResp
}
