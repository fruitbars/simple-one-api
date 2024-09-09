package adapter

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/sashabaranov/go-openai"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tchttp "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/http"
	hunyuan "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/hunyuan/v20230901"
	"go.uber.org/zap"
	"simple-one-api/pkg/mylog"
	myopenai "simple-one-api/pkg/openai"
	"simple-one-api/pkg/utils"
	"strings"
)

// ToolChoiceType 工具执行类型
type ToolChoiceType string

// 混元特有枚举值
const (
	None   ToolChoiceType = "none"
	Auto   ToolChoiceType = "auto"
	Custom ToolChoiceType = "custom"
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

		// 工具调用上下文字段
		hyToolCalls := make([]*hunyuan.ToolCall, len(tmpMsg.ToolCalls))
		for j, toolCall := range tmpMsg.ToolCalls {
			tmpToolCall := toolCall
			hyToolCalls[j] = &hunyuan.ToolCall{
				Id:   &tmpToolCall.ID,
				Type: (*string)(&tmpToolCall.Type),
				Function: &hunyuan.ToolCallFunction{
					Name:      &tmpToolCall.Function.Name,
					Arguments: &tmpToolCall.Function.Arguments,
				},
			}
		}

		request.Messages = append(request.Messages, &hunyuan.Message{
			Role:       &tmpMsg.Role,
			Content:    &tmpMsg.Content,
			ToolCallId: &tmpMsg.ToolCallID,
			ToolCalls:  hyToolCalls,
		})
	}

	topP := float64(oaiReq.TopP) // 将 *float32 转换为 float64
	request.TopP = &topP

	temperature := float64(oaiReq.Temperature) // 将 *float32 转换为 float64
	request.Temperature = &temperature

	request.Stream = &oaiReq.Stream

	// 工具定义字段
	hyTools := make([]*hunyuan.Tool, len(oaiReq.Tools))
	for i, oaiTool := range oaiReq.Tools {
		hyType := string(oaiTool.Type)
		oaFuncParamByte, _ := json.Marshal(oaiTool.Function.Parameters)
		hyFuncParam := string(oaFuncParamByte)
		hyTools[i] = &hunyuan.Tool{
			Type: &hyType,
			Function: &hunyuan.ToolFunction{
				Name:        &oaiTool.Function.Name,
				Description: &oaiTool.Function.Description,
				Parameters:  &hyFuncParam,
			},
		}
	}
	request.Tools = hyTools

	// 工具执行方式字段
	choiceType, toolChoice := convertHYToolChoice(oaiReq.ToolChoice)
	request.ToolChoice, request.CustomTool = (*string)(&choiceType), toolChoice

	return request
}

func convertHYToolChoice(oaiToolChoice interface{}) (ToolChoiceType, *hunyuan.Tool) {
	if oaiToolChoice == nil {
		return "", nil
	}
	functionKey := "function"

	switch tc := oaiToolChoice.(type) {
	case map[string]interface{}:
		choiceBytes, err := json.Marshal(tc)
		if err != nil {
			return "", nil
		}

		var choice openai.ToolChoice
		err = json.Unmarshal(choiceBytes, &choice)
		if err != nil {
			return "", nil
		}

		return Custom, &hunyuan.Tool{
			Type: &functionKey,
			Function: &hunyuan.ToolFunction{
				Name: &choice.Function.Name,
			},
		}
	case openai.ToolChoice:
		return Custom, &hunyuan.Tool{
			Type: &functionKey,
			Function: &hunyuan.ToolFunction{
				Name: &tc.Function.Name,
			},
		}
	case string:
		// 混元不支持any、require参数
		return ToolChoiceType(tc), nil
	default:
		return "", nil
	}
}

// 转换函数实现
func HunYuanResponseToOpenAIStreamResponse(event tchttp.SSEvent) (*myopenai.OpenAIStreamResponse, error) {
	var sResponse hunyuan.ChatCompletionsResponseParams
	if err := json.Unmarshal(event.Data, &sResponse); err != nil {
		return nil, err
	}

	id := event.Id
	if id == "" {
		id = uuid.New().String()
	}

	openAIResp := &myopenai.OpenAIStreamResponse{
		ID:      id,
		Created: utils.GetInt64(sResponse.Created),
	}
	openAIResp.Usage = &myopenai.Usage{
		PromptTokens:     int(utils.GetInt64(sResponse.Usage.PromptTokens)),
		CompletionTokens: int(utils.GetInt64(sResponse.Usage.CompletionTokens)),
		TotalTokens:      int(utils.GetInt64(sResponse.Usage.TotalTokens)),
	}
	for _, choice := range sResponse.Choices {
		openAIResp.Choices = append(openAIResp.Choices, myopenai.OpenAIStreamResponseChoice{
			Delta:        convertHYDelta(*choice.Delta),
			FinishReason: utils.GetString(choice.FinishReason),
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
			Message: convertHYMessage(*choice.Message),
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
func convertHYMessage(hunyuanMessage hunyuan.Message) myopenai.ResponseMessage {
	toolCalls := make([]myopenai.ToolCall, len(hunyuanMessage.ToolCalls))
	for i, _ := range hunyuanMessage.ToolCalls {
		index := i
		call := hunyuanMessage.ToolCalls[index]
		toolCalls[index] = myopenai.ToolCall{
			Index: &index,
			ID:    utils.GetString(call.Id),
			Type:  myopenai.ToolType(utils.GetString(call.Type)),
			Function: myopenai.FunctionCall{
				Name:      utils.GetString(call.Function.Name),
				Arguments: utils.GetString(call.Function.Arguments),
			},
		}
	}

	return myopenai.ResponseMessage{
		Role:       utils.GetString(hunyuanMessage.Role),
		Content:    utils.GetString(hunyuanMessage.Content),
		ToolCalls:  toolCalls,
		ToolCallID: utils.GetString(hunyuanMessage.ToolCallId),
	}
}

// 辅助函数：转换Message
func convertHYDelta(hunyuanDelta hunyuan.Delta) myopenai.ResponseDelta {
	toolCalls := make([]myopenai.ToolCall, len(hunyuanDelta.ToolCalls))
	for i, _ := range hunyuanDelta.ToolCalls {
		index := i
		call := hunyuanDelta.ToolCalls[index]
		toolCalls[index] = myopenai.ToolCall{
			Index: &index,
			ID:    utils.GetString(call.Id),
			Type:  myopenai.ToolType(utils.GetString(call.Type)),
			Function: myopenai.FunctionCall{
				Name:      utils.GetString(call.Function.Name),
				Arguments: utils.GetString(call.Function.Arguments),
			},
		}
	}

	return myopenai.ResponseDelta{
		Role:      utils.GetString(hunyuanDelta.Role),
		Content:   utils.GetString(hunyuanDelta.Content),
		ToolCalls: toolCalls,
	}
}
