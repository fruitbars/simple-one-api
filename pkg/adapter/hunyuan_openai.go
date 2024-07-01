package adapter

import (
	"encoding/json"
	"github.com/sashabaranov/go-openai"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tchttp "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/http"
	hunyuan "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/hunyuan/v20230901"
	"go.uber.org/zap"
	tecenthunyuan "simple-one-api/pkg/llm/tecent-hunyuan"
	"simple-one-api/pkg/mylog"
	myopenai "simple-one-api/pkg/openai"
	"simple-one-api/pkg/utils"
	"strings"
)

func OpenAIRequestToHunYuanRequest(oaiReq *openai.ChatCompletionRequest) *hunyuan.ChatCompletionsRequest {
	request := hunyuan.NewChatCompletionsRequest()

	model := oaiReq.Model
	request.Model = common.StringPtr(model)

	mylog.Logger.Info("messages", zap.Any("oaiReq.Messages", oaiReq.Messages))

	for i, msg := range oaiReq.Messages {
		//超级对齐，多余的system直接删除
		if strings.ToLower(msg.Role) == "system" && i > 0 {
			continue
		}

		tmpMsg := msg

		request.Messages = append(request.Messages, &hunyuan.Message{
			Role:    &tmpMsg.Role,
			Content: &tmpMsg.Content,
		})
	}

	topP := float64(oaiReq.TopP) // 将 *float32 转换为 float64
	request.TopP = &topP

	temperature := float64(oaiReq.Temperature) // 将 *float32 转换为 float64
	request.Temperature = &temperature

	request.Stream = &oaiReq.Stream

	return request
}

// 转换函数实现
func HunYuanResponseToOpenAIStreamResponse(event tchttp.SSEvent) (*myopenai.OpenAIStreamResponse, error) {

	var sResponse tecenthunyuan.StreamResponse
	json.Unmarshal(event.Data, &sResponse)

	//common.
	openAIResp := &myopenai.OpenAIStreamResponse{
		ID:      event.Id,
		Created: sResponse.Created,
		//Usage:   sResponse.Usage,
		//Error: sResponse.e,
	}
	openAIResp.Usage = &myopenai.Usage{
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
func HunYuanResponseToOpenAIResponse(qfResp *hunyuan.ChatCompletionsResponse) *myopenai.OpenAIResponse {
	if qfResp == nil || qfResp.Response == nil {
		return nil
	}

	// 初始化OpenAIResponse
	openAIResp := &myopenai.OpenAIResponse{
		ID:      utils.GetString(qfResp.Response.Id),
		Created: utils.GetInt64(qfResp.Response.Created),
		Usage:   convertUsage(qfResp.Response.Usage),
		Error:   convertError(qfResp.Response.ErrorMsg),
	}

	// 转换Choices
	for _, choice := range qfResp.Response.Choices {
		openAIResp.Choices = append(openAIResp.Choices, myopenai.Choice{
			//Index:   choice.Index,
			Message: convertMessage(*choice.Message),
			//LogProbs:     choice.LogProbs,
			FinishReason: utils.GetString(choice.FinishReason),
		})
	}

	return openAIResp
}

// 辅助函数：转换Usage
func convertUsage(hunyuanUsage *hunyuan.Usage) *myopenai.Usage {
	if hunyuanUsage == nil {
		return nil
	}
	return &myopenai.Usage{
		PromptTokens:     int(utils.GetInt64(hunyuanUsage.PromptTokens)),
		CompletionTokens: int(utils.GetInt64(hunyuanUsage.CompletionTokens)),
		TotalTokens:      int(utils.GetInt64(hunyuanUsage.TotalTokens)),
	}
}

// 辅助函数：转换ErrorMsg
func convertError(hunyuanError *hunyuan.ErrorMsg) *myopenai.ErrorDetail {
	if hunyuanError == nil {
		return nil
	}
	return &myopenai.ErrorDetail{
		Message: utils.GetString(hunyuanError.Msg),
		//Type:    hunyuanError.Type,
		//Param: hunyuanError.Param,
		Code: hunyuanError.Code,
	}
}

// 辅助函数：转换Message
func convertMessage(hunyuanMessage hunyuan.Message) myopenai.ResponseMessage {
	return myopenai.ResponseMessage{
		Role:    utils.GetString(hunyuanMessage.Role),
		Content: utils.GetString(hunyuanMessage.Content),
	}
}
