package utils

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"io"
	"net/http"
	"reflect"
	"simple-one-api/pkg/mylog"
	"strings"
	"time"
)

const defaultHTTPTimeout = 120 * time.Second

var sharedHTTPTransport = func() *http.Transport {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConns = 256
	transport.MaxIdleConnsPerHost = 64
	transport.MaxConnsPerHost = 256
	transport.IdleConnTimeout = 90 * time.Second
	return transport
}()

var (
	sharedHTTPClient      = &http.Client{Transport: sharedHTTPTransport, Timeout: defaultHTTPTimeout}
	sharedStreamingClient = &http.Client{Transport: sharedHTTPTransport}
)

func clientForTransport(httpTransport *http.Transport, streaming bool) *http.Client {
	if streaming {
		return NewHTTPClient(httpTransport, 0)
	}
	return NewHTTPClient(httpTransport, defaultHTTPTimeout)
}

// NewHTTPClient returns a client backed by the shared connection pool when no
// custom transport is required. A zero timeout is appropriate for streams
// whose lifetime is controlled by their request context.
func NewHTTPClient(transport http.RoundTripper, timeout time.Duration) *http.Client {
	if transport == nil || isNilRoundTripper(transport) {
		if timeout == 0 {
			return sharedStreamingClient
		}
		if timeout == defaultHTTPTimeout {
			return sharedHTTPClient
		}
		transport = sharedHTTPTransport
	}
	return &http.Client{Transport: transport, Timeout: timeout}
}

func isNilRoundTripper(transport http.RoundTripper) bool {
	value := reflect.ValueOf(transport)
	return value.Kind() == reflect.Pointer && value.IsNil()
}

// 非SSE的HTTP请求处理函数
func SendHTTPRequest(apiKey, url string, reqBody []byte, httpTransport *http.Transport) ([]byte, error) {
	return SendHTTPRequestContext(context.Background(), apiKey, url, reqBody, httpTransport)
}

func SendHTTPRequestContext(ctx context.Context, apiKey, url string, reqBody []byte, httpTransport *http.Transport) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	client := clientForTransport(httpTransport, false)
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
	return SendSSERequestContext(context.Background(), apiKey, url, reqBody, callback, httpTransport)
}

func SendSSERequestContext(ctx context.Context, apiKey, url string, reqBody []byte, callback func(data string), httpTransport *http.Transport) error {
	mylog.Logger.Debug("SendSSERequest", zap.String("url", url))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	req.Header.Set("Accept", "text/event-stream")

	client := clientForTransport(httpTransport, true)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errMsg string
		respBody, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
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
		if strings.HasPrefix(line, "data:") {
			data := strings.TrimSpace(line[5:])
			callback(data)
		}
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to read streaming response: %w", err)
		}
	}
}

func SendSSERequestWithHttpHeader(apiKey, url string, reqBody []byte, callback func(data string), httpTransport *http.Transport, header map[string]string) error {
	return SendSSERequestWithHttpHeaderContext(context.Background(), apiKey, url, reqBody, callback, httpTransport, header)
}

func SendSSERequestWithHttpHeaderContext(ctx context.Context, apiKey, url string, reqBody []byte, callback func(data string), httpTransport *http.Transport, header map[string]string) error {
	mylog.Logger.Debug("SendSSERequest", zap.String("url", url))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	//req.Header.Set("Accept", "text/event-stream")
	for k, v := range header {
		req.Header.Set(k, v)
	}

	client := clientForTransport(httpTransport, true)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
		if readErr != nil {
			return fmt.Errorf("failed to read error response: %w", readErr)
		}
		return fmt.Errorf("http status code: %d, %s", resp.StatusCode, string(respBody))
	}

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		//mylog.Logger.Debug("SendSSERequest", zap.String("line", line))
		if strings.HasPrefix(line, "data:") {
			data := strings.TrimSpace(line[5:])
			callback(data)
		}
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to read streaming response: %w", err)
		}
	}
}
