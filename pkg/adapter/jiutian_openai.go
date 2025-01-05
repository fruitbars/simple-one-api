package adapter

import (
	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"simple-one-api/pkg/llm/jiutian"
	"simple-one-api/pkg/mylog"
	"time"
)

// convertFinishReason 将字符串转换为OpenAI的FinishReason类型
func convertFinishReason(reason string) openai.FinishReason {
	switch reason {
	case "stop":
		return openai.FinishReasonStop
	case "length":
		return openai.FinishReasonLength
	case "content_filter":
		return openai.FinishReasonContentFilter
	case "function_call":
		return openai.FinishReasonFunctionCall
	default:
		return openai.FinishReasonNull
	}
}

// OpenAIRequestToJiuTianRequest 将OpenAI请求转换为九天模型请求
func OpenAIRequestToJiuTianRequest(oaiReq *openai.ChatCompletionRequest) *jiutian.ChatCompletionRequest {
	mylog.Logger.Info("Converting OpenAI request to JiuTian request",
		zap.String("model", oaiReq.Model),
		zap.Int("message_count", len(oaiReq.Messages)),
		zap.Float32("temperature", oaiReq.Temperature),
		zap.Float32("top_p", oaiReq.TopP))

	// 获取最后一条消息作为prompt
	lastMessage := oaiReq.Messages[len(oaiReq.Messages)-1].Content
	
	// 构建历史消息
	var history [][]string
	if len(oaiReq.Messages) > 1 {
		for i := 0; i < len(oaiReq.Messages)-1; i += 2 {
			if i+1 < len(oaiReq.Messages) {
				history = append(history, []string{
					oaiReq.Messages[i].Content,
					oaiReq.Messages[i+1].Content,
				})
			}
		}
	}

	mylog.Logger.Debug("Request conversion details",
		zap.String("prompt", lastMessage),
		zap.Int("history_length", len(history)))

	// 创建请求
	req := jiutian.NewChatCompletionRequest().
		WithModelID(oaiReq.Model). // 使用传入的模型ID
		WithPrompt(lastMessage).
		WithHistory(history).
		WithStream(oaiReq.Stream)

	// 设置温度参数（如果有）
	if oaiReq.Temperature > 0 {
		req.WithTemperature(oaiReq.Temperature)
	}

	// 设置top_p参数（如果有）
	if oaiReq.TopP > 0 {
		req.WithTopP(oaiReq.TopP)
	}

	mylog.Logger.Debug("Created JiuTian request",
		zap.String("model_id", req.ModelID),
		zap.Bool("stream", req.Stream),
		zap.Float32("temperature", req.Params.Temperature),
		zap.Float32("top_p", req.Params.TopP))

	return req
}

// JiuTianResponseToOpenAIResponse 将九天模型响应转换为OpenAI响应
func JiuTianResponseToOpenAIResponse(jiutianResp *jiutian.ChatCompletionResponse) *openai.ChatCompletionResponse {
	mylog.Logger.Info("Converting JiuTian response to OpenAI response",
		zap.Any("jiutian_response", map[string]interface{}{
			"usage":    jiutianResp.Usage,
			"response": jiutianResp.Response,
			"delta":    jiutianResp.Delta,
			"finished": jiutianResp.Finished,
			"history":  jiutianResp.History,
		}))

	// 创建选项
	choice := openai.ChatCompletionChoice{
		Index: 0,
		Message: openai.ChatCompletionMessage{
			Role:    "assistant",
			Content: jiutianResp.Response,
		},
	}

	// 设置结束原因
	if jiutianResp.Finished != "" {
		choice.FinishReason = convertFinishReason(jiutianResp.Finished)
	}

	resp := &openai.ChatCompletionResponse{
		ID:      "jiutian-" + time.Now().Format("20060102150405"),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   "Llama3.1-70B", // 使用实际的模型ID
		Choices: []openai.ChatCompletionChoice{choice},
		Usage: openai.Usage{
			PromptTokens:     jiutianResp.Usage.PromptTokens,
			CompletionTokens: jiutianResp.Usage.CompletionTokens,
			TotalTokens:      jiutianResp.Usage.TotalTokens,
		},
	}

	mylog.Logger.Info("Converted to OpenAI response",
		zap.Any("openai_response", map[string]interface{}{
			"id":      resp.ID,
			"model":   resp.Model,
			"choices": resp.Choices,
			"usage":   resp.Usage,
		}))

	return resp
}

// JiuTianStreamResponseToOpenAIStreamResponse 将九天模型的流式响应转换为OpenAI流式响应
func JiuTianStreamResponseToOpenAIStreamResponse(jiutianResp *jiutian.ChatCompletionStreamResponse) *openai.ChatCompletionStreamResponse {
	mylog.Logger.Info("Converting JiuTian stream response to OpenAI stream response",
		zap.Any("jiutian_stream_response", map[string]interface{}{
			"response": jiutianResp.Response,
			"delta":    jiutianResp.Delta,
			"finished": jiutianResp.Finished,
			"history":  jiutianResp.History,
		}))

	choice := openai.ChatCompletionStreamChoice{
		Index: 0,
		Delta: openai.ChatCompletionStreamChoiceDelta{
			Role:    "assistant",
			Content: jiutianResp.Response,
		},
	}

	// 只在收到结束标记时设置结束原因
	if jiutianResp.Delta == "[EOS]" {
		choice.FinishReason = convertFinishReason(jiutianResp.Finished)
	}

	resp := &openai.ChatCompletionStreamResponse{
		ID:      "jiutian-stream-" + time.Now().Format("20060102150405"),
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   "Llama3.1-70B", // 使用实际的模型ID
		Choices: []openai.ChatCompletionStreamChoice{choice},
	}

	mylog.Logger.Info("Converted to OpenAI stream response",
		zap.Any("openai_stream_response", map[string]interface{}{
			"id":      resp.ID,
			"model":   resp.Model,
			"choices": resp.Choices,
		}))

	return resp
} 