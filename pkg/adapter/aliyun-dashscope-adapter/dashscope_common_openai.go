package aliyun_dashscope_adapter

import (
	"github.com/sashabaranov/go-openai"
	"simple-one-api/pkg/llm/aliyun-dashscope/commsg/ds_com_request"
	"simple-one-api/pkg/llm/aliyun-dashscope/commsg/ds_com_resp"
	myopenai "simple-one-api/pkg/openai"
	"strings"
	"time"
)

func OpenAIRequestToDashScopeCommonRequest(oaiReq *openai.ChatCompletionRequest) *ds_com_request.ModelRequest {
	var dsComReq ds_com_request.ModelRequest

	dsComReq.Model = oaiReq.Model

	for _, msg := range oaiReq.Messages {

		var dsComMsg ds_com_request.Message

		dsComMsg.Role = msg.Role
		dsComMsg.Content = msg.Content

		dsComReq.Input.Messages = append(dsComReq.Input.Messages, dsComMsg)

	}

	var param ds_com_request.Parameters
	param.ResultFormat = "message"

	dsComReq.Parameters = &param

	return &dsComReq
}

func DashScopeCommonResponseToOpenAIResponse(dsComResp *ds_com_resp.ModelResponse) *myopenai.OpenAIResponse {
	if dsComResp == nil {
		return nil
	}

	var oaiChoices []myopenai.Choice

	for _, choice := range dsComResp.Output.Choices {
		var oaiChoice myopenai.Choice
		oaiChoice.Message.Role = choice.Message.Role
		oaiChoice.Message.Content = choice.Message.Content
		oaiChoices = append(oaiChoices, oaiChoice)
	}

	usage := &myopenai.Usage{
		PromptTokens:     dsComResp.Usage.InputTokens,
		CompletionTokens: dsComResp.Usage.OutputTokens,
		TotalTokens:      dsComResp.Usage.InputTokens + dsComResp.Usage.OutputTokens,
	}

	return &myopenai.OpenAIResponse{
		ID:      dsComResp.RequestID,
		Created: time.Now().Unix(),
		Model:   "",
		Choices: oaiChoices,
		Usage:   usage,
	}
}

func compareAndExtractDelta(prev, current string) string {
	if prev == "" {
		return current
	}

	if strings.HasPrefix(current, prev) {
		return strings.TrimPrefix(current, prev)
	}

	return current
}

func GetStreamResponseContent(dsResp *ds_com_resp.ModelStreamResponse) string {
	if dsResp == nil || len(dsResp.Output.Choices) == 0 {
		return ""
	}

	choice := dsResp.Output.Choices[0]

	return choice.Message.Content

}

func DashScopeCommonResponseToOpenAIStreamResponse(dsResp *ds_com_resp.ModelStreamResponse, prevContent string) *myopenai.OpenAIStreamResponse {
	openAIResp := &myopenai.OpenAIStreamResponse{
		ID:      dsResp.RequestID,
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(), // 使用当前 Unix 时间戳
	}

	for _, dsChoice := range dsResp.Output.Choices {
		deltaContent := compareAndExtractDelta(prevContent, dsChoice.Message.Content)
		choice := struct {
			Index        int                    `json:"index"`
			Delta        myopenai.ResponseDelta `json:"delta,omitempty"`
			Logprobs     interface{}            `json:"logprobs,omitempty"`
			FinishReason interface{}            `json:"finish_reason,omitempty"`
		}{
			Index: 0,
			Delta: myopenai.ResponseDelta{
				Role:    dsChoice.Message.Role,
				Content: deltaContent,
			},
		}

		openAIResp.Choices = append(openAIResp.Choices, choice)
	}

	// 转换 Usage
	openAIResp.Usage = &myopenai.Usage{
		PromptTokens:     dsResp.Usage.Kens,
		CompletionTokens: dsResp.Usage.OutputTokens,
		TotalTokens:      dsResp.Usage.Kens + dsResp.Usage.OutputTokens,
	}

	return openAIResp
}
