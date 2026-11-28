package handler

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sashabaranov/go-openai"
	"simple-one-api/pkg/config"
)

func TestGeminiHonorsRequestDeadline(t *testing.T) {
	for _, tc := range []struct {
		name    string
		stream  bool
		timeout time.Duration
		wantErr error
	}{
		{name: "slow response headers", timeout: 2 * time.Minute},
		{name: "slow stream body", stream: true, timeout: 2 * time.Minute},
		{name: "shorter request deadline", timeout: 30 * time.Second, wantErr: context.DeadlineExceeded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				// net.Pipe lets the upstream delay run on synctest's clock.
				conn, upstream := net.Pipe()
				transport := &http.Transport{
					DialContext: func(context.Context, string, string) (net.Conn, error) {
						return conn, nil
					},
				}
				defer transport.CloseIdleConnections()

				upstreamDone := make(chan struct{})
				defer func() {
					conn.Close()
					<-upstreamDone
				}()
				go func() {
					defer close(upstreamDone)
					defer upstream.Close()
					req, err := http.ReadRequest(bufio.NewReader(upstream))
					if err != nil {
						t.Error(err)
						return
					}
					if _, err := io.Copy(io.Discard, req.Body); err != nil {
						t.Error(err)
						return
					}
					const payload = `{"candidates":[{"content":{"role":"model","parts":[{"text":"hello"}]}}]}`
					if tc.stream {
						io.WriteString(upstream, "HTTP/1.1 200 OK\r\nContent-Type: text/event-stream\r\nConnection: close\r\n\r\n")
					}
					time.Sleep(61 * time.Second)
					if tc.stream {
						io.WriteString(upstream, "data: "+payload+"\n\n")
					} else {
						io.WriteString(upstream, "HTTP/1.1 200 OK\r\nConnection: close\r\n\r\n"+payload)
					}
				}()

				ctx, cancel := context.WithTimeout(t.Context(), tc.timeout)
				defer cancel()
				recorder := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(recorder)
				params := &OAIRequestParam{
					ctx:               ctx,
					chatCompletionReq: &openai.ChatCompletionRequest{Model: "gemini-test", Stream: tc.stream},
					modelDetails:      &config.ModelDetails{ServiceModel: config.ServiceModel{ServerURL: "http://gemini.test"}},
					httpTransport:     transport,
					ClientModel:       "gemini-test",
				}
				if err := OpenAI2GeminiHandler(c, params); !errors.Is(err, tc.wantErr) {
					t.Fatalf("OpenAI2GeminiHandler() error = %v, want %v", err, tc.wantErr)
				}
				if tc.wantErr == nil && !strings.Contains(recorder.Body.String(), `"content":"hello"`) {
					t.Fatalf("response missing the delayed content: %s", recorder.Body)
				}
			})
		})
	}
}
