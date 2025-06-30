package jiutian

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"io"
	"net/http"
	"simple-one-api/pkg/mylog"
	"strings"
	"time"
)

const (
	// DefaultBaseURL 默认的九天模型API基础URL
	DefaultBaseURL = "https://jiutian.10086.cn/largemodel/api/v1"
	// DefaultTimeout 默认超时时间
	DefaultTimeout = 3 * time.Minute
)

// Message 九天模型的消息结构
type Message struct {
	Role    string `json:"role"`    // 角色：system/user/assistant
	Content string `json:"content"` // 消息内容
}

// ChatCompletionRequest 九天模型的对话请求结构
type ChatCompletionRequest struct {
	ModelID     string     `json:"modelId"`              // 模型ID
	Prompt      string     `json:"prompt"`               // 当前问题
	Params      *Params    `json:"params"`               // 参数配置
	History     [][]string `json:"history"`              // 历史对话记录
	Stream      bool       `json:"stream"`               // 是否使用流式响应
	apiKey      string     // API密钥
	baseURL     string     // API基础URL
	transport   http.RoundTripper // HTTP传输层
}

// Params 模型参数配置
type Params struct {
	Temperature float32 `json:"temperature"` // 温度参数
	TopP        float32 `json:"top_p"`      // 核采样参数
}

// NewChatCompletionRequest 创建新的对话请求
func NewChatCompletionRequest() *ChatCompletionRequest {
	return &ChatCompletionRequest{
		ModelID: "Llama3.1-70B", // 默认模型
		Params: &Params{
			Temperature: 0.7,  // 默认温度
			TopP:        0.95, // 默认top_p
		},
		History: make([][]string, 0),
		Stream:  false,
		baseURL: DefaultBaseURL,
	}
}

// WithModelID 设置模型ID
func (r *ChatCompletionRequest) WithModelID(modelID string) *ChatCompletionRequest {
	r.ModelID = modelID
	return r
}

// WithPrompt 设置当前问题
func (r *ChatCompletionRequest) WithPrompt(prompt string) *ChatCompletionRequest {
	r.Prompt = prompt
	return r
}

// WithHistory 设置历史对话记录
func (r *ChatCompletionRequest) WithHistory(history [][]string) *ChatCompletionRequest {
	r.History = history
	return r
}

// WithStream 设置是否使用流式响应
func (r *ChatCompletionRequest) WithStream(stream bool) *ChatCompletionRequest {
	r.Stream = stream
	return r
}

// WithTemperature 设置温度参数
func (r *ChatCompletionRequest) WithTemperature(temperature float32) *ChatCompletionRequest {
	r.Params.Temperature = temperature
	return r
}

// WithTopP 设置top_p参数
func (r *ChatCompletionRequest) WithTopP(topP float32) *ChatCompletionRequest {
	r.Params.TopP = topP
	return r
}

// WithAPIKey 设置API密钥
func (r *ChatCompletionRequest) WithAPIKey(apiKey string) *ChatCompletionRequest {
	r.apiKey = apiKey
	return r
}

// WithBaseURL 设置API基础URL
func (r *ChatCompletionRequest) WithBaseURL(baseURL string) *ChatCompletionRequest {
	if baseURL != "" {
		r.baseURL = baseURL
	}
	return r
}

// WithTransport 设置HTTP传输层
func (r *ChatCompletionRequest) WithTransport(transport http.RoundTripper) *ChatCompletionRequest {
	r.transport = transport
	return r
}

// validate 验证请求参数
func (r *ChatCompletionRequest) validate() error {
	if r.ModelID == "" {
		return errors.New("modelId is required")
	}
	if r.Prompt == "" {
		return errors.New("prompt is required")
	}
	if r.Params == nil {
		return errors.New("params is required")
	}
	if r.Params.Temperature < 0 || r.Params.Temperature > 2 {
		return errors.New("temperature must be between 0 and 2")
	}
	if r.Params.TopP < 0 || r.Params.TopP > 1 {
		return errors.New("top_p must be between 0 and 1")
	}
	if r.apiKey == "" {
		return errors.New("API key is required")
	}
	return nil
}

// generateToken 生成JWT token
func (r *ChatCompletionRequest) generateToken() (string, error) {
	mylog.Logger.Info("Generating JiuTian JWT token")

	// 分割API Key
	parts := strings.Split(r.apiKey, ".")
	if len(parts) != 2 {
		mylog.Logger.Error("Invalid API key format", zap.String("api_key", r.apiKey))
		return "", errors.New("invalid API key format")
	}
	id, secret := parts[0], parts[1]
	mylog.Logger.Debug("API key parsed", zap.String("id", id))

	// 创建token
	now := time.Now().Unix()
	claims := jwt.MapClaims{
		"api_key":   id,
		"exp":       now + 3600, // 1小时有效期
		"timestamp": now,
	}

	mylog.Logger.Debug("Creating JWT claims",
		zap.String("api_key", id),
		zap.Int64("exp", now+3600),
		zap.Int64("timestamp", now))

	// 设置header
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["alg"] = "HS256"
	token.Header["typ"] = "JWT"
	token.Header["sign_type"] = "SIGN"

	mylog.Logger.Debug("JWT header set",
		zap.String("alg", "HS256"),
		zap.String("typ", "JWT"),
		zap.String("sign_type", "SIGN"))

	// 签名
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		mylog.Logger.Error("Failed to sign token",
			zap.Error(err),
			zap.String("id", id))
		return "", fmt.Errorf("failed to sign token: %v", err)
	}

	mylog.Logger.Info("JWT token generated successfully",
		zap.String("token", tokenString),
		zap.Int64("expires_at", now+3600))

	return tokenString, nil
}

// CreateCompletion 发送非流式请求
func (r *ChatCompletionRequest) CreateCompletion() (*ChatCompletionResponse, error) {
	mylog.Logger.Info("Creating JiuTian chat completion")

	// 验证请求参数
	if err := r.validate(); err != nil {
		return nil, err
	}

	// 生成token
	token, err := r.generateToken()
	if err != nil {
		mylog.Logger.Error("Failed to generate token", zap.Error(err))
		return nil, err
	}

	// 准备请求URL
	url := fmt.Sprintf("%s/completions", r.baseURL)
	mylog.Logger.Info("Accessing JiuTian API", zap.String("full_url", url))

	// 准备请求数据
	jsonData, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}

	// 创建HTTP客户端
	client := &http.Client{
		Timeout: DefaultTimeout,
	}
	
	// 设置Transport
	if r.transport != nil {
		client.Transport = r.transport
	} else {
		client.Transport = http.DefaultTransport
	}

	// 创建HTTP请求
	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	// 设置请求头
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	// 记录完整的请求信息
	mylog.Logger.Info("Request details",
		zap.String("url", url),
		zap.String("method", httpReq.Method),
		zap.String("content_type", httpReq.Header.Get("Content-Type")),
		zap.String("authorization", "Bearer "+token[:10]+"..."), // 只显示token的前10个字符
		zap.String("request_body", string(jsonData)))

	// 发送请求
	resp, err := client.Do(httpReq)
	if err != nil {
		mylog.Logger.Error("Failed to send request",
			zap.Error(err),
			zap.String("url", url))
		return nil, err
	}
	defer resp.Body.Close()

	// 记录响应头信息
	mylog.Logger.Info("Response headers",
		zap.Int("status_code", resp.StatusCode),
		zap.String("content_type", resp.Header.Get("Content-Type")),
		zap.String("content_length", resp.Header.Get("Content-Length")),
		zap.Any("all_headers", resp.Header))

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		mylog.Logger.Error("API request failed",
			zap.Int("status_code", resp.StatusCode),
			zap.String("body", string(body)))
		return nil, fmt.Errorf("API request failed with status code: %d, body: %s", resp.StatusCode, string(body))
	}

	// 处理响应数据
	responseStr := string(body)
	mylog.Logger.Debug("Raw response", zap.String("body", responseStr))

	// 如果响应以 "data:" 开头，需要去掉这个前缀
	if strings.HasPrefix(responseStr, "data:") {
		responseStr = strings.TrimPrefix(responseStr, "data:")
		// 去掉可能存在的换行符
		responseStr = strings.TrimSpace(responseStr)
	}

	// 解析响应
	var response ChatCompletionResponse
	if err := json.Unmarshal([]byte(responseStr), &response); err != nil {
		mylog.Logger.Error("Failed to parse response",
			zap.Error(err),
			zap.String("body", responseStr))
		return nil, err
	}

	mylog.Logger.Debug("Received response from JiuTian API",
		zap.Any("usage", response.Usage),
		zap.String("response", response.Response),
		zap.String("finished", response.Finished))

	return &response, nil
}

// CreateCompletionStream 发送流式请求
func (r *ChatCompletionRequest) CreateCompletionStream() (*http.Response, error) {
	mylog.Logger.Info("Creating JiuTian stream chat completion")

	// 验证请求参数
	if err := r.validate(); err != nil {
		return nil, err
	}

	// 生成token
	token, err := r.generateToken()
	if err != nil {
		mylog.Logger.Error("Failed to generate token", zap.Error(err))
		return nil, err
	}

	// 准备请求URL
	url := fmt.Sprintf("%s/completions", r.baseURL)
	mylog.Logger.Info("Accessing JiuTian API Stream", zap.String("full_url", url))

	// 准备请求数据
	jsonData, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}

	// 创建HTTP客户端
	client := &http.Client{
		Timeout: DefaultTimeout,
	}
	
	// 设置Transport
	if r.transport != nil {
		client.Transport = r.transport
	} else {
		client.Transport = http.DefaultTransport
	}

	// 创建HTTP请求
	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	// 设置请求头
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("Cache-Control", "no-cache")
	httpReq.Header.Set("Connection", "keep-alive")

	// 记录完整的请求信息
	mylog.Logger.Info("Stream request details",
		zap.String("url", url),
		zap.String("method", httpReq.Method),
		zap.String("content_type", httpReq.Header.Get("Content-Type")),
		zap.String("accept", httpReq.Header.Get("Accept")),
		zap.String("cache_control", httpReq.Header.Get("Cache-Control")),
		zap.String("connection", httpReq.Header.Get("Connection")),
		zap.String("authorization", "Bearer "+token[:10]+"..."), // 只显示token的前10个字符
		zap.String("request_body", string(jsonData)))

	// 发送请求
	resp, err := client.Do(httpReq)
	if err != nil {
		mylog.Logger.Error("Failed to send stream request",
			zap.Error(err),
			zap.String("url", url))
		return nil, err
	}

	// 记录响应头信息
	mylog.Logger.Info("Stream response headers",
		zap.Int("status_code", resp.StatusCode),
		zap.String("content_type", resp.Header.Get("Content-Type")),
		zap.String("transfer_encoding", resp.Header.Get("Transfer-Encoding")),
		zap.Any("all_headers", resp.Header))

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		mylog.Logger.Error("Stream API request failed",
			zap.Int("status_code", resp.StatusCode))
		return nil, fmt.Errorf("API request failed with status code: %d", resp.StatusCode)
	}

	return resp, nil
} 