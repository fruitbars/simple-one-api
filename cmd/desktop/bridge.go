package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"simple-one-api/pkg/config"
)

const desktopChatEventPrefix = "simple-one-api:chat:"
const desktopErrorBodyLimit = 64 << 10

type desktopEventEmitter func(context.Context, string, ...interface{})

type DesktopBridge struct {
	mu       sync.Mutex
	ctx      context.Context
	api      http.Handler
	requests map[string]context.CancelFunc
	emit     desktopEventEmitter
}

func NewDesktopBridge(api http.Handler) *DesktopBridge {
	return &DesktopBridge{
		api: api, requests: make(map[string]context.CancelFunc), emit: runtime.EventsEmit,
	}
}

func (bridge *DesktopBridge) startup(ctx context.Context) {
	bridge.mu.Lock()
	bridge.ctx = ctx
	bridge.mu.Unlock()
}

func (bridge *DesktopBridge) shutdown(context.Context) {
	bridge.mu.Lock()
	for _, cancel := range bridge.requests {
		cancel()
	}
	bridge.requests = make(map[string]context.CancelFunc)
	bridge.mu.Unlock()
}

// StreamChat executes the regular in-process API handler and forwards each
// response write as base64 so WebView text decoding remains byte-accurate.
func (bridge *DesktopBridge) StreamChat(requestID, apiKey, payload string) error {
	if !validDesktopRequestID(requestID) {
		return errors.New("invalid desktop stream request ID")
	}
	bridge.mu.Lock()
	if bridge.ctx == nil || bridge.api == nil || bridge.emit == nil {
		bridge.mu.Unlock()
		return errors.New("desktop stream bridge is unavailable")
	}
	if _, exists := bridge.requests[requestID]; exists {
		bridge.mu.Unlock()
		return errors.New("desktop stream request already exists")
	}
	requestContext, cancel := context.WithCancel(bridge.ctx)
	bridge.requests[requestID] = cancel
	bridge.mu.Unlock()
	defer func() {
		cancel()
		bridge.mu.Lock()
		delete(bridge.requests, requestID)
		bridge.mu.Unlock()
	}()

	request, err := http.NewRequestWithContext(requestContext, http.MethodPost, "/v1/chat/completions", strings.NewReader(payload))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	key := strings.TrimSpace(apiKey)
	if key == "" {
		key = strings.TrimSpace(config.CurrentAPIKey())
	}
	if key != "" {
		request.Header.Set("Authorization", "Bearer "+key)
	}
	eventName := desktopChatEventPrefix + requestID
	writer := newDesktopStreamWriter(func(data []byte) {
		bridge.emit(requestContext, eventName, base64.StdEncoding.EncodeToString(data))
	})
	bridge.api.ServeHTTP(writer, request)
	if writer.Status() >= http.StatusBadRequest {
		detail := strings.TrimSpace(writer.ErrorBody())
		if detail == "" {
			detail = http.StatusText(writer.Status())
		}
		return fmt.Errorf("request failed (HTTP %d): %s", writer.Status(), detail)
	}
	return requestContext.Err()
}

func (bridge *DesktopBridge) CancelChat(requestID string) {
	bridge.mu.Lock()
	cancel := bridge.requests[requestID]
	bridge.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func validDesktopRequestID(value string) bool {
	if value == "" || len(value) > 128 {
		return false
	}
	for _, character := range value {
		if (character < 'a' || character > 'z') && (character < 'A' || character > 'Z') &&
			(character < '0' || character > '9') && character != '-' && character != '_' {
			return false
		}
	}
	return true
}

type desktopStreamWriter struct {
	header    http.Header
	status    int
	errorBody bytes.Buffer
	emit      func([]byte)
}

func newDesktopStreamWriter(emit func([]byte)) *desktopStreamWriter {
	return &desktopStreamWriter{header: make(http.Header), emit: emit}
}

func (writer *desktopStreamWriter) Header() http.Header { return writer.header }

func (writer *desktopStreamWriter) WriteHeader(status int) {
	if writer.status == 0 {
		writer.status = status
	}
}

func (writer *desktopStreamWriter) Write(data []byte) (int, error) {
	if writer.status == 0 {
		writer.status = http.StatusOK
	}
	if writer.status >= http.StatusBadRequest {
		remaining := desktopErrorBodyLimit - writer.errorBody.Len()
		if remaining > len(data) {
			remaining = len(data)
		}
		if remaining > 0 {
			_, _ = writer.errorBody.Write(data[:remaining])
		}
		return len(data), nil
	}
	writer.emit(data)
	return len(data), nil
}

func (writer *desktopStreamWriter) Flush() {}

func (writer *desktopStreamWriter) Status() int {
	if writer.status == 0 {
		return http.StatusOK
	}
	return writer.status
}

func (writer *desktopStreamWriter) ErrorBody() string { return writer.errorBody.String() }
