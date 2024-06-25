package mycommon

import (
	"github.com/sashabaranov/go-openai"
	"strings"
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
