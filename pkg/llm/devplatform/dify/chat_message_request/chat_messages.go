package chat_messages

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

var baseURL = "https://api.dify.ai/v1"

func CallChatMessages(query, conversationID string, apiKey string, streamMode bool) (string, error) {
	url := fmt.Sprintf("%s/chat-messages", baseURL)

	responseMode := "streaming"
	if !streamMode {
		responseMode = "blocking"
	}
	// 创建请求体
	requestBody := ChatMessageRequest{
		Inputs:         map[string]interface{}{},
		Query:          query,
		ResponseMode:   responseMode,
		ConversationID: conversationID,
		User:           "abc-123",
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	// 创建 HTTP POST 请求
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	// 设置请求头
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
