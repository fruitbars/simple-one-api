package handler

import (
	"encoding/json"
	"fmt"
	"github.com/fruitbars/gosparkclient"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"
	"simple-one-api/pkg/adapter"
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/mylog"
	"simple-one-api/pkg/utils"
	"strings"
)

// getURLAndDomain 根据模型名称返回相应的 URL 地址和 domain 参数
func getURLAndDomain(modelName string) (string, string, error) {
	modelNameLower := strings.ToLower(modelName)

	switch modelNameLower {
	case "4.0ultra":
		return "wss://spark-api.xf-yun.com/v4.0/chat", "4.0Ultra", nil
	case "spark-max32k", "max-32k":
		return "wss://spark-api.xf-yun.com/chat/max-32k", "generalv3.5", nil
	case "spark3.5-max", "spark-max", "generalv3.5":
		return "wss://spark-api.xf-yun.com/v3.5/chat", "generalv3.5", nil
	case "spark-pro", "generalv3":
		return "wss://spark-api.xf-yun.com/v3.1/chat", "generalv3", nil
	case "spark-v2.0", "generalv2":
		return "wss://spark-api.xf-yun.com/v2.1/chat", "generalv2", nil
		//case "spark-lite", "general":
		//return "wss://spark-api.xf-yun.com/v1.1/chat", "general", nil
	case "spark-lite", "general":
		return "wss://spark-api.xf-yun.com/v1.1/chat", "lite", nil
	case "spark-pro-128k":
		return "wss://spark-api.xf-yun.com/chat/pro-128k", "pro-128k", nil
	default:
		return "", "", fmt.Errorf("unsupported model name: %s", modelName)
	}
}

func OpenAI2XingHuoHandler(c *gin.Context, oaiReqParam *OAIRequestParam) error {
	oaiReq := oaiReqParam.chatCompletionReq
	s := oaiReqParam.modelDetails
	credentials := oaiReqParam.creds
	appid, _ := utils.GetStringFromMap(credentials, config.KEYNAME_APPID)
	apiKey, _ := utils.GetStringFromMap(credentials, config.KEYNAME_API_KEY)
	apiSecret, _ := utils.GetStringFromMap(credentials, config.KEYNAME_API_SECRET)

	serverUrl, domain, err := getServerURLAndDomain(s.ServerURL, credentials, oaiReq.Model)
	if err != nil {
		return err
	}

	//mycommon.GetCredentialsLimit()

	client := gosparkclient.NewSparkClientWithOptions(appid, apiKey, apiSecret, serverUrl, domain)
	if oaiReqParam.httpTransport != nil {
		client.Transport = oaiReqParam.httpTransport
	}

	xhReq := adapter.OpenAIRequestToXingHuoRequest(oaiReq)

	xhDataJson, _ := json.Marshal(xhReq)
	mylog.Logger.Info(string(xhDataJson))

	clientModel := oaiReqParam.ClientModel
	if oaiReq.Stream {
		return handleXingHuoStreamMode(c, client, xhReq, clientModel)
	}

	return handleXingHuoStandardMode(c, client, xhReq, clientModel)
}

func getServerURLAndDomain(configServerURL string, credentials map[string]interface{}, model string) (string, string, error) {

	serverUrl, defaultDomain, err := getURLAndDomain(model)
	if err != nil {
		mylog.Logger.Warn("error", zap.Error(err)) // 记录错误对象
	}

	if configServerURL != "" {
		serverUrl = configServerURL
	}

	domain, _ := utils.GetStringFromMap(credentials, config.KEYNAME_DOMAIN)
	if domain == "" {
		domain = defaultDomain
	}

	return serverUrl, domain, nil
}

func handleXingHuoStreamMode(c *gin.Context, client *gosparkclient.SparkClient, xhReq *gosparkclient.SparkChatRequest, model string) error {
	utils.SetEventStreamHeaders(c)

	_, err := client.SparkChatWithCallback(*xhReq, func(response gosparkclient.SparkAPIResponse) {

		oaiRespStream := adapter.XingHuoResponseToOpenAIStreamResponse(&response)
		oaiRespStream.Model = model

		respData, err := json.Marshal(&oaiRespStream)
		if err != nil {
			mylog.Logger.Error("Error marshaling response:", zap.Error(err))
			return
		}

		// 假设 mylog.Logger 是一个已经配置好的 zap.Logger 实例
		mylog.Logger.Debug("SparkChatWithCallback Response HTTP data",
			zap.String("data", string(respData))) // 记录响应数据

		if oaiRespStream.Error != nil {
			// 假设 mylog.Logger 是一个已经配置好的 zap.Logger 实例
			mylog.Logger.Error("Error response",
				zap.Any("error", *oaiRespStream.Error)) // 记录错误对象

			return
		}

		_, err = c.Writer.WriteString("data: " + string(respData) + "\n\n")
		if err != nil {
			// 假设 mylog.Logger 是一个已经配置好的 zap.Logger 实例
			mylog.Logger.Error("An error occurred",
				zap.Error(err)) // 记录错误对象

			return
		}
		c.Writer.(http.Flusher).Flush()
	})

	return err
}

func handleXingHuoStandardMode(c *gin.Context, client *gosparkclient.SparkClient, xhReq *gosparkclient.SparkChatRequest, model string) error {
	xhResp, err := client.SparkChatWithCallback(*xhReq, nil)
	if err != nil {

		mylog.Logger.Error("An error occurred", zap.String("appid", client.AppID),
			zap.String("apikey", client.ApiKey),
			zap.Error(err))

		return err
	}

	oaiResp := adapter.XingHuoResponseToOpenAIResponse(xhResp)
	oaiResp.Model = model

	// 假设 mylog.Logger 是一个已经配置好的 zap.Logger 实例
	mylog.Logger.Info("Standard response",
		zap.Any("response", *oaiResp)) // 记录响应对象

	c.JSON(http.StatusOK, oaiResp)
	return nil
}
