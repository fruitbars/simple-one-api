package adapter

import (
	"encoding/json"
	"github.com/sashabaranov/go-openai"
	myopenai "simple-one-api/pkg/openai"
)

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
