package handler

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"simple-one-api/pkg/adapter"
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/llm/jiutian"
	"simple-one-api/pkg/mylog"
	"simple-one-api/pkg/utils"
	"go.uber.org/zap"
)

// OpenAI2JiuTianHandler 处理OpenAI到九天模型的请求转换
func OpenAI2JiuTianHandler(c *gin.Context, oaiReqParam *OAIRequestParam) error {
	mylog.Logger.Info("Starting JiuTian request handling")
	
	oaiReq := oaiReqParam.chatCompletionReq
	s := oaiReqParam.modelDetails

	// 获取API Key
	apiKey, _ := utils.GetStringFromMap(oaiReqParam.creds, config.KEYNAME_API_KEY)
	if apiKey == "" {
		return fmt.Errorf("API key not found")
	}

	// 转换请求
	jiutianReq := adapter.OpenAIRequestToJiuTianRequest(oaiReq)
	
	// 分别设置各个参数
	jiutianReq.WithAPIKey(apiKey)
	jiutianReq.WithBaseURL(s.ServerURL)
	
	// 确保transport被正确设置
	if oaiReqParam.httpTransport != nil {
		mylog.Logger.Debug("Setting custom transport for JiuTian request")
		jiutianReq.WithTransport(oaiReqParam.httpTransport)
	} else {
		mylog.Logger.Debug("Using default transport for JiuTian request")
		jiutianReq.WithTransport(http.DefaultTransport)
	}

	// 处理流式请求
	if oaiReq.Stream {
		return handleJiuTianStreamRequest(c, jiutianReq, oaiReqParam.ClientModel)
	}

	// 处理非流式请求
	return handleJiuTianNonStreamRequest(c, jiutianReq, oaiReqParam.ClientModel)
}

// handleJiuTianStreamRequest 处理流式请求
func handleJiuTianStreamRequest(c *gin.Context, jiutianReq *jiutian.ChatCompletionRequest, clientModel string) error {
	mylog.Logger.Info("Handling JiuTian stream request")

	// 记录请求头信息
	mylog.Logger.Info("Original request headers",
		zap.Any("headers", c.Request.Header))

	// 记录请求内容
	reqData, _ := json.Marshal(jiutianReq)
	mylog.Logger.Info("Request content",
		zap.String("request_body", string(reqData)))

	// 发送流式请求
	resp, err := jiutianReq.CreateCompletionStream()
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 设置响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	// 记录设置的响应头
	mylog.Logger.Info("Set response headers",
		zap.String("content_type", c.Writer.Header().Get("Content-Type")),
		zap.String("cache_control", c.Writer.Header().Get("Cache-Control")),
		zap.String("connection", c.Writer.Header().Get("Connection")))

	// 读取并转发响应
	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		// 处理SSE数据
		if bytes.HasPrefix(line, []byte("data: ")) {
			data := bytes.TrimPrefix(line, []byte("data: "))
			// 去掉可能存在的换行符
			data = bytes.TrimSpace(data)
			
			mylog.Logger.Info("Received stream data", 
				zap.String("raw_data", string(data)),
				zap.Int("data_length", len(data)))

			var jiutianResp jiutian.ChatCompletionStreamResponse
			if err := json.Unmarshal(data, &jiutianResp); err != nil {
				mylog.Logger.Error("Failed to parse stream response",
					zap.Error(err),
					zap.String("data", string(data)))
				continue
			}

			// 记录解析后的九天响应
			mylog.Logger.Info("Parsed JiuTian response",
				zap.Any("jiutian_response", map[string]interface{}{
					"response": jiutianResp.Response,
					"delta":    jiutianResp.Delta,
					"finished": jiutianResp.Finished,
					"history":  jiutianResp.History,
				}))

			// 转换为OpenAI流式响应
			streamResp := adapter.JiuTianStreamResponseToOpenAIStreamResponse(&jiutianResp)
			streamResp.Model = clientModel

			// 发送响应
			responseData, _ := json.Marshal(streamResp)
			mylog.Logger.Info("Sending stream response",
				zap.String("response_data", string(responseData)))

			c.Writer.Write([]byte("data: "))
			c.Writer.Write(responseData)
			c.Writer.Write([]byte("\n\n"))
			c.Writer.Flush()
		}
	}

	return nil
}

// handleJiuTianNonStreamRequest 处理非流式请求
func handleJiuTianNonStreamRequest(c *gin.Context, jiutianReq *jiutian.ChatCompletionRequest, clientModel string) error {
	mylog.Logger.Info("Handling JiuTian non-stream request")

	// 记录请求头信息
	mylog.Logger.Info("Original request headers",
		zap.Any("headers", c.Request.Header))

	// 记录请求内容
	reqData, _ := json.Marshal(jiutianReq)
	mylog.Logger.Info("Request content",
		zap.String("request_body", string(reqData)))

	// 发送请求
	jiutianResp, err := jiutianReq.CreateCompletion()
	if err != nil {
		return err
	}

	// 记录九天响应
	mylog.Logger.Info("Received JiuTian response",
		zap.Any("jiutian_response", map[string]interface{}{
			"usage":    jiutianResp.Usage,
			"response": jiutianResp.Response,
			"delta":    jiutianResp.Delta,
			"finished": jiutianResp.Finished,
			"history":  jiutianResp.History,
		}))

	// 转换为OpenAI响应
	chatResp := adapter.JiuTianResponseToOpenAIResponse(jiutianResp)
	chatResp.Model = clientModel

	// 记录响应信息
	responseData, _ := json.Marshal(chatResp)
	mylog.Logger.Info("Sending final response",
		zap.String("response_data", string(responseData)))

	// 发送响应
	c.JSON(http.StatusOK, chatResp)
	return nil
} 