package handler

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"net/http"
	"simple-one-api/pkg/adapter"
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/llm/devplatform/dify/chat_message_request"
	"simple-one-api/pkg/llm/devplatform/dify/chunk_chat_completion_response"
	"simple-one-api/pkg/mylog"
	myopenai "simple-one-api/pkg/openai"
	"simple-one-api/pkg/utils"
	"time"
)

func OpenAI2DifyHandler(c *gin.Context, oaiReqParam *OAIRequestParam) error {
	oaiReq := oaiReqParam.chatCompletionReq
	difyReq := adapter.OpenAIRequestToDifyRequest(oaiReqParam.chatCompletionReq)
	credentials := oaiReqParam.creds

	respID := uuid.New().String()

	apiKey, _ := utils.GetStringFromMap(credentials, config.KEYNAME_API_KEY)

	if oaiReq.Stream == false {

		difyResp, err := chat_message_request.CallChatMessagesNoneStreamMode(difyReq, apiKey, nil)
		if err != nil {
			mylog.Logger.Error(err.Error())
			return err
		}

		// 转换响应
		myresp := adapter.DifyResponseToOpenAIResponse(difyResp)
		myresp.Model = oaiReqParam.ClientModel

		c.JSON(http.StatusOK, myresp)

		return nil
	}

	// 流式处理
	cb := func(eventData string) {
		mylog.Logger.Debug("Received event: " + eventData)
		var commonEvent chunk_chat_completion_response.CommonEvent
		if err := json.Unmarshal([]byte(eventData), &commonEvent); err != nil {
			mylog.Logger.Error("Error parsing common event: " + err.Error())
			return
		}

		// 处理不同的事件类型
		if err := processEvent(c, eventData, oaiReqParam, commonEvent.Event, respID); err != nil {
			mylog.Logger.Error("Error processing event: " + err.Error())
			return
		}
	}

	// 调用流式接口
	if err := chat_message_request.CallChatMessagesStreamMode(difyReq, apiKey, cb, oaiReqParam.httpTransport); err != nil {
		mylog.Logger.Error(err.Error())
		return err
	}

	return nil
}

// 处理不同事件类型的通用函数
func processEvent(c *gin.Context, eventData string, oaiReqParam *OAIRequestParam, eventType string, respID string) error {
	var oaiRespStream *myopenai.OpenAIStreamResponse
	var err error

	// 根据 event 类型解析对应的事件
	switch eventType {
	case "message":
		var messageEvent chunk_chat_completion_response.MessageEvent
		if err = json.Unmarshal([]byte(eventData), &messageEvent); err != nil {
			mylog.Logger.Error(err.Error())
			return err
		}
		oaiRespStream = adapter.DifyResponseToOpenAIResponseStream(&messageEvent)
	case "message_end":
		var messageEndEvent chunk_chat_completion_response.MessageEndEvent
		if err = json.Unmarshal([]byte(eventData), &messageEndEvent); err != nil {
			mylog.Logger.Error(err.Error())
			return err
		}

		mylog.Logger.Debug("processEvent", zap.Any("messageEndEvent", messageEndEvent))
		oaiRespStream = adapter.DifyMessageEndEventToOpenAIResponseStream(&messageEndEvent)
	default:
		// 如果是未知的 event 类型，可以选择忽略或记录错误
		mylog.Logger.Warn("Unknown event type: " + eventType)
		return nil
	}

	oaiRespStream.ID = respID
	oaiRespStream.Object = "chat.completion.chunk"
	oaiRespStream.Created = time.Now().Unix()

	// 设置模型
	oaiRespStream.Model = oaiReqParam.ClientModel

	// 将响应数据写入客户端
	return writeResponse(c, oaiRespStream)
}

// 将响应数据写入客户端的辅助函数
func writeResponse(c *gin.Context, oaiRespStream interface{}) error {
	respData, err := json.Marshal(oaiRespStream)
	if err != nil {
		return err
	}

	mylog.Logger.Info(string(respData))

	_, err = c.Writer.WriteString("data: " + string(respData) + "\n\n")
	if err != nil {
		return err
	}

	// 确保响应被及时发送
	c.Writer.(http.Flusher).Flush()
	return nil
}
