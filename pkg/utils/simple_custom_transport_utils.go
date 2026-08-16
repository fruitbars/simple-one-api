package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// CustomTransport 是一个自定义的 RoundTripper
type SimpleCustomTransport struct {
	Transport       http.RoundTripper
	ExtraJSONFields map[string]json.RawMessage
}

// RoundTrip 实现了 http.RoundTripper 接口
func (c *SimpleCustomTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if len(c.ExtraJSONFields) > 0 && req.Body != nil && req.Header.Get("Content-Type") == "application/json" {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("read request body: %w", err)
		}
		var structured map[string]json.RawMessage
		if err := json.Unmarshal(body, &structured); err != nil {
			return nil, fmt.Errorf("decode request body: %w", err)
		}
		merged := make(map[string]json.RawMessage, len(c.ExtraJSONFields)+len(structured))
		for key, value := range c.ExtraJSONFields {
			merged[key] = value
		}
		for key, value := range structured {
			merged[key] = value
		}
		body, err = json.Marshal(merged)
		if err != nil {
			return nil, fmt.Errorf("encode request body: %w", err)
		}
		req.Body = io.NopCloser(bytes.NewReader(body))
		req.ContentLength = int64(len(body))
	}
	resp, err := c.Transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	// 检查 HTTP 状态码，如果是错误状态码，读取最多 1024 个字节的响应体并返回错误
	if resp.StatusCode >= 400 {
		bodyBytes := make([]byte, 1024)
		n, readErr := resp.Body.Read(bodyBytes)
		if readErr != nil && readErr != io.EOF {
			return nil, fmt.Errorf("error reading error response body: %v", readErr)
		}
		resp.Body.Close()
		return nil, fmt.Errorf("HTTP error: %s, body: %s", resp.Status, string(bodyBytes[:n]))
	}

	return resp, nil
}
