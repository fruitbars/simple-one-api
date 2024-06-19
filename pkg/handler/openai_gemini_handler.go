package handler

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"simple-one-api/pkg/mylog"

	//"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sashabaranov/go-openai"
	"simple-one-api/pkg/adapter"
	"simple-one-api/pkg/config"
	googlegemini "simple-one-api/pkg/llm/google-gemini"
	"simple-one-api/pkg/utils"
)

// 定义常量
const (
	BaseURL        = "https://generativelanguage.googleapis.com/v1beta/models"
	RequestTimeout = 5 * time.Minute
)

// 使用全局客户端
var httpClient = &http.Client{
	Timeout: RequestTimeout,
}

// OpenAI2GeminiHandler 主要的处理函数
func OpenAI2GeminiHandler(c *gin.Context, s *config.ModelDetails, oaiReq openai.ChatCompletionRequest) error {
	geminiReq := adapter.OpenAIRequestToGeminiRequest(oaiReq)
	jsonData, err := json.Marshal(geminiReq)
	if err != nil {
		logError("marshalling data", err)
		return err
	}

	apiKey := s.Credentials[config.KEYNAME_API_KEY]
	geminiURL := fmt.Sprintf("%s/%s:%s%s", BaseURL, oaiReq.Model, getRequestType(oaiReq.Stream), apiKey)

	mylog.Debug(geminiURL)

	req, err := http.NewRequest("POST", geminiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		logError("creating request", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		logError("sending request", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errMsg, _ := io.ReadAll(resp.Body)
		return errors.New(string(errMsg))
	}

	if oaiReq.Stream {
		return handleStreamResponse(c, resp)
	}
	return handleRegularResponse(c, resp)
}

// 处理流响应
func handleStreamResponse(c *gin.Context, resp *http.Response) error {
	utils.SetEventStreamHeaders(c)
	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			logError("reading response", err)
			return err
		}

		if strings.HasPrefix(line, "data: ") {
			if err := processAndSendData(c, line); err != nil {
				return err
			}
		}
	}
	return nil
}

// 处理常规响应
func handleRegularResponse(c *gin.Context, resp *http.Response) error {
	responseBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		logError("reading response", err)
		return err
	}

	if resp.StatusCode != 200 {
		return errors.New(string(responseBytes))
	}

	var geminiResp googlegemini.GeminiResponse
	if err := json.Unmarshal(responseBytes, &geminiResp); err != nil {
		logError("unmarshalling response", err)
		return err
	}

	c.JSON(http.StatusOK, adapter.GeminiResponseToOpenAIResponse(&geminiResp))
	return nil
}

// 处理并发送流数据
func processAndSendData(c *gin.Context, line string) error {
	data := strings.TrimPrefix(line, "data: ")
	data = strings.TrimSpace(data)
	if data == "" {
		return nil
	}
	var response googlegemini.GeminiResponse
	if err := json.Unmarshal([]byte(data), &response); err != nil {
		logError("unmarshalling response", err)
		return err
	}

	respData, err := json.Marshal(adapter.GeminiResponseToOpenAIStreamResponse(&response))
	if err != nil {
		logError("marshaling response", err)
		return err
	}

	mylog.Debug(string(respData))

	if _, err := c.Writer.WriteString("data: " + string(respData) + "\n\n"); err != nil {
		logError("writing response", err)
		return err
	}
	c.Writer.(http.Flusher).Flush()
	return nil
}

// 日志错误
func logError(message string, err error) {
	mylog.Error("Error %s: %v\n", message, err)
}

// 获取请求类型，决定是流还是非流
func getRequestType(isStream bool) string {
	if isStream {
		// 使用 & 连接键值对
		return "streamGenerateContent?alt=sse&key="
	}
	return "generateContent?key="
}
