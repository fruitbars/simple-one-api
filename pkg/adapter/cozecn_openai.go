package adapter

import (
	"fmt"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"simple-one-api/pkg/llm/devplatform/cozecn"
	"simple-one-api/pkg/mycommon"
	"simple-one-api/pkg/mylog"
	myopenai "simple-one-api/pkg/openai"
	"strings"
	"time"
)

func OpenAIRequestToCozecnRequest(oaiReq *openai.ChatCompletionRequest) *cozecn.CozeRequest {
	hisMessages := mycommon.ConvertSystemMessages2NoSystem(oaiReq.Messages)
	messageCount := len(hisMessages)

	// Directly get the content of the last message as the query
	query := hisMessages[messageCount-1].Content

	var cozeMessages []cozecn.Message
	if messageCount > 1 {
		// Iterate through the messages except the last one
		for i := 0; i < messageCount-1; i++ {
			msg := hisMessages[i] // Corrected to use hisMessages instead of oaiReq.Messages
			mt := ""
			if strings.ToLower(msg.Role) == "assistant" {
				mt = "answer"
			}

			cozeMessages = append(cozeMessages, cozecn.Message{
				Role:        msg.Role,
				Type:        mt,
				Content:     msg.Content,
				ContentType: "text",
			})
		}
	}

	// Set a default user if the user field is empty
	user := oaiReq.User
	if user == "" {
		user = "12345678"
	}

	mylog.Logger.Debug("cozeMessages", zap.Any("cozeMessages", cozeMessages))

	return &cozecn.CozeRequest{
		ConversationID: "123",        // Assuming a static conversation ID
		BotID:          oaiReq.Model, // Assuming the model as the bot ID
		User:           user,
		Query:          query,
		Stream:         oaiReq.Stream,
		ChatHistory:    cozeMessages,
	}
}

func CozecnReponseToOpenAIResponse(resp *cozecn.Response) *myopenai.OpenAIResponse {
	if resp.Code != 0 {
		return &myopenai.OpenAIResponse{
			ID: resp.ConversationID,
			Error: &myopenai.ErrorDetail{
				Message: resp.Msg,
				Code:    resp.Code,
			},
		}
	}

	choices := make([]myopenai.Choice, len(resp.Messages))
	for i, msg := range resp.Messages {
		if msg.Type == "verbose" {
			continue
		}

		choices[i] = myopenai.Choice{
			Index: i,
			Message: myopenai.ResponseMessage{
				Role:    msg.Role,
				Content: msg.Content,
			},
			FinishReason: "stop", // Assuming all responses are finished
		}
	}

	return &myopenai.OpenAIResponse{
		ID:      resp.ConversationID,
		Object:  "text_completion",
		Created: time.Now().Unix(),
		Choices: choices,
		Error: func() *myopenai.ErrorDetail {
			if resp.Code != 200 {
				return &myopenai.ErrorDetail{
					Code:    fmt.Sprintf("%d", resp.Code),
					Message: resp.Msg,
				}
			}
			return nil
		}(),
	}
}

func CozecnReponseToOpenAIResponseStream(resp *cozecn.StreamResponse) *myopenai.OpenAIStreamResponse {
	var choices []myopenai.OpenAIStreamResponseChoice

	if resp.Event == "message" {
		choices = append(choices, myopenai.OpenAIStreamResponseChoice{
			Index: resp.Index,
			Delta: myopenai.ResponseDelta{
				Role:    resp.Message.Role,
				Content: resp.Message.Content,
			},
		})
	}

	var errorDetail *myopenai.ErrorDetail
	if resp.Event == "error" {
		errorDetail = &myopenai.ErrorDetail{
			Message: resp.ErrorInformation.Msg,
			Code:    resp.ErrorInformation.Code,
		}
	}

	return &myopenai.OpenAIStreamResponse{
		ID:      resp.ConversationID,
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Choices: choices,
		Error:   errorDetail,
	}
}
