package adapter

import (
	"github.com/Azure/azure-sdk-for-go/sdk/ai/azopenai"
	"github.com/sashabaranov/go-openai"
)

func convertMessages2AzureMessage(chatMessages []openai.ChatCompletionMessage) []azopenai.ChatRequestMessageClassification {
	var messages []azopenai.ChatRequestMessageClassification

	for _, msg := range chatMessages {
		switch msg.Role {
		case "system":
			messages = append(messages, &azopenai.ChatRequestSystemMessage{
				Content: azopenai.NewChatRequestSystemMessageContent(msg.Content),
			})
		case "user":
			messages = append(messages, &azopenai.ChatRequestUserMessage{
				Content: azopenai.NewChatRequestUserMessageContent(msg.Content),
			})
		case "assistant":
			messages = append(messages, &azopenai.ChatRequestAssistantMessage{
				Content: azopenai.NewChatRequestAssistantMessageContent(msg.Content),
			})
		default:
			// 如果遇到未知的role，可以选择忽略或报错
			continue
		}
	}

	return messages
}

func OpenAIRequestToAzureRequest(oaiReq *openai.ChatCompletionRequest) *azopenai.ChatCompletionsOptions {
	azureMessages := convertMessages2AzureMessage(oaiReq.Messages)
	return &azopenai.ChatCompletionsOptions{
		Messages: azureMessages,
	}
}

func AzureResponseToOpenAIResponse(input *azopenai.GetChatCompletionsResponse) *openai.ChatCompletionResponse {
	// 转换 Choices
	choices := make([]openai.ChatCompletionChoice, len(input.ChatCompletions.Choices))
	for i, choice := range input.ChatCompletions.Choices {
		choices[i] = openai.ChatCompletionChoice{
			Index: i,
			Message: openai.ChatCompletionMessage{
				Role:    safeString((*string)(choice.Message.Role)),
				Content: safeString(choice.Message.Content),
				Refusal: safeString(choice.Message.Refusal),
				//FunctionCall: choice.Message.FunctionCall,
				//ToolCalls: choice.Message.ToolCalls,
			},
			//Name:         choice.Message.Name,
			//LogProbs:             safeString(choice.LogProbs),
			//ContentFilterResults: choice.ContentFilterResults,
		}
	}

	// 转换 PromptFilterResults
	/*
		promptFilterResults := make([]openai.PromptFilterResult, len(input.ChatCompletions.PromptFilterResults))
		for i, result := range input.ChatCompletions.PromptFilterResults {
			promptFilterResults[i] = openai.PromptFilterResult{
				// 根据 PromptFilterResults 中字段对应的内容进行赋值
				// 示例: Prompt, Outcome 等字段
			}
		}

	*/

	// 转换 ChatCompletionResponse
	return &openai.ChatCompletionResponse{
		ID:      *input.ChatCompletions.ID,
		Object:  "chat.completion",
		Created: input.ChatCompletions.Created.Unix(),
		Model:   *input.ChatCompletions.Model,
		Choices: choices,
		Usage: openai.Usage{
			PromptTokens:     safeInt(input.ChatCompletions.Usage.PromptTokens),
			CompletionTokens: safeInt(input.ChatCompletions.Usage.CompletionTokens),
			TotalTokens:      safeInt(input.ChatCompletions.Usage.TotalTokens),
		},
		SystemFingerprint: *input.ChatCompletions.SystemFingerprint,
		//PromptFilterResults: promptFilterResults,
	}
}

func safeString(input *string) string {
	if input == nil {
		return ""
	}
	return *input
}

func safeInt(input *int32) int {
	if input == nil {
		return 0
	}
	return int(*input)
}
