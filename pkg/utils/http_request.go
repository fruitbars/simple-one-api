package utils

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
func SendHTTPRequest(apiKey, url string, reqBody []byte, httpTransport *http.Transport) ([]byte, error) {
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

	if resp.StatusCode != http.StatusOK {
		errMsg := string(respBody)
		return nil, fmt.Errorf("http status code: %d, %s", resp.StatusCode, errMsg)
	}

	return respBody, nil
}

// SSE的HTTP请求处理函数，带回调处理每次接收的数据
func SendSSERequest(apiKey, url string, reqBody []byte, callback func(data string), httpTransport *http.Transport) error {
	mylog.Logger.Debug("SendSSERequest", zap.String("url", url))
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
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errMsg string
		respBody, err := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		if err != nil {
			mylog.Logger.Error(err.Error())
		}
		if len(respBody) > 0 {
			errMsg = string(respBody)
		} else {
			errMsg = "empty response body"
		}

		return fmt.Errorf("http status code: %d, %s", resp.StatusCode, errMsg)
	}

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		mylog.Logger.Debug("SendSSERequest", zap.String("line", line))
		if err != nil {
			break
		}
		if strings.HasPrefix(line, "data:") {
			data := strings.TrimSpace(line[5:])
			callback(data)
		}
	}
	return nil
}

func SendSSERequestWithHttpHeader(apiKey, url string, reqBody []byte, callback func(data string), httpTransport *http.Transport, header map[string]string) error {
	mylog.Logger.Debug("SendSSERequest", zap.String("url", url))
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	//req.Header.Set("Accept", "text/event-stream")
	for k, v := range header {
		req.Header.Set(k, v)
	}

	client := &http.Client{}
	if httpTransport != nil {
		client.Transport = httpTransport
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		//mylog.Logger.Debug("SendSSERequest", zap.String("line", line))
		if err != nil {
			break
		}
		if strings.HasPrefix(line, "data:") {
			data := strings.TrimSpace(line[5:])
			callback(data)
		}
	}
	return nil
}
