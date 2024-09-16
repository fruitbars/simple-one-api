package chat_retrieve

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// 发起GET请求并处理响应
func ChatRetrieve(chatID string, conversationID string, token string) (*ChatRetrieveResponse, error) {
	// 构造请求URL
	url := fmt.Sprintf("https://api.coze.cn/v3/chat/retrieve?chat_id=%s&conversation_id=%s", chatID, conversationID)

	// 创建一个新的请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// 设置头部
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	// 创建HTTP客户端并发送请求
	client := &http.Client{
		Timeout: 3 * time.Minute,
	}
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

	// 解析响应体JSON
	var chatResponse ChatRetrieveResponse
	err = json.Unmarshal(body, &chatResponse)
	if err != nil {
		return nil, err
	}

	return &chatResponse, nil
}
