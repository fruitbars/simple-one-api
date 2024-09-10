package adapter

import (
	"encoding/json"
	"github.com/fruitbars/gosparkclient"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"log"
	"simple-one-api/pkg/mylog"
	myopenai "simple-one-api/pkg/openai"
	"time"
)

func OpenAIRequestToXingHuoRequest(openAIReq *openai.ChatCompletionRequest) *gosparkclient.SparkChatRequest {
	var sparkChatReq gosparkclient.SparkChatRequest

	// 将 OpenAIRequest 的 Messages 转换为 SparkChatRequest 的 Message
	for _, msg := range openAIReq.Messages {
		sparkChatReq.Message = append(sparkChatReq.Message, struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	sparkChatReq.Topk = int(openAIReq.TopP)
	if sparkChatReq.Topk < 0 {
		sparkChatReq.Topk = 0
	}
	if sparkChatReq.Topk > 6 {
		sparkChatReq.Topk = 6
	}

	sparkChatReq.Temperature = float64(openAIReq.Temperature)
	if sparkChatReq.Temperature < 0 {
		sparkChatReq.Temperature = 0
	}
	if sparkChatReq.Temperature > 1 {
		sparkChatReq.Temperature = 1
	}

	sparkChatReq.Maxtokens = openAIReq.MaxTokens

	switch v := openAIReq.ToolChoice.(type) {
	case string:
		if v == "auto" {
			var xffunctions []*openai.FunctionDefinition
			for _, tool := range openAIReq.Tools {
				xffunctions = append(xffunctions, tool.Function)
			}
			for len(xffunctions) > 0 {
				toolsJsonData, err := json.Marshal(xffunctions)
				if err != nil {
					log.Println(err)
				}
				sparkChatReq.Functions = toolsJsonData
			}
		}

	case map[string]interface{}:
		mylog.Logger.Warn("ToolChoice is an object, ignore")
	default:
		mylog.Logger.Debug("Unhandled type, ignore", zap.Any("type", v))
	}

	return &sparkChatReq
}

func XingHuoResponseToOpenAIResponse(qfResp *gosparkclient.SparkAPIResponse) *myopenai.OpenAIResponse {
	openAIResp := &myopenai.OpenAIResponse{
		ID:                qfResp.Header.Sid,
		Object:            "text_completion",
		SystemFingerprint: qfResp.Header.Message,
	}

	// 转换 Choices
	for _, choice := range qfResp.Payload.Choices.Text {
		openAIResp.Choices = append(openAIResp.Choices, myopenai.Choice{
			Index: choice.Index,
			Message: myopenai.ResponseMessage{
				Role:    choice.Role,
				Content: choice.Content,
			},
		})
	}

	// 转换 Usage
	openAIResp.Usage = &myopenai.Usage{
		PromptTokens:     qfResp.Payload.Usage.Text.PromptTokens,
		CompletionTokens: qfResp.Payload.Usage.Text.CompletionTokens,
		TotalTokens:      qfResp.Payload.Usage.Text.TotalTokens,
	}

	// 设置 Created 时间为当前 Unix 时间戳（如果需要的话）
	openAIResp.Created = time.Now().Unix() // 你可以使用 time.Now().Unix() 设置为当前时间戳

	return openAIResp
}

func XingHuoResponseToOpenAIStreamResponse(qfResp *gosparkclient.SparkAPIResponse) *myopenai.OpenAIStreamResponse {
	openAIResp := &myopenai.OpenAIStreamResponse{
		ID:                qfResp.Header.Sid,
		Object:            "chat.completion.chunk",
		Created:           time.Now().Unix(), // 使用当前 Unix 时间戳
		SystemFingerprint: qfResp.Header.Message,
	}

	// 转换 Choices
	for _, choice := range qfResp.Payload.Choices.Text {
		openAIResp.Choices = append(openAIResp.Choices, struct {
			Index        int                    `json:"index"`
			Delta        myopenai.ResponseDelta `json:"delta,omitempty"`
			Logprobs     interface{}            `json:"logprobs,omitempty"`
			FinishReason interface{}            `json:"finish_reason,omitempty"`
		}{
			Index: choice.Index,
			Delta: myopenai.ResponseDelta{
				Role:    choice.Role,
				Content: choice.Content,
			},
		})
	}

	// 转换 Usage
	openAIResp.Usage = &myopenai.Usage{
		PromptTokens:     qfResp.Payload.Usage.Text.PromptTokens,
		CompletionTokens: qfResp.Payload.Usage.Text.CompletionTokens,
		TotalTokens:      qfResp.Payload.Usage.Text.TotalTokens,
	}

	return openAIResp
}
