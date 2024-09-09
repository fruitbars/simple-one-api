package handler

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"
	"simple-one-api/pkg/adapter"
	"simple-one-api/pkg/config"
	baiduqianfan "simple-one-api/pkg/llm/baidu-qianfan"
	"simple-one-api/pkg/mylog"
	"simple-one-api/pkg/utils"
)

func OpenAI2QianFanHandler(c *gin.Context, oaiReqParam *OAIRequestParam) error {

	oaiReq := oaiReqParam.chatCompletionReq
	//s := oaiReqParam.modelDetails
	credentials := oaiReqParam.creds
	apiKey, _ := utils.GetStringFromMap(credentials, config.KEYNAME_API_KEY)
	secretKey, _ := utils.GetStringFromMap(credentials, config.KEYNAME_SECRET_KEY)
	configAddress, _ := utils.GetStringFromMap(credentials, config.KEYNAME_ADDRESSS)
	qfReq := adapter.OpenAIRequestToQianFanRequest(oaiReq)

	client := &http.Client{}
	if oaiReqParam.httpTransport != nil {
		client.Transport = oaiReqParam.httpTransport
	}

	clientModel := oaiReqParam.ClientModel

	if oaiReq.Stream {
		return handleQianFanStreamRequest(c, client, apiKey, secretKey, oaiReq.Model, clientModel, configAddress, qfReq)
	} else {
		return handleQianFanStandardRequest(c, client, apiKey, secretKey, oaiReq.Model, clientModel, configAddress, qfReq)
	}
}

func handleQianFanStreamRequest(c *gin.Context, client *http.Client, apiKey, secretKey, model string, clientModel string, configAddress string, qfReq *baiduqianfan.QianFanRequest) error {
	utils.SetEventStreamHeaders(c)

	err := baiduqianfan.QianFanCallSSE(client, apiKey, secretKey, model, configAddress, qfReq, func(qfResp *baiduqianfan.QianFanResponse) {
		oaiRespStream := adapter.QianFanResponseToOpenAIStreamResponse(qfResp)
		oaiRespStream.Model = clientModel

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

func handleQianFanStandardRequest(c *gin.Context, client *http.Client, apiKey, secretKey, model string, clientModel string, configAddress string, qfReq *baiduqianfan.QianFanRequest) error {
	qfResp, err := baiduqianfan.QianFanCall(client, apiKey, secretKey, model, configAddress, qfReq)
	if err != nil {
		mylog.Logger.Error("Error during API call",
			zap.Error(err)) // 记录错误对象

		return err
	}

	oaiResp := adapter.QianFanResponseToOpenAIResponse(qfResp)
	oaiResp.Model = clientModel
	mylog.Logger.Info("Standard response",
		zap.Any("response", oaiResp)) // 记录标准响应对象

	c.JSON(http.StatusOK, oaiResp)
	return nil
}
