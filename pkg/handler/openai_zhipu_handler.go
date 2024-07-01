package handler

import (
	"fmt"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"simple-one-api/pkg/mylog"
	"simple-one-api/pkg/utils"
	"strings"
)

func extractBase64Data(base64Image string) (string, error) {
	if base64Image == "" {
		return "", fmt.Errorf("base64Image is empty")
	}

	parts := strings.SplitN(base64Image, ",", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid base64Image format")
	}

	return parts[1], nil
}

func AdjustChatCompletionRequestForZhiPu(oaiReq *openai.ChatCompletionRequest) {
	for i := range oaiReq.Messages {
		msg := &oaiReq.Messages[i]
		if len(msg.MultiContent) > 0 {
			mylog.Logger.Info("2")
			for _, content := range msg.MultiContent {
				if content.Type == openai.ChatMessagePartTypeImageURL {
					if strings.HasPrefix(content.ImageURL.URL, "data:image/") {
						encodedData, err := extractBase64Data(content.ImageURL.URL)
						if err != nil {
							debugstr := content.ImageURL.URL[:utils.Min(len(content.ImageURL.URL), 10)] + "..."
							mylog.Logger.Warn("base64 format err", zap.String("ImageURL.URL", debugstr))
							continue
						}

						content.ImageURL.URL = encodedData
					}
				}
			}
		}
	}
}
