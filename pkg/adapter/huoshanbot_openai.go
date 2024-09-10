package adapter

import (
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
	myopenai "simple-one-api/pkg/openai"
	"time"
)

func HuoShanBotResponseToOpenAIResponse(huoshanBotResp *model.BotChatCompletionResponse) *myopenai.OpenAIResponse {
	if huoshanBotResp == nil {
		return nil
	}

	resp := huoshanBotResp.ChatCompletionResponse

	// 转换 Choices
	var choices []myopenai.Choice
	for _, choice := range resp.Choices {
		var content string
		if choice.Message.Content != nil && choice.Message.Content.StringValue != nil {
			content = *choice.Message.Content.StringValue
		}

		choices = append(choices, myopenai.Choice{
			Index: choice.Index,
			Message: myopenai.ResponseMessage{
				Role:    choice.Message.Role,
				Content: content,
			},
			LogProbs:     nil, // 假设 logprobs 不存在
			FinishReason: string(choice.FinishReason),
		})
	}

	var mu model.BotModelUsage
	if huoshanBotResp.BotUsage != nil && len(huoshanBotResp.BotUsage.ModelUsage) > 0 {
		mu = *huoshanBotResp.BotUsage.ModelUsage[0]
	}
	// 转换 Usage
	usage := &myopenai.Usage{
		PromptTokens:     mu.PromptTokens,
		CompletionTokens: mu.CompletionTokens,
		TotalTokens:      mu.TotalTokens,
	}

	// 创建 OpenAIResponse
	openAIResp := &myopenai.OpenAIResponse{
		ID:      resp.ID,
		Object:  resp.Object,
		Created: resp.Created,
		Model:   resp.Model,
		Choices: choices,
		Usage:   usage,
		// Error 信息可以根据具体需求进行设置
	}

	return openAIResp
}

// HuoShanBotResponseToOpenAIStreamResponse converts a HuoShanBot stream response to an OpenAIStreamResponse
func HuoShanBotResponseToOpenAIStreamResponse(huoshanBotResp *model.BotChatCompletionStreamResponse) *myopenai.OpenAIStreamResponse {

	var mu model.BotModelUsage
	if huoshanBotResp.BotUsage != nil && len(huoshanBotResp.BotUsage.ModelUsage) > 0 {
		mu = *huoshanBotResp.BotUsage.ModelUsage[0]
	}
	// 转换 Usage
	usage := &myopenai.Usage{
		PromptTokens:     mu.PromptTokens,
		CompletionTokens: mu.CompletionTokens,
		TotalTokens:      mu.TotalTokens,
	}

	response := &myopenai.OpenAIStreamResponse{
		ID:      huoshanBotResp.ID,
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   huoshanBotResp.Model,
		Choices: make([]myopenai.OpenAIStreamResponseChoice, len(huoshanBotResp.Choices)),
		Usage:   usage,
		//Error:   mapErrorDetails(huoshanBotResp.Error),
	}

	for i, choice := range huoshanBotResp.Choices {

		response.Choices[i] = myopenai.OpenAIStreamResponseChoice{
			Index: choice.Index,
			Delta: myopenai.ResponseDelta{
				Role:    choice.Delta.Role,
				Content: choice.Delta.Content,
			},
			Logprobs:     nil,
			FinishReason: nil,
		}
	}

	return response
}
