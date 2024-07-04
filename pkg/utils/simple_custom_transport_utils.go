package utils

import (
	"fmt"
	"io"
	"net/http"
)

// CustomTransport 是一个自定义的 RoundTripper
type SimpleCustomTransport struct {
	Transport http.RoundTripper
}

// RoundTrip 实现了 http.RoundTripper 接口
func (c *SimpleCustomTransport) RoundTrip(req *http.Request) (*http.Response, error) {
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
