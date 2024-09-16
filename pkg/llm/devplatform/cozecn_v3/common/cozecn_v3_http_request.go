package common

import (
	"bufio"
	"bytes"
	"fmt"
	"go.uber.org/zap"
	"io"
	"net/http"
	"simple-one-api/pkg/mylog"
	"strings"
)

// 非SSE的HTTP请求处理函数
func SendCozeV3HTTPRequest(apiKey, url string, reqBody []byte, httpTransport *http.Transport) ([]byte, error) {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	client := &http.Client{}

	if httpTransport != nil {
		client.Transport = httpTransport
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return respBody, nil
}

// SSE的HTTP请求处理函数，带回调处理每次接收的数据
func SendCozeV3StreamHttpRequest(apiKey, url string, reqBody []byte, callback func(event, data string), httpTransport *http.Transport) error {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	req.Header.Set("Accept", "text/event-stream")

	client := &http.Client{}
	if httpTransport != nil {
		client.Transport = httpTransport
	}

	mylog.Logger.Info("SendCozeV3StreamHttpRequest", zap.String("url", url), zap.String("reqBody", string(reqBody)))

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	scanner := bufio.NewScanner(reader)

	// 自定义分割函数，以两个换行符作为分隔
	scanner.Split(splitOnDoubleNewline)
	for scanner.Scan() {
		part := scanner.Text()
		rParts := strings.Split(part, "\n")
		if len(rParts) == 2 {
			if strings.HasPrefix(rParts[0], "event:") && strings.HasPrefix(rParts[1], "data:") {
				event, data := strings.TrimSpace(rParts[0][6:]), strings.TrimSpace(rParts[1][5:])
				//log.Println(event, data)
				callback(event, data)
			}
		}
	}

	return nil
}

// 修正后的 splitOnDoubleNewline，不返回空的 token
func splitOnDoubleNewline(data []byte, atEOF bool) (advance int, token []byte, err error) {
	doubleNewline := []byte("\n\n")
	if i := bytes.Index(data, doubleNewline); i >= 0 {
		if i == 0 {
			// 如果分隔符在开头，跳过它
			return len(doubleNewline), nil, nil
		}
		// 返回分隔符前的内容
		return i + len(doubleNewline), data[0:i], nil
	}
	if atEOF {
		if len(data) == 0 {
			return 0, nil, nil
		}
		return len(data), data, nil
	}
	// 请求更多的数据
	return 0, nil, nil
}
