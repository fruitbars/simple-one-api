package adapter

import (
	"github.com/google/uuid"
	"github.com/sashabaranov/go-openai"
	"simple-one-api/pkg/llm/ollama"
	myopenai "simple-one-api/pkg/openai"
	"simple-one-api/pkg/utils"
)

const (
	jsonFormat   = "json"
	textFormat   = "text"
	stopFinish   = "stop"
	lengthFinish = "length"
)

func OpenAIRequestToOllamaRequest(oaiReq *openai.ChatCompletionRequest) *ollama.ChatRequest {
	messages := make([]ollama.Message, len(oaiReq.Messages))
	for i, msg := range oaiReq.Messages {
		messages[i] = ollama.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	options := ollama.AdvancedModelOptions{
		Temperature: oaiReq.Temperature,
		TopP:        oaiReq.TopP,
		NumPredict:  oaiReq.MaxTokens,
	}

	return &ollama.ChatRequest{
		Model:    oaiReq.Model,
		Messages: messages,
		Stream:   oaiReq.Stream,
		Options:  options,
		Format:   getFormat(oaiReq.ResponseFormat),
	}
}

func getFormat(format *openai.ChatCompletionResponseFormat) string {
	if format == nil {
		return ""
	}

	switch format.Type {
	case openai.ChatCompletionResponseFormatTypeJSONObject:
		return jsonFormat
	case openai.ChatCompletionResponseFormatTypeText:
		return textFormat
	default:
		return ""
	}
}

func OllamaResponseToOpenAIResponse(resp *ollama.ChatResponse) *myopenai.OpenAIResponse {
	if resp == nil {
		return nil
	}

	choices := []myopenai.Choice{
		{
			Index: 0,
			Message: myopenai.ResponseMessage{
				Role:    resp.Message.Role,
				Content: resp.Message.Content,
			},
			//FinishReason: determineFinishReason(resp.Done),
		},
	}

	usage := &myopenai.Usage{
		PromptTokens:     resp.PromptEvalCount,
		CompletionTokens: resp.EvalCount,
		TotalTokens:      resp.PromptEvalCount + resp.EvalCount,
	}

	timeCreate, _ := utils.ParseRFC3339NanoToUnixTime(resp.CreatedAt)

	return &myopenai.OpenAIResponse{
		ID:      uuid.New().String(),
		Created: timeCreate,
		Model:   resp.Model,
		Choices: choices,
		Usage:   usage,
	}
}

func determineFinishReason(done bool) string {
	if done {
		return stopFinish
	}
	return lengthFinish
}

func OllamaResponseToOpenAIStreamResponse(resp *ollama.ChatResponse) *myopenai.OpenAIStreamResponse {
	if resp == nil {
		return nil
	}

	//log.Println(resp.Message.Role, resp.Message.Content)
	choices := []myopenai.OpenAIStreamResponseChoice{
		{
			Index: 0,
			Delta: myopenai.ResponseDelta{
				Role:    resp.Message.Role,
				Content: resp.Message.Content,
			},
		},
	}

	usage := &myopenai.Usage{
		PromptTokens:     resp.PromptEvalCount,
		CompletionTokens: resp.EvalCount,
		TotalTokens:      resp.PromptEvalCount + resp.EvalCount,
	}

	timeCreate, _ := utils.ParseRFC3339NanoToUnixTime(resp.CreatedAt)

	return &myopenai.OpenAIStreamResponse{
		ID:      uuid.New().String(),
		Created: timeCreate,
		Model:   resp.Model,
		Choices: choices,
		Usage:   usage,
	}
}
