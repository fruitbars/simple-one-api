package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/mylog"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	mylog.Logger = zap.NewNop()
	m.Run()
}

func TestMinimaxReturnsUpstreamHTTPErrorWithoutMutatingConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	model := &config.ModelDetails{ServiceModel: config.ServiceModel{ServerURL: server.URL}}
	params := &OAIRequestParam{
		ctx:               context.Background(),
		chatCompletionReq: &openai.ChatCompletionRequest{Model: "abab", Messages: []openai.ChatCompletionMessage{{Role: "user", Content: "hello"}}},
		modelDetails:      model,
		creds:             map[string]interface{}{config.KEYNAME_API_KEY: "key", config.KEYNAME_GROUP_ID: "group"},
		ClientModel:       "abab",
	}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	if err := OpenAI2MinimaxHandler(c, params); err == nil {
		t.Fatal("OpenAI2MinimaxHandler() returned nil for upstream 503")
	}
	if model.ServerURL != server.URL {
		t.Fatalf("ServerURL mutated to %q", model.ServerURL)
	}
}

func TestMinimaxRequestHonorsCancellation(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
	}))
	t.Cleanup(func() {
		releaseOnce.Do(func() { close(release) })
		server.Close()
	})

	ctx, cancel := context.WithCancel(context.Background())
	params := &OAIRequestParam{
		ctx:               ctx,
		chatCompletionReq: &openai.ChatCompletionRequest{Model: "abab"},
		modelDetails:      &config.ModelDetails{ServiceModel: config.ServiceModel{ServerURL: server.URL}},
		creds:             map[string]interface{}{config.KEYNAME_API_KEY: "key", config.KEYNAME_GROUP_ID: "group"},
		ClientModel:       "abab",
	}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	done := make(chan error, 1)
	go func() { done <- OpenAI2MinimaxHandler(c, params) }()
	<-started
	cancel()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("request succeeded after cancellation")
		}
		releaseOnce.Do(func() { close(release) })
	case <-time.After(time.Second):
		t.Fatal("upstream request did not stop after cancellation")
	}
}
