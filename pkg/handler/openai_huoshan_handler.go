package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sashabaranov/go-openai"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
	"io"
	"log"
	"net/http"
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/utils"
)

var defaultHuoShanServerURL = "https://ark.cn-beijing.volces.com/api/v3"

func OpenAI2HuoShanHandler(c *gin.Context, s *config.ModelDetails, oaiReq openai.ChatCompletionRequest) error {
	log.Println("OpenAI2HuoShanHandler")
	accessKey := s.Credentials[config.KEYNAME_ACCESS_KEY]
	secretKey := s.Credentials[config.KEYNAME_SECRET_KEY]
	serverURL := s.ServerURL
	if serverURL == "" {
		serverURL = defaultHuoShanServerURL
	}

	client := arkruntime.NewClientWithAkSk(
		accessKey,
		secretKey,
		arkruntime.WithBaseUrl(serverURL),
		arkruntime.WithRegion("cn-beijing"),
	)

	// 创建火山引擎请求
	huoshanReq := model.ChatCompletionRequest{
		Model:    oaiReq.Model, // 假设 s 包含火山引擎所需的模型ID
		Messages: []*model.ChatCompletionMessage{},
	}

	// 复制消息
	for _, msg := range oaiReq.Messages {
		huoshanMsg := &model.ChatCompletionMessage{
			Role:    msg.Role, // 根据需要设定角色
			Content: &model.ChatCompletionMessageContent{StringValue: volcengine.String(msg.Content)},
		}
		huoshanReq.Messages = append(huoshanReq.Messages, huoshanMsg)
	}
	ctx := context.Background()

	if oaiReq.Stream {
		utils.SetEventStreamHeaders(c)
		stream, err := client.CreateChatCompletionStream(ctx, huoshanReq)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return err
		}
		defer stream.Close()

		c.Stream(func(w io.Writer) bool {
			recv, err := stream.Recv()
			if err == io.EOF {
				return false // 结束流
			}
			if err != nil {
				fmt.Printf("Stream chat error: %v\n", err)
				return false
			}
			//recv.Model = oaiReq.Model
			jsonData, err := json.Marshal(recv)

			_, err = c.Writer.WriteString("data: " + string(jsonData) + "\n\n")
			if err != nil {
				log.Println(err)
				//return err
			}
			c.Writer.(http.Flusher).Flush()

			return true // 继续读取流
		})

		return nil
	} else {
		// 发送请求
		resp, err := client.CreateChatCompletion(ctx, huoshanReq)
		if err != nil {
			return err
		}

		// 处理响应
		c.JSON(http.StatusOK, resp) // 根据需要调整HTTP状态码和返回的数据
		return nil
	}

}
