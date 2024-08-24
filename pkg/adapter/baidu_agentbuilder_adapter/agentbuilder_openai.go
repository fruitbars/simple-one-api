package baidu_agentbuilder_adapter

import (
	"github.com/sashabaranov/go-openai"
	"simple-one-api/pkg/llm/devplatform/baidu_agentbuilder"
	myopenai "simple-one-api/pkg/openai"
	"time"
)

func AgentBuilderResponseToOpenAIResponse(abResp *baidu_agentbuilder.GetAnswerResponse) *myopenai.OpenAIResponse {
	openAIResp := &myopenai.OpenAIResponse{
		ID:     abResp.LogID,
		Object: "text_completion",
		//SystemFingerprint: qfResp.Header.Message,
	}

	// 转换 Choices
	for i := 0; i < len(abResp.Data.Content); i++ {
		openAIResp.Choices = append(openAIResp.Choices, myopenai.Choice{
			Index: 0,
			Message: myopenai.ResponseMessage{
				Role:    openai.ChatMessageRoleAssistant,
				Content: abResp.Data.Content[i].Data,
			},
		})
	}

	// 设置 Created 时间为当前 Unix 时间戳（如果需要的话）
	openAIResp.Created = time.Now().Unix() // 你可以使用 time.Now().Unix() 设置为当前时间戳

	return openAIResp
}

func AgentBuilderResponseToOpenAIStreamResponse(abStreamResp *baidu_agentbuilder.ConversationResponse) *myopenai.OpenAIStreamResponse {
	openAIResp := &myopenai.OpenAIStreamResponse{
		ID:      abStreamResp.LogID,
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(), // 使用当前 Unix 时间戳
		//SystemFingerprint: qfResp.Header.Message,
	}

	for i := 0; i < len(abStreamResp.Data.Message.Content); i++ {
		content := abStreamResp.Data.Message.Content[i]
		if content.DataType == "null" {
			continue
		}

		openAIResp.Choices = append(openAIResp.Choices, struct {
			Index        int                    `json:"index"`
			Delta        myopenai.ResponseDelta `json:"delta,omitempty"`
			Logprobs     interface{}            `json:"logprobs,omitempty"`
			FinishReason interface{}            `json:"finish_reason,omitempty"`
		}{
			Index: 0,
			Delta: myopenai.ResponseDelta{
				Role:    openai.ChatMessageRoleAssistant,
				Content: content.Data.Text,
			},
		})
	}

	return openAIResp
}
