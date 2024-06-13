package handler

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/sashabaranov/go-openai"
	"log"
	"net/http"
	"simple-one-api/pkg/adapter"
	"simple-one-api/pkg/config"
	baidu_qianfan "simple-one-api/pkg/llm/baidu-qianfan"
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
			log.Println("Error marshaling response:", err)
			return
		}

		log.Println("Response HTTP data:", string(respData))

		if qfResp.ErrorCode != 0 && oaiRespStream.Error != nil {
			log.Println("Error response:", *oaiRespStream.Error)
			c.JSON(http.StatusBadRequest, qfResp)
			return
		}

		c.Writer.WriteString("data: " + string(respData) + "\n\n")
		c.Writer.(http.Flusher).Flush()
	})

	if err != nil {
		log.Println("Error during SSE call:", err)
		return err
	}

	return nil
}

func handleQianFanStandardRequest(c *gin.Context, apiKey, secretKey, model string, qfReq *baidu_qianfan.QianFanRequest) error {
	qfResp, err := baidu_qianfan.QianFanCall(apiKey, secretKey, model, qfReq)
	if err != nil {
		log.Println("Error during API call:", err)
		return err
	}

	oaiResp := adapter.QianFanResponseToOpenAIResponse(qfResp)
	oaiResp.Model = model
	log.Println("Standard response:", oaiResp)

	c.JSON(http.StatusOK, oaiResp)
	return nil
}
