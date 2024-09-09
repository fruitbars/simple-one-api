package adapter

import (
	"simple-one-api/pkg/llm/claude"
	"simple-one-api/pkg/mycomdef"
	myopenai "simple-one-api/pkg/openai"
)

// 将 MsgMessageStart 转换为 OpenAIStreamResponse 的函数
func ConvertMsgMessageStartToOpenAIStreamResponse(msg *claude.MsgMessageStart) *myopenai.OpenAIStreamResponse {
	response := &myopenai.OpenAIStreamResponse{
		ID:    msg.Message.ID,
		Model: msg.Message.Model,
		Usage: &myopenai.Usage{
			PromptTokens:     msg.Message.Usage.InputTokens,
			CompletionTokens: msg.Message.Usage.OutputTokens,
			TotalTokens:      msg.Message.Usage.InputTokens + msg.Message.Usage.OutputTokens,
		},
		Choices: []myopenai.OpenAIStreamResponseChoice{
			{
				Delta: myopenai.ResponseDelta{
					Role:    msg.Message.Role,
					Content: "", // 因为原数据中的 content 是一个空数组
				},
			},
		},
	}
	return response
}

func ConvertMsgContentBlockDeltaToOpenAIStreamResponse(msg *claude.MsgContentBlockDelta) *myopenai.OpenAIStreamResponse {
	return &myopenai.OpenAIStreamResponse{
		Choices: []myopenai.OpenAIStreamResponseChoice{
			{
				Index: msg.Index,
				Delta: myopenai.ResponseDelta{
					Role:    mycomdef.KEYNAME_ASSISTANT,
					Content: msg.Delta.Text,
				},
			},
		},
	}
}
