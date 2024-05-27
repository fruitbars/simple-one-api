package adapter

import (
	"github.com/sashabaranov/go-openai"
	myopenai "simple-one-api/pkg/openai"
)

func OpenAIRequestToOpenAIRequest(openAIReq myopenai.OpenAIRequest) *openai.ChatCompletionRequest {
	req := openai.ChatCompletionRequest{
		Model: openAIReq.Model,
		//MaxTokens: oaiReq.MaxTokens,
	}

	for _, msg := range openAIReq.Messages {
		tmpMsg := msg

		req.Messages = append(req.Messages, openai.ChatCompletionMessage{
			Role:    tmpMsg.Role,
			Content: tmpMsg.Content,
		})
	}

	if openAIReq.FrequencyPenalty != nil {
		req.FrequencyPenalty = float32(*openAIReq.FrequencyPenalty)
	}

	if openAIReq.LogitBias != nil {
		//req.LogitBias = openAIReq.LogitBias
	}

	if openAIReq.LogProbs != nil {
		req.LogProbs = *openAIReq.LogProbs
	}

	if openAIReq.TopLogProbs != nil {
		req.TopLogProbs = *openAIReq.TopLogProbs
	}

	if openAIReq.MaxTokens != nil {
		req.MaxTokens = *openAIReq.MaxTokens
	}

	if openAIReq.N != nil {
		req.N = *openAIReq.N
	}

	if openAIReq.PresencePenalty != nil {
		req.PresencePenalty = *openAIReq.PresencePenalty
	}

	if openAIReq.ResponseFormat != nil {
		//req.ResponseFormat = *openAIReq.ResponseFormat
	}

	if openAIReq.Seed != nil {
		req.Seed = openAIReq.Seed
	}

	if openAIReq.Stop != nil {
		req.Stop = openAIReq.Stop
	}

	if openAIReq.Stream != nil {
		req.Stream = *openAIReq.Stream
	}

	if openAIReq.Temperature != nil {
		req.Temperature = *openAIReq.Temperature
	}

	if openAIReq.TopP != nil {
		req.TopP = *openAIReq.TopP
	}

	if openAIReq.Tools != nil {
		//req.Tools = openAIReq.Tools
	}

	if openAIReq.User != nil {
		req.User = *openAIReq.User
	}

	return &req
}
