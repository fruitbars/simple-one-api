package handler

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	hunyuan "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/hunyuan/v20230901"
	"log"
	"net/http"
	"simple-one-api/pkg/adapter"
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/openai"
	mycommon "simple-one-api/pkg/utils"
)

func OpenAI2HunYuanHander(c *gin.Context, s *config.ModelDetails, oaiReq openai.OpenAIRequest) error {
	// 实例化一个认证对象，入参需要传入腾讯云账户 SecretId 和 SecretKey，此处还需注意密钥对的保密
	// 代码泄露可能会导致 SecretId 和 SecretKey 泄露，并威胁账号下所有资源的安全性。以下代码示例仅供参考，建议采用更安全的方式来使用密钥，请参见：https://cloud.tencent.com/document/product/1278/85305
	// 密钥可前往官网控制台 https://console.cloud.tencent.com/cam/capi 进行获取
	secretId := s.Credentials["secret_id"]
	secretKey := s.Credentials["secret_key"]
	credential := common.NewCredential(
		secretId,
		secretKey,
	)
	// 实例化一个client选项，可选的，没有特殊需求可以跳过
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "hunyuan.tencentcloudapi.com"
	// 实例化要请求产品的client对象,clientProfile是可选的
	client, _ := hunyuan.NewClient(credential, "", cpf)

	// 实例化一个请求对象,每个接口都会对应一个request对象
	request := adapter.OpenAIRequestToHunYuanRequest(oaiReq)

	djData, _ := json.Marshal(request)
	log.Println("huanyuan request:", string(djData))

	// 返回的resp是一个ChatCompletionsResponse的实例，与请求对象对应
	response, err := client.ChatCompletions(request)
	if _, ok := err.(*errors.TencentCloudSDKError); ok {
		log.Println("An API error has returned:", err)
		return err
	}

	// 输出json格式的字符串回包
	if response.Response != nil {
		// 非流式响应
		oaiResp := adapter.HunYuanResponseToOpenAIResponse(response)
		log.Println(oaiResp)
		oaiResp.Model = oaiReq.Model
		// 设置响应的内容类型并发送JSON响应
		c.JSON(http.StatusOK, oaiResp)

	} else {
		mycommon.SetEventStreamHeaders(c)
		// 流式响应
		for event := range response.Events {

			oaiStreamResp, err := adapter.HunYuanResponseToOpenAIStreamResponse(event)
			if err != nil {
				log.Println(err)
				return err
			}
			oaiStreamResp.Model = oaiReq.Model
			respData, err := json.Marshal(&oaiStreamResp)
			c.Writer.WriteString("data: " + string(respData) + "\n\n")
			c.Writer.(http.Flusher).Flush()
		}

	}

	return nil
}
