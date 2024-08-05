package handler

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	hunyuan "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/hunyuan/v20230901"
	"net/http"
	"simple-one-api/pkg/adapter"
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/mylog"
	"simple-one-api/pkg/utils"
)

func OpenAI2HunYuanHandler(c *gin.Context, oaiReqParam *OAIRequestParam) error {
	// 创建认证对象
	oaiReq := oaiReqParam.chatCompletionReq
	//s := oaiReqParam.modelDetails
	credentials := oaiReqParam.creds
	secretId, _ := utils.GetStringFromMap(credentials, config.KEYNAME_SECRET_ID)
	secretKey, _ := utils.GetStringFromMap(credentials, config.KEYNAME_SECRET_KEY)
	credential := common.NewCredential(
		secretId,
		secretKey,
	)

	// 创建客户端配置
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "hunyuan.tencentcloudapi.com"

	// 创建HunYuan客户端
	client, err := hunyuan.NewClient(credential, "", cpf)
	if err != nil {
		mylog.Logger.Error(err.Error())
		return err
	}

	if oaiReqParam.httpTransport != nil {
		client.Client.WithHttpTransport(oaiReqParam.httpTransport)
	}

	// 创建HunYuan请求对象
	request := adapter.OpenAIRequestToHunYuanRequest(oaiReq)

	// 打印请求数据
	djData, _ := json.Marshal(request)
	mylog.Logger.Info(string(djData))

	// 发送请求并处理响应
	response, err := client.ChatCompletions(request)
	if err != nil {
		mylog.Logger.Error(err.Error())
		return err
	}

	// 处理响应数据
	return handleHunYuanResponse(c, response, oaiReq.Model, oaiReqParam)
}

// handleHunYuanResponse 处理HunYuan的响应数据
func handleHunYuanResponse(c *gin.Context, response *hunyuan.ChatCompletionsResponse, model string, oaiReqParam *OAIRequestParam) error {
	if response.Response != nil {
		// 非流式响应
		return handleHunYuanNonStreamResponse(c, response, model, oaiReqParam)
	}

	// 流式响应
	utils.SetEventStreamHeaders(c)
	for event := range response.Events {
		oaiStreamResp, err := adapter.HunYuanResponseToOpenAIStreamResponse(event)
		if err != nil {
			mylog.Logger.Error(err.Error())
			return err
		}
		oaiStreamResp.Model = oaiReqParam.ClientModel
		respData, err := json.Marshal(&oaiStreamResp)
		if err != nil {
			mylog.Logger.Error(err.Error())
			return err
		}
		mylog.Logger.Info(string(respData))
		_, err = c.Writer.WriteString("data: " + string(respData) + "\n\n")
		if err != nil {
			mylog.Logger.Error(err.Error())
			return err
		}
		c.Writer.(http.Flusher).Flush()
	}
	return nil
}

// handleNonStreamResponse 处理非流式响应
func handleHunYuanNonStreamResponse(c *gin.Context, response *hunyuan.ChatCompletionsResponse, model string, oaiReqParam *OAIRequestParam) error {
	oaiResp := adapter.HunYuanResponseToOpenAIResponse(response)
	oaiResp.Model = oaiReqParam.ClientModel

	jdata, _ := json.Marshal(*oaiResp)
	mylog.Logger.Info(string(jdata))
	c.JSON(http.StatusOK, oaiResp)
	return nil
}
