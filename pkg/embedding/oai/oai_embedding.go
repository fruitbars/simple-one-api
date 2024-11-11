package oai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// GenerateEmbedding 生成文本的嵌入向量
func OpenAIEmbedding(embReq *EmbeddingRequest, apiKey string, proxyTransport *http.Transport) (*EmbeddingResponse, error) {

	url := "https://api.openai.com/v1/embeddings"
	requestBody, err := json.Marshal(embReq)
	if err != nil {
		return nil, fmt.Errorf("JSON 编码错误: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("创建请求错误: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	var client *http.Client
	if proxyTransport != nil {
		client = &http.Client{
			Timeout:   60 * time.Second,
			Transport: proxyTransport,
		}
	} else {
		client = &http.Client{
			Timeout:   60 * time.Second,
			Transport: http.DefaultTransport, // 使用默认的 Transport
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求错误: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应错误: %v", err)
	}

	var response EmbeddingResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}
