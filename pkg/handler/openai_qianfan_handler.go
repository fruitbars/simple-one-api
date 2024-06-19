package handler

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"net/http"
	"simple-one-api/pkg/adapter"
	"simple-one-api/pkg/config"
	baidu_qianfan "simple-one-api/pkg/llm/baidu-qianfan"
	"simple-one-api/pkg/mylog"
	"simple-one-api/pkg/utils"
)

func OpenAI2QianFanHandler(c *gin.Context, s *config.ModelDetails, oaiReq openai.ChatCompletionRequest) error {
	apiKey := s.Credentials[config.KEYNAME_API_KEY]
	secretKey := s.Credentials[config.KEYNAME_SECRET_KEY]
	qfReq := adapter.OpenAIRequestToQianFanRequest(oaiReq)

	if oaiReq.Stream {
		return handleQianFanStreamRequest(c, apiKey, secretKey, oaiReq.Model, qfReq)
	} else {
		return handleQianFanStandardRequest(c, apiKey, secretKey, oaiReq.Model, qfReq)
	}
}

func handleQianFanStreamRequest(c *gin.Context, apiKey, secretKey, model string, qfReq *baidu_qianfan.QianFanRequest) error {
	utils.SetEventStreamHeaders(c)

	err := baidu_qianfan.QianFanCallSSE(apiKey, secretKey, model, qfReq, func(qfResp *baidu_qianfan.QianFanResponse) {
		oaiRespStream := adapter.QianFanResponseToOpenAIStreamResponse(qfResp)
		oaiRespStream.Model = model

		respData, err := json.Marshal(&oaiRespStream)
		if err != nil {
			mylog.Logger.Error("Error marshaling response",
				zap.Error(err)) // 记录错误对象

			return
		}

		mylog.Logger.Info("Response HTTP data",
			zap.String("http_data", string(respData))) // 记录 HTTP 响应数据

		if qfResp.ErrorCode != 0 && oaiRespStream.Error != nil {
			mylog.Logger.Error("Error response",
				zap.Any("error", *oaiRespStream.Error)) // 记录错误对象

			c.JSON(http.StatusBadRequest, qfResp)
			return
		}

		c.Writer.WriteString("data: " + string(respData) + "\n\n")
		c.Writer.(http.Flusher).Flush()
	})

	if err != nil {
		mylog.Logger.Error("Error during SSE call",
			zap.Error(err)) // 记录错误对象

		return err
	}

	return nil
}

func handleQianFanStandardRequest(c *gin.Context, apiKey, secretKey, model string, qfReq *baidu_qianfan.QianFanRequest) error {
	qfResp, err := baidu_qianfan.QianFanCall(apiKey, secretKey, model, qfReq)
	if err != nil {
		mylog.Logger.Error("Error during API call",
			zap.Error(err)) // 记录错误对象

		return err
	}

	oaiResp := adapter.QianFanResponseToOpenAIResponse(qfResp)
	oaiResp.Model = model
	mylog.Logger.Info("Standard response",
		zap.Any("response", oaiResp)) // 记录标准响应对象

	c.JSON(http.StatusOK, oaiResp)
	return nil
}
