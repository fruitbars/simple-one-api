package chat_message_request

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"simple-one-api/pkg/llm/devplatform/dify/chat_completion_response"
	"simple-one-api/pkg/utils"
)

var baseURL = "https://api.dify.ai/v1"

func CallChatMessagesStreamMode(difyReq *ChatMessageRequest, apiKey string, callback func(data string), httpTransport *http.Transport) error {
	serverUrl := "https://api.dify.ai/v1/chat-messages"

	reqData, _ := json.Marshal(difyReq)

	return utils.SendSSERequest(apiKey, serverUrl, reqData, callback, httpTransport)
}

func CallChatMessagesNoneStreamMode(difyReq *ChatMessageRequest, apiKey string, httpTransport *http.Transport) (*chat_completion_response.ChatCompletionResponse, error) {
	serverUrl := "https://api.dify.ai/v1/chat-messages"
	// 创建请求体

	reqData, _ := json.Marshal(difyReq)

	respData, err := utils.SendHTTPRequest(apiKey, serverUrl, reqData, httpTransport)
	if err != nil {
		return nil, err
	}

	var difyResp chat_completion_response.ChatCompletionResponse
	err = json.Unmarshal(respData, &difyResp)
	if err != nil {
		return nil, err
	}

	return &difyResp, nil
}
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
