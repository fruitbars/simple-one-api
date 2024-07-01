package handler

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"io"
	"regexp"
	"simple-one-api/pkg/mycommon"
	"simple-one-api/pkg/mylog"

	//"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"simple-one-api/pkg/adapter"
	"simple-one-api/pkg/config"
	googlegemini "simple-one-api/pkg/llm/google-gemini"
	"simple-one-api/pkg/utils"
)

// 定义常量
const (
	BaseURL        = "https://generativelanguage.googleapis.com/v1beta/models"
	RequestTimeout = 1 * time.Minute
)

// 使用全局客户端
var httpClient = &http.Client{
	Timeout: RequestTimeout,
}

// OpenAI2GeminiHandler 主要的处理函数
func OpenAI2GeminiHandler(c *gin.Context, oaiReqParam *OAIRequestParam) error {
	oaiReq := oaiReqParam.chatCompletionReq
	//s := oaiReqParam.modelDetails
	credentials := oaiReqParam.creds

	//mylog.Logger.Info("oaiReq", zap.Any("oaiReq", oaiReq))
	geminiReq := adapter.OpenAIRequestToGeminiRequest(oaiReq)

	debugGeminiReq, _ := adapter.DeepCopyGeminiRequest(geminiReq)
	mylog.Logger.Info("debugGeminiReq", zap.Any("debugGeminiReq", debugGeminiReq))

	jsonData, err := json.Marshal(geminiReq)
	if err != nil {
		mylog.Logger.Error(err.Error())
		return err
	}

	apiKey, _ := utils.GetStringFromMap(credentials, config.KEYNAME_API_KEY)
	geminiURL := fmt.Sprintf("%s/%s:%s%s", BaseURL, oaiReq.Model, getRequestType(oaiReq.Stream), apiKey)

	mylog.Logger.Debug(geminiURL)
	//mylog.Logger.Debug(string(jsonData))

	req, err := http.NewRequest("POST", geminiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		mylog.Logger.Error(err.Error())
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		errStr := err.Error()
		re := regexp.MustCompile(`key=[^&]*`)
		outputErr := re.ReplaceAllString(errStr, "key=***")

		mylog.Logger.Error(outputErr, zap.Error(err))
		return err
	}
	defer resp.Body.Close()

	err = mycommon.CheckStatusCode(resp)
	if err != nil {
		mylog.Logger.Error(err.Error())
		return err
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
			mylog.Logger.Error(err.Error())
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
		mylog.Logger.Error(err.Error())
		return err
	}

	mylog.Logger.Info(string(responseBytes))

	if resp.StatusCode != 200 {
		mylog.Logger.Error(string(responseBytes))
		return errors.New(string(responseBytes))
	}

	var geminiResp googlegemini.GeminiResponse
	if err := json.Unmarshal(responseBytes, &geminiResp); err != nil {
		mylog.Logger.Error(err.Error())
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

	mylog.Logger.Debug("process genimi data:", zap.String("data", data))

	var response googlegemini.GeminiResponse
	if err := json.Unmarshal([]byte(data), &response); err != nil {
		mylog.Logger.Error(err.Error())
		return err
	}

	respData, err := json.Marshal(adapter.GeminiResponseToOpenAIStreamResponse(&response))
	if err != nil {
		mylog.Logger.Error(err.Error())
		return err
	}

	mylog.Logger.Info(string(respData))

	if _, err := c.Writer.WriteString("data: " + string(respData) + "\n\n"); err != nil {
		mylog.Logger.Warn(err.Error())
	}
	c.Writer.(http.Flusher).Flush()
	return nil
}

// 获取请求类型，决定是流还是非流
func getRequestType(isStream bool) string {
	if isStream {
		// 使用 & 连接键值对
		return "streamGenerateContent?alt=sse&key="
	}
	return "generateContent?key="
}
