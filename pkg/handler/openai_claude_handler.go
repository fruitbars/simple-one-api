package handler

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"io"
	"net/http"
	"simple-one-api/pkg/adapter"
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/llm/claude"
	"simple-one-api/pkg/mycommon"
	"simple-one-api/pkg/mylog"
	myopenai "simple-one-api/pkg/openai"
	"simple-one-api/pkg/utils"
	"strings"
	"time"
)

var defaultClaudeServerURL = "https://api.anthropic.com/v1/messages"

func OpenAI2ClaudeHandler(c *gin.Context, oaiReqParam *OAIRequestParam) error {
	oaiReq := oaiReqParam.chatCompletionReq
	s := oaiReqParam.modelDetails
	credentials := oaiReqParam.creds

	apiKey, _ := utils.GetStringFromMap(credentials, config.KEYNAME_API_KEY)

	claudeReq := adapter.OpenAIRequestToClaudeRequest(oaiReq)

	claudeServerURL := s.ServerURL

	if claudeServerURL == "" {
		claudeServerURL = defaultClaudeServerURL
	}

	client := &http.Client{
		Timeout: 3 * time.Minute,
	}
	if oaiReqParam.httpTransport != nil {
		client.Transport = oaiReqParam.httpTransport
	}

	mylog.Logger.Info("OpenAI2ClaudeHandler", zap.Any("claudeReq", claudeReq))
	// 使用统一的错误处理函数
	if err := sendClaudeRequest(c, client, apiKey, claudeServerURL, claudeReq, oaiReq, oaiReqParam); err != nil {
		mylog.Logger.Error(err.Error(), zap.String("claudeServerURL", claudeServerURL),
			zap.Any("claudeReq", claudeReq), zap.Any("oaiReq", oaiReq))
		return err
	}

	return nil
}

func sendClaudeRequest(c *gin.Context, client *http.Client, apiKey, url string, request interface{}, oaiReq *openai.ChatCompletionRequest, oaiReqParam *OAIRequestParam) error {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("json编码错误: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		mylog.Logger.Error(err.Error())
		return err
	}

	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		mylog.Logger.Error(err.Error())
		return err
	}
	defer resp.Body.Close()

	err = mycommon.CheckStatusCode(resp)
	if err != nil {
		mylog.Logger.Error("sendClaudeRequest", zap.Error(err))
		return err
	}

	if oaiReq.Stream {
		return handleClaudeStreamResponse(c, resp, oaiReq, oaiReqParam)
	}

	return handleClaudeResponse(c, resp, oaiReq, oaiReqParam)
}

func handleClaudeResponse(c *gin.Context, resp *http.Response, oaiReq *openai.ChatCompletionRequest, oaiReqParam *OAIRequestParam) error {

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		mylog.Logger.Error(err.Error())
		return err
	}

	mylog.Logger.Info("response", zap.String("body", string(body)))

	var claudeResp claude.ResponseBody
	if err := json.Unmarshal(body, &claudeResp); err != nil {
		mylog.Logger.Error(err.Error())
		return fmt.Errorf("json解码错误: %v", err)
	}

	myresp := adapter.ClaudeReponseToOpenAIResponse(&claudeResp)
	myresp.Model = oaiReqParam.ClientModel
	c.JSON(http.StatusOK, myresp)

	return nil
}

func handleClaudeStreamResponse(c *gin.Context, resp *http.Response, oaiReq *openai.ChatCompletionRequest, oaiReqParam *OAIRequestParam) error {
	reader := bufio.NewReader(resp.Body)

	var eventBuilder strings.Builder
	var dataBuilder strings.Builder

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("读取流响应错误: %v", err)
		}

		lineStr := strings.TrimSpace(string(line))
		if len(lineStr) == 0 {
			// 完整的事件消息读取完毕，发送SSE消息
			if eventBuilder.Len() > 0 && dataBuilder.Len() > 0 {
				processClaudeStreamEvent(c, eventBuilder.String(), dataBuilder.String(), oaiReqParam.ClientModel)
				// 重置builders
				eventBuilder.Reset()
				dataBuilder.Reset()
			}

			continue
		}

		if strings.HasPrefix(lineStr, "event: ") {
			eventBuilder.WriteString(strings.TrimPrefix(lineStr, "event: "))
		} else if strings.HasPrefix(lineStr, "data: ") {
			dataBuilder.WriteString(strings.TrimPrefix(lineStr, "data: "))
		}
	}

	return nil
}

func processClaudeStreamEvent(c *gin.Context, eventType string, eventData string, clientModel string) error {
	switch eventType {
	case "message_start":
		return handleClaudeEvent(c, eventData, claude.MsgMessageStart{}, adapter.ConvertMsgMessageStartToOpenAIStreamResponse, clientModel)
	case "content_block_delta":
		return handleClaudeEvent(c, eventData, claude.MsgContentBlockDelta{}, adapter.ConvertMsgContentBlockDeltaToOpenAIStreamResponse, clientModel)
	case "content_block_start":
		// 处理content_block_start事件
	case "content_block_stop":
		// 处理content_block_stop事件
	case "message_delta":
		// 处理message_delta事件
	case "message_stop":
		// 处理message_stop事件
	case "ping":
		// 处理ping事件
	default:
		// 可以添加日志来记录未知事件类型
		mylog.Logger.Error("Unknown event type: " + eventType)
	}

	return nil
}

// handleEvent 处理事件的通用逻辑
func handleClaudeEvent[T any](c *gin.Context, eventData string, eventStruct T, converter func(*T) *myopenai.OpenAIStreamResponse, clientModel string) error {
	if err := json.Unmarshal([]byte(eventData), &eventStruct); err != nil {
		mylog.Logger.Error(err.Error())
		return err
	}

	respStruct := converter(&eventStruct)
	respStruct.Model = clientModel
	respData, err := json.Marshal(&respStruct)
	if err != nil {
		mylog.Logger.Error(err.Error())
		return err
	}

	_, err = c.Writer.WriteString("data: " + string(respData) + "\n\n")
	if err != nil {
		mylog.Logger.Error(err.Error())
		return err
	}

	if flusher, ok := c.Writer.(http.Flusher); ok {
		flusher.Flush()
	} else {
		return fmt.Errorf("response writer does not implement http.Flusher")
	}

	return nil
}
