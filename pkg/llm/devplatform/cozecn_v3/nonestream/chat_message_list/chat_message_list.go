package chat_message_list

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// 发起POST请求并处理响应
func ChatMessageslist(chatID string, conversationID string, token string) (*MessageListResponse, error) {
	// 构造请求URL
	url := fmt.Sprintf("https://api.coze.cn/v3/chat/message/list?chat_id=%s&conversation_id=%s", chatID, conversationID)

	// 创建一个新的POST请求，空的body
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte("{}")))
	if err != nil {
		return nil, err
	}

	// 设置头部
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	// 创建HTTP客户端并发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	//	log.Println(string(body))

	// 解析响应体JSON
	var messageListResponse MessageListResponse
	err = json.Unmarshal(body, &messageListResponse)
	if err != nil {
		return nil, err
	}

	return &messageListResponse, nil
}
