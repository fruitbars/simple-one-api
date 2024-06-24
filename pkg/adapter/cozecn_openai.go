package adapter

import (
	"fmt"
	"github.com/sashabaranov/go-openai"
	"simple-one-api/pkg/common"
	"simple-one-api/pkg/llm/devplatform/cozecn"
	myopenai "simple-one-api/pkg/openai"
	"strings"
	"time"
)

func OpenAIRequestToCozecnRequest(oaiReq openai.ChatCompletionRequest) *cozecn.CozeRequest {

	hisMessages := common.ConvertSystemMessages2NoSystem(oaiReq.Messages)

	cozeMessages := make([]cozecn.Message, 0, len(hisMessages)-1)
	query := oaiReq.Messages[len(hisMessages)-1].Content // 最后一条消息作为查询

	for i := 0; i < len(hisMessages)-1; i++ {
		msg := oaiReq.Messages[i]
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

	user := oaiReq.User
	if user == "" {
		user = "12345678"
	}

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
			Delta: struct {
				Role    string `json:"role,omitempty"`
				Content string `json:"content,omitempty"`
			}{
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
