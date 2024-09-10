package aliyun_dashscope_adapter

import (
	"github.com/sashabaranov/go-openai"
	"simple-one-api/pkg/llm/aliyun-dashscope/common_btype"
	"simple-one-api/pkg/mycomdef"
	"simple-one-api/pkg/mycommon"
	myopenai "simple-one-api/pkg/openai"
	"time"
)

func OpenAIRequestToDashScopeBTypeRequest(oaiReq *openai.ChatCompletionRequest) *common_btype.DSBtypeRequestBody {
	var dsReq common_btype.DSBtypeRequestBody

	dsReq.Model = oaiReq.Model

	systemContent := mycommon.GetSystemMessage(oaiReq.Messages)

	if len(systemContent) > 0 {
		dsReq.Input.Prompt = systemContent + "\n"
	}

	dsReq.Input.Prompt += mycommon.GetLastestMessage(oaiReq.Messages)

	return &dsReq
}

func DashScopeBTypeResponseToOpenAIResponse(llamaResp *common_btype.DSBtypeResponseBody) *myopenai.OpenAIResponse {
	if llamaResp == nil {
		return nil
	}

	choices := []myopenai.Choice{
		{
			Index: 0,
			Message: myopenai.ResponseMessage{
				Role:    mycomdef.KEYNAME_ASSISTANT,
				Content: llamaResp.Output.Text,
			},
			//FinishReason: determineFinishReason(resp.Done),
		},
	}

	usage := &myopenai.Usage{
		PromptTokens:     llamaResp.Usage.InputTokens,
		CompletionTokens: llamaResp.Usage.OutputTokens,
		TotalTokens:      llamaResp.Usage.InputTokens + llamaResp.Usage.OutputTokens,
	}

	return &myopenai.OpenAIResponse{
		ID:      llamaResp.RequestID,
		Created: time.Now().Unix(),
		Model:   "",
		Choices: choices,
		Usage:   usage,
	}
}

func DashScopeBTypeResponseToOpenAIStreamResponse(dsResp *common_btype.DSBtypeResponseBody) *myopenai.OpenAIStreamResponse {
	openAIResp := &myopenai.OpenAIStreamResponse{
		ID:      dsResp.RequestID,
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(), // 使用当前 Unix 时间戳
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
			Content: dsResp.Output.Text,
		},
	})

	// 转换 Usage
	openAIResp.Usage = &myopenai.Usage{
		PromptTokens:     dsResp.Usage.InputTokens,
		CompletionTokens: dsResp.Usage.OutputTokens,
		TotalTokens:      dsResp.Usage.InputTokens + dsResp.Usage.OutputTokens,
	}

	return openAIResp
}
