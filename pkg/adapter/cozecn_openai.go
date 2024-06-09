package adapter

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/sashabaranov/go-openai"
	"simple-one-api/pkg/devplatform/cozecn"
	myopenai "simple-one-api/pkg/openai"
	"strings"
	"time"
)

func OpenAIRequestToCozecnRequest(oaiReq openai.ChatCompletionRequest) *cozecn.CozeRequest {
	var query string
	var cozeMessages []cozecn.Message

	// Check if there are any messages
	if len(oaiReq.Messages) > 0 {
		hisMessagesLen := len(oaiReq.Messages)
		// If the first message is of role "system", skip it
		startIndex := 0
		if strings.ToLower(oaiReq.Messages[0].Role) == "system" {
			startIndex = 1

			query = oaiReq.Messages[0].Content + "\n"

			hisMessagesLen--
		}

		// Get the last message and use it as the query
		lastMsg := oaiReq.Messages[len(oaiReq.Messages)-1]
		query += lastMsg.Content
		hisMessagesLen--

		if hisMessagesLen > 0 {
			hisMessages := oaiReq.Messages[startIndex : hisMessagesLen-1]

			// Convert all previous messages to coze messages
			if len(hisMessages) > 1 {
				cozeMessages = make([]cozecn.Message, hisMessagesLen)
				for i, msg := range hisMessages {
					mt := ""
					if strings.ToLower(msg.Role) == "assistant" {
						mt = "answer"
					}

					cozeMessages[i] = cozecn.Message{
						Role:        msg.Role,
						Type:        mt,
						Content:     msg.Content,
						ContentType: "text",
					}
				}
			}
		}

	}

	user := oaiReq.User
	if user == "" {
		user = uuid.New().String()
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
		Created: int(time.Now().Unix()),
		Choices: choices,
		Error:   errorDetail,
	}
}
