package adapter

import (
	"encoding/json"
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

func OpenAIResponseToOpenAIResponse(resp *openai.ChatCompletionResponse) *myopenai.OpenAIResponse {
	if resp == nil {
		return nil
	}

	var choices []myopenai.Choice
	for _, choice := range resp.Choices {
		message := myopenai.ResponseMessage{
			Role:    choice.Message.Role,
			Content: choice.Message.Content,
		}
		var logProbs json.RawMessage
		if choice.LogProbs != nil {
			logProbs, _ = json.Marshal(choice.LogProbs)
		}
		choices = append(choices, myopenai.Choice{
			Index:        choice.Index,
			Message:      message,
			LogProbs:     &logProbs,
			FinishReason: string(choice.FinishReason),
		})
	}

	usage := myopenai.Usage{
		PromptTokens:     resp.Usage.PromptTokens,
		CompletionTokens: resp.Usage.CompletionTokens,
		TotalTokens:      resp.Usage.TotalTokens,
	}

	return &myopenai.OpenAIResponse{
		ID:                resp.ID,
		Object:            resp.Object,
		Created:           resp.Created,
		Model:             resp.Model,
		SystemFingerprint: resp.SystemFingerprint,
		Choices:           choices,
		Usage:             &usage,
	}
}
