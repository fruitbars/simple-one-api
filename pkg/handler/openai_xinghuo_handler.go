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
	modelNameLower := strings.ToLower(modelName)

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

func OpenAI2XingHuoHandler(c *gin.Context, s *config.ModelDetails, oaiReq openai.ChatCompletionRequest) error {
	appid := s.Credentials[config.KEYNAME_APPID]
	apiKey := s.Credentials[config.KEYNAME_API_KEY]
	apiSecret := s.Credentials[config.KEYNAME_API_SECRET]

	serverUrl, domain, err := getServerURLAndDomain(s, oaiReq.Model)
	if err != nil {
		return err
	}

	client := gosparkclient.NewSparkClientWithOptions(appid, apiKey, apiSecret, serverUrl, domain)
	xhReq := adapter.OpenAIRequestToXingHuoRequest(oaiReq)

	xhDataJson, _ := json.Marshal(xhReq)
	log.Println(string(xhDataJson))

	if oaiReq.Stream {
		return handleStreamMode(c, client, xhReq, oaiReq.Model)
	}

	return handleStandardMode(c, client, xhReq, oaiReq.Model)
}

func getServerURLAndDomain(s *config.ModelDetails, model string) (string, string, error) {
	defaultUrl, defaultDomain, err := getURLAndDomain(model)
	if err != nil {
		return "", "", err
	}

	serverUrl := defaultUrl
	if s.ServerURL != "" {
		serverUrl = s.ServerURL
	}

	domain := s.Credentials[config.KEYNAME_DOMAIN]
	if domain == "" {
		domain = defaultDomain
	}

	return serverUrl, domain, nil
}

func handleStreamMode(c *gin.Context, client *gosparkclient.SparkClient, xhReq *gosparkclient.SparkChatRequest, model string) error {
	log.Println("stream mode")
	utils.SetEventStreamHeaders(c)

	_, err := client.SparkChatWithCallback(*xhReq, func(response gosparkclient.SparkAPIResponse) {
		if len(response.Payload.Choices.Text) > 0 {
			log.Println(response.Header.Sid, response.Payload.Choices.Text[0].Content)
		}

		oaiRespStream := adapter.XingHuoResponseToOpenAIStreamResponse(&response)
		oaiRespStream.Model = model

		respData, err := json.Marshal(&oaiRespStream)
		if err != nil {
			log.Println("Error marshaling response:", err)
			return
		}

		log.Println("Response HTTP data:", string(respData))

		if oaiRespStream.Error != nil {
			log.Println("Error response:", *oaiRespStream.Error)
			return
		}

		_, err = c.Writer.WriteString("data: " + string(respData) + "\n\n")
		if err != nil {
			log.Println(err)
			return
		}
		c.Writer.(http.Flusher).Flush()
	})

	return err
}

func handleStandardMode(c *gin.Context, client *gosparkclient.SparkClient, xhReq *gosparkclient.SparkChatRequest, model string) error {
	xhResp, err := client.SparkChatWithCallback(*xhReq, nil)
	if err != nil {
		log.Println("Error during API call:", err)
		return err
	}

	oaiResp := adapter.XingHuoResponseToOpenAIResponse(xhResp)
	oaiResp.Model = model

	log.Println("Standard response:", *oaiResp)

	c.JSON(http.StatusOK, oaiResp)
	return nil
}
