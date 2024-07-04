package utils

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// CustomTransport 是一个自定义的 RoundTripper
type CustomTransport struct {
	Transport http.RoundTripper
}

// RoundTrip 实现了 http.RoundTripper 接口
func (c *CustomTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := c.Transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	// 检查 HTTP 状态码，如果是错误状态码，读取响应体并返回错误
	if resp.StatusCode >= 400 {
		bodyBytes, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("error reading error response body: %v", readErr)
		}
		resp.Body.Close()
		return nil, fmt.Errorf("HTTP error: %s, body: %s", resp.Status, string(bodyBytes))
	}

	// 创建一个新的响应体
	modifiedBody := &modifiedReadCloser{
		originalBody: resp.Body,
		reader:       bufio.NewReader(resp.Body),
	}
	resp.Body = modifiedBody

	return resp, nil
}

// modifiedReadCloser 是一个自定义的 ReadCloser，用于修改响应体内容
type modifiedReadCloser struct {
	originalBody io.ReadCloser
	buf          *bytes.Buffer
	reader       *bufio.Reader
}

func (m *modifiedReadCloser) Read(p []byte) (int, error) {
	// 如果缓冲区为空，从原始响应体读取数据并处理
	if m.buf == nil || m.buf.Len() == 0 {
		line, err := m.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return 0, io.EOF
			}
			return 0, err
		}
		// 仅在 "data:" 而不是 "data: " 的情况下进行替换
		if strings.HasPrefix(line, "data:") && !strings.HasPrefix(line, "data: ") {
			line = strings.Replace(line, "data:", "data: ", 1)
		}
		m.buf = bytes.NewBufferString(line)
	}
	return m.buf.Read(p)
}

func (m *modifiedReadCloser) Close() error {
	return m.originalBody.Close()
}
