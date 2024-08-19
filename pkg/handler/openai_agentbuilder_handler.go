package handler

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"log"
	"net/http"
	"simple-one-api/pkg/adapter/baidu_agentbuilder_adapter"
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/llm/devplatform/baidu_agentbuilder"
	"simple-one-api/pkg/mycommon"
	"simple-one-api/pkg/mylog"
	"simple-one-api/pkg/utils"
)

func OpenAI2AgentBuilderHandler(c *gin.Context, oaiReqParam *OAIRequestParam) error {
	oaiReq := oaiReqParam.chatCompletionReq
	//s := oaiReqParam.modelDetails
	credentials := oaiReqParam.creds
	secretKey, _ := utils.GetStringFromMap(credentials, config.KEYNAME_SECRET_KEY)

	query := mycommon.GetLastestMessage(oaiReq.Messages)

	if oaiReq.Stream {
		cb := func(data string) {
			log.Println(data)
			var resp baidu_agentbuilder.ConversationResponse
			err := json.Unmarshal([]byte(data), &resp)
			if err != nil {
				mylog.Logger.Error("An error occurred",
					zap.Error(err)) // 记录错误对象
				return
			}

			oaiRespStream := baidu_agentbuilder_adapter.AgentBuilderResponseToOpenAIStreamResponse(&resp)

			oaiRespStream.Model = oaiReq.Model

			respData, err := json.Marshal(&oaiRespStream)
			if err != nil {
				mylog.Logger.Error("Error marshaling response:", zap.Error(err))
				return
			}

			// 假设 mylog.Logger 是一个已经配置好的 zap.Logger 实例
			mylog.Logger.Info("Response HTTP data",
				zap.String("data", string(respData))) // 记录响应数据

			_, err = c.Writer.WriteString("data: " + string(respData) + "\n\n")
			if err != nil {
				// 假设 mylog.Logger 是一个已经配置好的 zap.Logger 实例
				mylog.Logger.Error("An error occurred",
					zap.Error(err)) // 记录错误对象

				return
			}
			c.Writer.(http.Flusher).Flush()
		}

		err := baidu_agentbuilder.Conversation(oaiReq.Model, secretKey, query, cb)
		if err != nil {
			// 假设 mylog.Logger 是一个已经配置好的 zap.Logger 实例
			mylog.Logger.Error("OpenAI2AgentBuilderHandler|baidu_agentbuilder.Conversation",
				zap.Error(err)) // 记录错误对象

			return err
		}

	} else {
		abResp, err := baidu_agentbuilder.GetAnswer(oaiReq.Model, secretKey, query)
		if err != nil {

			return err
		}

		oaiResp := baidu_agentbuilder_adapter.AgentBuilderResponseToOpenAIResponse(abResp)

		oaiResp.Model = oaiReq.Model

		// 假设 mylog.Logger 是一个已经配置好的 zap.Logger 实例
		mylog.Logger.Info("Standard response",
			zap.Any("response", *oaiResp)) // 记录响应对象

		c.JSON(http.StatusOK, oaiResp)

	}

	return nil
}
