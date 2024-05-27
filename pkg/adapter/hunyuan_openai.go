package adapter

import (
	"encoding/json"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tchttp "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/http"
	hunyuan "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/hunyuan/v20230901"
	"log"
	tecent_hunyuan "simple-one-api/pkg/llm/tecent-hunyuan"
	"simple-one-api/pkg/openai"
	"simple-one-api/pkg/utils"
)

func OpenAIRequestToHunYuanRequest(oaiReq openai.OpenAIRequest) *hunyuan.ChatCompletionsRequest {
	request := hunyuan.NewChatCompletionsRequest()

	request.Model = common.StringPtr(oaiReq.Model)

	log.Println(oaiReq.Messages)
	for _, msg := range oaiReq.Messages {
		tmpMsg := msg

		request.Messages = append(request.Messages, &hunyuan.Message{
			Role:    &tmpMsg.Role,
			Content: &tmpMsg.Content,
		})
	}

	if oaiReq.TopP != nil {
		topP := float64(*oaiReq.TopP) // 将 *float32 转换为 float64
		request.TopP = &topP
	}

	if oaiReq.Temperature != nil {
		temperature := float64(*oaiReq.Temperature) // 将 *float32 转换为 float64
		request.Temperature = &temperature
	}
	if oaiReq.Stream != nil {
		request.Stream = oaiReq.Stream
	}

	return request
}

// 转换函数实现
func HunYuanResponseToOpenAIStreamResponse(event tchttp.SSEvent) (*openai.OpenAIStreamResponse, error) {

	var sResponse tecent_hunyuan.StreamResponse
	json.Unmarshal(event.Data, &sResponse)

	//common.
	openAIResp := &openai.OpenAIStreamResponse{
		ID:      event.Id,
		Created: sResponse.Created,
		//Usage:   sResponse.Usage,
		//Error: sResponse.e,
	}
	openAIResp.Usage = &openai.Usage{
		PromptTokens:     sResponse.Usage.PromptTokens,
		CompletionTokens: sResponse.Usage.CompletionTokens,
		TotalTokens:      sResponse.Usage.TotalTokens,
	}

	for _, choice := range sResponse.Choices {
		openAIResp.Choices = append(openAIResp.Choices, struct {
			Index int `json:"index,omitempty"`
			Delta struct {
				Role    string `json:"role,omitempty"`
				Content string `json:"content,omitempty"`
			} `json:"delta,omitempty"`
			Logprobs     interface{} `json:"logprobs,omitempty"`
			FinishReason interface{} `json:"finish_reason,omitempty"`
		}{
			Index: 0,
			Delta: struct {
				Role    string `json:"role,omitempty"`
				Content string `json:"content,omitempty"`
			}{
				Role:    choice.Delta.Role,
				Content: choice.Delta.Content,
			},
			//Logprobs:     choice.LogProbs,
			FinishReason: choice.FinishReason,
		})
	}

	return openAIResp, nil
}

// 转换函数实现
func HunYuanResponseToOpenAIResponse(qfResp *hunyuan.ChatCompletionsResponse) *openai.OpenAIResponse {
	if qfResp == nil || qfResp.Response == nil {
		return nil
	}

	// 初始化OpenAIResponse
	openAIResp := &openai.OpenAIResponse{
		ID:      utils.GetString(qfResp.Response.Id),
		Created: utils.GetInt64(qfResp.Response.Created),
		Usage:   convertUsage(qfResp.Response.Usage),
		Error:   convertError(qfResp.Response.ErrorMsg),
	}

	// 转换Choices
	for _, choice := range qfResp.Response.Choices {
		openAIResp.Choices = append(openAIResp.Choices, openai.Choice{
			//Index:   choice.Index,
			Message: convertMessage(*choice.Message),
			//LogProbs:     choice.LogProbs,
			FinishReason: utils.GetString(choice.FinishReason),
		})
	}

	return openAIResp
}

// 辅助函数：转换Usage
func convertUsage(hunyuanUsage *hunyuan.Usage) *openai.Usage {
	if hunyuanUsage == nil {
		return nil
	}
	return &openai.Usage{
		PromptTokens:     int(utils.GetInt64(hunyuanUsage.PromptTokens)),
		CompletionTokens: int(utils.GetInt64(hunyuanUsage.CompletionTokens)),
		TotalTokens:      int(utils.GetInt64(hunyuanUsage.TotalTokens)),
	}
}

// 辅助函数：转换ErrorMsg
func convertError(hunyuanError *hunyuan.ErrorMsg) *openai.ErrorDetail {
	if hunyuanError == nil {
		return nil
	}
	return &openai.ErrorDetail{
		Message: utils.GetString(hunyuanError.Msg),
		//Type:    hunyuanError.Type,
		//Param: hunyuanError.Param,
		Code: hunyuanError.Code,
	}
}

// 辅助函数：转换Message
func convertMessage(hunyuanMessage hunyuan.Message) openai.ResponseMessage {
	return openai.ResponseMessage{
		Role:    utils.GetString(hunyuanMessage.Role),
		Content: utils.GetString(hunyuanMessage.Content),
	}
}
