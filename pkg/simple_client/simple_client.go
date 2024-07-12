package simple_client

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/sashabaranov/go-openai"
	"io"
	"net/http"
	"net/http/httptest"
	"simple-one-api/pkg/handler"
)

func init() {

}

type SimpleClient struct {
}

func NewSimpleClient(authToken string) *SimpleClient {
	//config := DefaultConfig(authToken)
	return NewSimpleClientWithConfig()
}

// NewClientWithConfig creates new OpenAI API client for specified config.
func NewSimpleClientWithConfig() *SimpleClient {
	return &SimpleClient{
		//config: config,
	}
}

func (c *SimpleClient) CreateChatCompletion(
	ctx context.Context,
	request openai.ChatCompletionRequest,
) (response openai.ChatCompletionResponse, err error) {
	request.Stream = false
	reqBody, _ := json.Marshal(request)
	httpReq, _ := http.NewRequest("POST", "/v1/chat/completions", bytes.NewBuffer(reqBody))
	httpReq.Header.Set("Content-Type", "application/json")

	// 使用httptest创建一个响应记录器
	w := httptest.NewRecorder()

	// 使用Gin的默认上下文
	ginc, _ := gin.CreateTestContext(w)

	// 使用上面创建的HTTP请求
	ginc.Request = httpReq
	handler.HandleOpenAIRequest(ginc, &request)

	json.Unmarshal(w.Body.Bytes(), &response)

	return
}

func (c *SimpleClient) CreateChatCompletionStream(
	ctx context.Context,
	request openai.ChatCompletionRequest,
) (stream *SimpleChatCompletionStream, err error) {
	request.Stream = true
	// 创建io.Pipe连接
	reader, writer := io.Pipe()

	// 配置gin的上下文和请求
	ginc := gin.New()
	ginc.Use(func(ctx *gin.Context) {
		ctx.Writer = NewCustomResponseWriter(writer)
	})
	ginc.POST("/v1/chat/completions", func(ctx *gin.Context) {
		handler.HandleOpenAIRequest(ctx, &request)
	})

	// 模拟发送请求
	go func() {
		defer writer.Close()
		requestData, _ := json.Marshal(request)
		httpReq, _ := http.NewRequest("POST", "/v1/chat/completions", bytes.NewBuffer(requestData))
		httpReq.Header.Set("Content-Type", "application/json")
		ginc.ServeHTTP(httptest.NewRecorder(), httpReq)
	}()

	return NewSimpleChatCompletionStream(reader), nil
}
