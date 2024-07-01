package mycommon

import (
	"encoding/base64"
	"fmt"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"io"
	"net/http"
	"simple-one-api/pkg/mylog"
	"strings"
	"time"
)

// ProcessMessages 根据消息的角色处理聊天历史。
func ConvertSystemMessages2NoSystem(oaiReq []openai.ChatCompletionMessage) []openai.ChatCompletionMessage {
	var systemQuery string
	if len(oaiReq) == 0 {
		return oaiReq
	}

	// 如果第一条消息的角色是 "system"，根据条件处理消息
	if strings.ToLower(oaiReq[0].Role) == "system" {
		if len(oaiReq) == 1 {
			oaiReq[0].Role = "user"
		} else {
			systemQuery = oaiReq[0].Content
			oaiReq = oaiReq[1:] // 移除系统消息
			oaiReq[0].Content = systemQuery + "\n" + oaiReq[0].Content
		}
	}

	return oaiReq
}

func NormalizeMessages(oaiReqMessage []openai.ChatCompletionMessage) []openai.ChatCompletionMessage {
	//var systemQuery string
	if len(oaiReqMessage) == 0 {
		return oaiReqMessage
	}

	// 处理第一条消息是 system 的情况
	if strings.ToLower(oaiReqMessage[0].Role) == "system" {
		if len(oaiReqMessage) == 1 {
			oaiReqMessage[0].Role = "user"
		}
	}

	// 创建一个新的切片来存储规范化的消息
	var normalizedMessages []openai.ChatCompletionMessage

	// 跟踪上一个角色
	var lastRole string

	// 遍历消息数组
	for i, msg := range oaiReqMessage {
		role := strings.ToLower(msg.Role)
		if role == "system" && i > 0 {
			// 移除非第一条出现的 system 消息
			continue
		}
		if role == "user" || role == "assistant" {
			// 检查角色是否交替出现
			if role == lastRole {
				continue
			}
			normalizedMessages = append(normalizedMessages, msg)
			lastRole = role
		} else {
			// 保留不认识的角色
			normalizedMessages = append(normalizedMessages, msg)
		}
	}

	return normalizedMessages
}

// getImageURLData 分析给定的 URL 字符串，并返回其 base64 编码数据和 MIME 类型
func GetImageURLData(dataStr string) (string, string, error) {
	if strings.HasPrefix(dataStr, "data:") {
		// 处理 base64 编码的图片数据
		sepIndex := strings.Index(dataStr, ",")
		if sepIndex == -1 {
			return "", "", fmt.Errorf("invalid data URL format")
		}
		mime := dataStr[5:sepIndex]
		base64Data := dataStr[sepIndex+1:]
		return base64Data, mime, nil
	} else if strings.HasPrefix(dataStr, "http") {
		// 处理 HTTP URL
		client := &http.Client{
			Timeout: 30 * time.Second, // 设置30秒超时
		}
		response, err := client.Get(dataStr)
		if err != nil {
			return "", "", fmt.Errorf("error fetching image: %v", err)
		}
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			return "", "", fmt.Errorf("failed to download image: HTTP status %d", response.StatusCode)
		}

		// 通过 base64.NewEncoder 创建一个写入器，直接将数据编码为 base64
		var base64Writer strings.Builder
		encoder := base64.NewEncoder(base64.StdEncoding, &base64Writer)
		defer encoder.Close()

		// 从 response.Body 直接流式读取数据到 base64 编码器
		if _, err := io.Copy(encoder, response.Body); err != nil {
			return "", "", fmt.Errorf("error encoding image data to base64: %v", err)
		}

		mimeType := response.Header.Get("Content-Type")
		return base64Writer.String(), mimeType, nil
	}

	return "", "", fmt.Errorf("unsupported URL format")
}

func AdjustOpenAIRequestParams(oaiReq *openai.ChatCompletionRequest) {
	adjustedTemperature, adjustedTopP, adjustedMaxTokens, err := AdjustParamsToRange(oaiReq.Model, oaiReq.Temperature, oaiReq.TopP, oaiReq.MaxTokens)

	if err != nil {
		return
	}
	oaiReq.Temperature = adjustedTemperature
	oaiReq.TopP = adjustedTopP
	oaiReq.MaxTokens = adjustedMaxTokens

	mylog.Logger.Debug("", zap.Float32("adjustedTemperature", adjustedTemperature),
		zap.Float32("adjustedTopP", adjustedTopP),
		zap.Int("MaxTokens", adjustedMaxTokens),
	)
}
