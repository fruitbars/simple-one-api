package adapter

import (
	"encoding/json"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"simple-one-api/pkg/llm/devplatform/cozecn_v3/common"
	"simple-one-api/pkg/llm/devplatform/cozecn_v3/nonestream/chat_message_list"
	"simple-one-api/pkg/llm/devplatform/cozecn_v3/streammode"
	"simple-one-api/pkg/mycommon"
	"simple-one-api/pkg/mylog"
	myopenai "simple-one-api/pkg/openai"
	"time"
)

func OpenAIRequestToCozecnV3Request(oaiReq *openai.ChatCompletionRequest) *common.ChatRequest {
	hisMessages := mycommon.ConvertSystemMessages2NoSystem(oaiReq.Messages)
	messageCount := len(hisMessages)

	// 提前处理默认用户ID
	user := oaiReq.User
	if user == "" {
		user = "12345678"
	}

	cozev3Messages := buildCozeV3Messages(hisMessages)

	mylog.Logger.Debug("cozev3Messages", zap.Int("messageCount", messageCount), zap.Any("cozev3Messages", cozev3Messages), zap.Any("hisMessages", hisMessages))

	return &common.ChatRequest{
		BotID:              oaiReq.Model,
		UserID:             user,
		Stream:             oaiReq.Stream,
		AutoSaveHistory:    true,
		AdditionalMessages: cozev3Messages,
	}
}

// 独立函数处理消息构建逻辑
func buildCozeV3Messages(messages []openai.ChatCompletionMessage) []common.Message {
	var cozev3Messages []common.Message

	for _, msg := range messages {
		cozev3msg := common.Message{
			Role:        msg.Role,
			Content:     msg.Content,
			ContentType: "text",
		}

		if len(msg.MultiContent) > 0 {
			// 处理多内容消息
			mcContentMsg := buildMultiContentMessages(msg.MultiContent)
			content, err := json.Marshal(mcContentMsg)
			if err != nil {
				mylog.Logger.Error("Error marshalling multi-content message", zap.Error(err))
				continue // 跳过错误消息
			}

			cozev3msg.Content = string(content)
			cozev3msg.ContentType = "object_string"
		}

		cozev3Messages = append(cozev3Messages, cozev3msg)
	}

	mylog.Logger.Debug("buildCozeV3Messages", zap.Any("cozev3Messages len", len(cozev3Messages)))

	return cozev3Messages
}

// 处理多内容消息的构建
func buildMultiContentMessages(multiContent []openai.ChatMessagePart) []common.ObjectStringMessage {
	var mcContentMsg []common.ObjectStringMessage

	for _, mc := range multiContent {
		switch mc.Type {
		case openai.ChatMessagePartTypeText:
			mcContentMsg = append(mcContentMsg, common.ObjectStringMessage{
				Type: "text",
				Text: mc.Text,
			})
		case openai.ChatMessagePartTypeImageURL:
			mcContentMsg = append(mcContentMsg, common.ObjectStringMessage{
				Type:    "image_url",
				FileURL: mc.ImageURL.URL,
			})
		}
	}

	return mcContentMsg
}

func CozecnV3ReponseToOpenAIResponse(resp *chat_message_list.MessageListResponse) *myopenai.OpenAIResponse {

	var choices []myopenai.Choice

	// 提前分配 choices 的容量，假设消息量较小，可以动态增长
	choices = make([]myopenai.Choice, 0, len(resp.Data))

	var id string

	for i, msg := range resp.Data {
		// 获取ID并构建 Choice
		if id == "" {
			id = msg.ID
		}

		if msg.Type != "answer" {
			continue // 直接跳过非 "answer" 类型的消息
		}

		choices = append(choices, myopenai.Choice{
			Index: i,
			Message: myopenai.ResponseMessage{
				Role:    msg.Role,
				Content: msg.Content,
			},
		})
	}

	// 设置最后一个 choice 的 FinishReason
	if len(choices) > 0 {
		choices[len(choices)-1].FinishReason = "stop"
	}

	// 构建并返回 OpenAIResponse
	return &myopenai.OpenAIResponse{
		ID:      id,
		Object:  "text_completion",
		Created: time.Now().Unix(),
		Choices: choices,
	}

}

func CozecnV3ReponseToOpenAIResponseStream(resp *streammode.EventData) *myopenai.OpenAIStreamResponse {
	var choices []myopenai.OpenAIStreamResponseChoice

	choices = append(choices, myopenai.OpenAIStreamResponseChoice{
		Index: 0,
		Delta: myopenai.ResponseDelta{
			Role:    resp.Role,
			Content: resp.Content,
		},
	})
	usage := myopenai.Usage{
		PromptTokens:     resp.Usage.InputCount,
		CompletionTokens: resp.Usage.OutputCount,
		TotalTokens:      resp.Usage.TokenCount,
	}
	return &myopenai.OpenAIStreamResponse{
		ID:      resp.ID,
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Choices: choices,
		//Error:   errorDetail,
		Usage: &usage,
	}
	return nil
}
