package handler

import (
	"encoding/json"
	"fmt"
	"github.com/fruitbars/gosparkclient"
	"github.com/gin-gonic/gin"
	"github.com/sashabaranov/go-openai"
	"log"
	"net/http"
	"simple-one-api/pkg/adapter"
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/utils"
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

func OpenAI2XingHuoHander(c *gin.Context, s *config.ModelDetails, oaiReq openai.ChatCompletionRequest) error {
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

	xhDataJson, _ := json.Marshal(xhReq)
	log.Println(string(xhDataJson))

	if oaiReq.Stream {
		log.Println("stream mode")
		utils.SetEventStreamHeaders(c)
		_, err := client.SparkChatWithCallback(*xhReq, func(response gosparkclient.SparkAPIResponse) {
			if len(response.Payload.Choices.Text) > 0 {
				log.Println(response.Header.Sid, response.Payload.Choices.Text[0].Content)
			}

			oaiRespStream := adapter.XingHuoResponseToOpenAIStreamResponse(&response)
			oaiRespStream.Model = oaiReq.Model
			respData, err := json.Marshal(&oaiRespStream)
			if err != nil {
				log.Println(err)
				return
			} else {
				log.Println("response http data", string(respData))

				if oaiRespStream.Error != nil {
					log.Println(*oaiRespStream.Error)
					return
				} else {
					c.Writer.WriteString("data: " + string(respData) + "\n\n")
					c.Writer.(http.Flusher).Flush()
				}
			}
		})

		return err

	} else {
		//client := gosparkclient.NewSparkClientWithOptions(appid, apiKey, apiSecret, serverUrl, domain)
		//xhReq := adapter.OpenAIRequestToXingHuoRequest(oaiReq)
		xhresp, err := client.SparkChatWithCallback(*xhReq, nil)
		if err != nil {
			log.Println(err)
			return err
		}

		myresp := adapter.XingHuoResponseToOpenAIResponse(xhresp)
		myresp.Model = oaiReq.Model

		log.Println("响应：", *myresp)

		c.JSON(http.StatusOK, myresp)
	}

	return nil
}
