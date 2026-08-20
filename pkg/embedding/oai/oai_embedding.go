package oai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultEmbeddingURL = "https://api.openai.com/v1/embeddings"
const maxEmbeddingErrorBytes = 1 << 20

func resolveEmbeddingURL(serverURL string) (string, error) {
	serverURL = strings.TrimSpace(serverURL)
	if serverURL == "" {
		return defaultEmbeddingURL, nil
	}

	endpoint, err := url.Parse(serverURL)
	if err != nil || (endpoint.Scheme != "http" && endpoint.Scheme != "https") || endpoint.Host == "" {
		return "", fmt.Errorf("invalid OpenAI embedding server_url %q", serverURL)
	}
	if endpoint.Fragment != "" {
		return "", fmt.Errorf("OpenAI embedding server_url must not contain a fragment")
	}

	path := strings.TrimRight(endpoint.Path, "/")
	switch {
	case strings.HasSuffix(path, "/embeddings"):
	case strings.HasSuffix(path, "/chat/completions"):
		path = strings.TrimSuffix(path, "/chat/completions") + "/embeddings"
	default:
		path += "/embeddings"
	}
	endpoint.Path = path
	endpoint.RawPath = ""
	return endpoint.String(), nil
}

// OpenAIEmbedding sends an embedding request to OpenAI or a compatible endpoint.
func OpenAIEmbedding(ctx context.Context, embReq *EmbeddingRequest, apiKey, serverURL string, proxyTransport *http.Transport) (*EmbeddingResponse, error) {
	endpoint, err := resolveEmbeddingURL(serverURL)
	if err != nil {
		return nil, err
	}
	requestBody, err := json.Marshal(embReq)
	if err != nil {
		return nil, fmt.Errorf("JSON 编码错误: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(requestBody))
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
		return nil, fmt.Errorf("请求错误: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxEmbeddingErrorBytes))
		if readErr != nil {
			return nil, fmt.Errorf("读取错误响应失败: %v", readErr)
		}
		return nil, fmt.Errorf("embedding upstream returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

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
