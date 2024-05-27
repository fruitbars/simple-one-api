package handler

import (
	"encoding/json"
	"fmt"
	"github.com/fruitbars/gosparkclient"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"simple-one-api/pkg/adapter"
	"simple-one-api/pkg/common"
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/openai"
	"strings"
)

// getURLAndDomain 根据模型名称返回相应的 URL 地址和 domain 参数
func getURLAndDomain(modelName string) (string, string, error) {
	// 将模型名称转换为小写，便于不区分大小写的比较
	modelNameLower := strings.ToLower(modelName)

	// 根据模型名称匹配对应的 URL 地址和 domain 参数
	switch modelNameLower {
	case "spark3.5-max":
		return "wss://spark-api.xf-yun.com/v3.5/chat", "generalv3.5", nil
	case "spark-pro":
		return "wss://spark-api.xf-yun.com/v3.1/chat", "generalv3", nil
	case "spark-v2.0":
		return "wss://spark-api.xf-yun.com/v2.1/chat", "generalv2", nil
	case "spark-lite":
		return "wss://spark-api.xf-yun.com/v1.1/chat", "general", nil
	default:
		return "", "", fmt.Errorf("unsupported model name: %s", modelName)
	}
}

func OpenAI2XingHuoHander(c *gin.Context, s *config.ModelDetails, oaiReq openai.OpenAIRequest) {
	appid := s.Credentials["appid"]
	apiKey := s.Credentials["api_key"]
	apiSecret := s.Credentials["api_secret"]

	defaultUrl, defaultDomain, _ := getURLAndDomain(oaiReq.Model)
	serverUrl := defaultUrl
	if s.ServerURL != "" {
		serverUrl = s.ServerURL
	}
	domain, ok := s.Credentials["domain"]
	if !ok || domain == "" {
		domain = defaultDomain
	}

	client := gosparkclient.NewSparkClientWithOptions(appid, apiKey, apiSecret, serverUrl, domain)
	xhReq := adapter.OpenAIRequestToXingHuoRequest(oaiReq)

	if oaiReq.Stream != nil && *oaiReq.Stream {
		common.SetEventStreamHeaders(c)
		client.SparkChatWithCallback(*xhReq, func(response gosparkclient.SparkAPIResponse) {
			if len(response.Payload.Choices.Text) > 0 {
				log.Println(response.Header.Sid, response.Payload.Choices.Text[0].Content)
			}

			oaiRespStream := adapter.XingHuoResponseToOpenAIStreamResponse(&response)
			respData, err := json.Marshal(&oaiRespStream)
			if err != nil {
				log.Println(err)
			} else {
				log.Println("response http data", string(respData))

				if oaiRespStream.Error != nil {

					c.JSON(http.StatusBadRequest, oaiRespStream)
				} else {
					c.Writer.WriteString("data: " + string(respData) + "\n\n")
					c.Writer.(http.Flusher).Flush()
				}

			}
		})
	} else {

	}
}
