package handler

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"simple-one-api/pkg/adapter"
	"simple-one-api/pkg/config"
	baidu_qianfan "simple-one-api/pkg/llm/baidu-qianfan"
	"simple-one-api/pkg/openai"
	"simple-one-api/pkg/utils"
)

func OpenAI2QianFanHander(c *gin.Context, s *config.ModelDetails, oaiReq openai.OpenAIRequest) error {
	if oaiReq.Stream != nil && *oaiReq.Stream {
		apiKey := s.Credentials["api_key"]
		secretKey := s.Credentials["secret_key"]
		qfReq := adapter.OpenAIRequestToQianFanRequest(oaiReq)

		utils.SetEventStreamHeaders(c)

		// 创建 HTTP 客户端请求并处理 SSE
		err := baidu_qianfan.QianFanCallSSE(apiKey, secretKey, oaiReq.Model, qfReq, func(qfResp *baidu_qianfan.QianFanResponse) {
			// 将数据转发给客户端
			oaiRespStream := adapter.QianFanResponseToOpenAIStreamResponse(qfResp)
			oaiRespStream.Model = oaiReq.Model
			respData, err := json.Marshal(&oaiRespStream)
			if err != nil {
				log.Println(err)
			} else {
				log.Println("response http data", string(respData))

				if qfResp.ErrorCode != 0 && oaiRespStream.Error != nil {
					log.Println(*oaiRespStream.Error)
					c.JSON(http.StatusBadRequest, *qfResp)
					return
				}
				c.Writer.WriteString("data: " + string(respData) + "\n\n")
				c.Writer.(http.Flusher).Flush()
			}

		})

		if err != nil {
			log.Println(err)
			return err
		}

	} else {
		apiKey := s.Credentials["api_key"]
		secretKey := s.Credentials["secret_key"]
		qfReq := adapter.OpenAIRequestToQianFanRequest(oaiReq)

		// 确保传入的 QianFanRequest 对象 qfReq 是正确初始化并准备好的
		qfResp, err := baidu_qianfan.QianFanCall(apiKey, secretKey, oaiReq.Model, qfReq)

		if err != nil {
			log.Println(err)
			return err
		} else {
			//var oaiResp *openai.OpenAIResponse
			oaiResp := adapter.QianFanResponseToOpenAIResponse(qfResp)
			oaiResp.Model = oaiReq.Model
			log.Println(oaiResp)
			//oaiResp.Model = oaiReq.Model
			// 设置响应的内容类型并发送JSON响应
			c.JSON(http.StatusOK, oaiResp)
		}
	}

	return nil
}
