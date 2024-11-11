package adapter

import (
	"github.com/sashabaranov/go-openai"
	"simple-one-api/pkg/llm/devplatform/dify/chat_completion_response"
	"simple-one-api/pkg/llm/devplatform/dify/chunk_chat_completion_response"
	"simple-one-api/pkg/mycommon"
	myopenai "simple-one-api/pkg/openai"
	"time"
)
import "simple-one-api/pkg/llm/devplatform/dify/chat_message_request"

func OpenAIRequestToDifyRequest(oaiReq *openai.ChatCompletionRequest) *chat_message_request.ChatMessageRequest {
	var difyReq chat_message_request.ChatMessageRequest
	difyReq.Query = mycommon.GetLastestMessage(oaiReq.Messages)
	if oaiReq.Stream {
		difyReq.ResponseMode = "streaming"
	} else {
		difyReq.ResponseMode = "blocking"
	}

	difyReq.User = oaiReq.User

	if difyReq.User == "" {
		difyReq.User = "abc-123"
	}

	return &difyReq
}

func DifyResponseToOpenAIResponse(difyResp *chat_completion_response.ChatCompletionResponse) *openai.ChatCompletionResponse {
	var oaiResp openai.ChatCompletionResponse

	oaiResp.ID = difyResp.MessageID
	oaiResp.Object = "chat.completion"
	oaiResp.Created = difyResp.CreatedAt.Unix()
	//oaiResp.Model = difyResp.Model
	//oaiResp.Choices = difyResp.Choices

	var choice openai.ChatCompletionChoice
	choice.Message = openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleAssistant,
		Content: difyResp.Answer,
	}

	oaiResp.Choices = append(oaiResp.Choices, choice)

	return &oaiResp
}

func DifyResponseToOpenAIResponseStream(difyResp *chunk_chat_completion_response.MessageEvent) *myopenai.OpenAIStreamResponse {
	var oaiStreamResp myopenai.OpenAIStreamResponse

	oaiStreamResp.Choices = []myopenai.OpenAIStreamResponseChoice{
		{
			Delta: myopenai.ResponseDelta{
				Role:    openai.ChatMessageRoleAssistant,
				Content: difyResp.Answer,
			},
		},
	}

	return &oaiStreamResp
}

func DifyMessageEndEventToOpenAIResponseStream(difyResp *chunk_chat_completion_response.MessageEndEvent) *myopenai.OpenAIStreamResponse {
	if difyResp == nil {
		return nil
	}

	var oaiuasge myopenai.Usage

	oaiuasge.PromptTokens = difyResp.Metadata.Usage.PromptTokens
	oaiuasge.CompletionTokens = difyResp.Metadata.Usage.CompletionTokens
	oaiuasge.TotalTokens = difyResp.Metadata.Usage.TotalTokens

	return &myopenai.OpenAIStreamResponse{
		ID:      difyResp.ID,
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		//Error:   errorDetail,
		Usage: &oaiuasge,
	}
}
