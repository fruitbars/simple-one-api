package handler

import (
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/sashabaranov/go-openai"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
	"go.uber.org/zap"
	"io"
	"net/http"
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/mylog"
	"simple-one-api/pkg/utils"
)

const DefaultHuoShanServerURL = "https://ark.cn-beijing.volces.com/api/v3"

func configureClient(s *config.ModelDetails) (*arkruntime.Client, error) {
	serverURL := s.ServerURL
	if serverURL == "" {
		serverURL = DefaultHuoShanServerURL
	}

	client := arkruntime.NewClientWithAkSk(
		s.Credentials[config.KEYNAME_ACCESS_KEY],
		s.Credentials[config.KEYNAME_SECRET_KEY],
		arkruntime.WithBaseUrl(serverURL),
		arkruntime.WithRegion("cn-beijing"),
	)
	return client, nil
}

func OpenAI2HuoShanHandler(c *gin.Context, s *config.ModelDetails, oaiReq openai.ChatCompletionRequest) error {

	client, err := configureClient(s)
	if err != nil {
		handleErrorResponse(c, err)
		return err
	}

	huoshanReq := prepareHuoshanRequest(oaiReq, s)
	ctx := context.Background()

	if oaiReq.Stream {
		return handleHuoShanStream(ctx, c, client, huoshanReq)
	} else {
		return handleSingleHuoShanRequest(ctx, c, client, huoshanReq)
	}
}

func prepareHuoshanRequest(oaiReq openai.ChatCompletionRequest, s *config.ModelDetails) model.ChatCompletionRequest {
	huoshanReq := model.ChatCompletionRequest{
		Model:    oaiReq.Model,
		Messages: []*model.ChatCompletionMessage{},
	}

	for _, msg := range oaiReq.Messages {
		huoshanMsg := &model.ChatCompletionMessage{
			Role:    msg.Role,
			Content: &model.ChatCompletionMessageContent{StringValue: volcengine.String(msg.Content)},
		}
		huoshanReq.Messages = append(huoshanReq.Messages, huoshanMsg)
	}

	return huoshanReq
}

func handleHuoShanStream(ctx context.Context, c *gin.Context, client *arkruntime.Client, huoshanReq model.ChatCompletionRequest) error {
	utils.SetEventStreamHeaders(c)
	stream, err := client.CreateChatCompletionStream(ctx, huoshanReq)
	if err != nil {
		handleErrorResponse(c, err)
		return err
	}
	defer stream.Close()

	c.Stream(func(w io.Writer) bool {
		recv, err := stream.Recv()
		if err == io.EOF {
			return false
		}
		if err != nil {
			// 假设 mylog.Logger 是一个已经配置好的 zap.Logger 实例
			mylog.Logger.Error("Stream chat error",
				zap.Error(err)) // 记录错误对象

			return false
		}

		jsonData, _ := json.Marshal(recv)

		// 假设 mylog.Logger 是一个已经配置好的 zap.Logger 实例
		mylog.Logger.Info("JSON Data",
			zap.String("json_data", string(jsonData))) // 记录 JSON 数据字符串

		_, err = c.Writer.WriteString("data: " + string(jsonData) + "\n\n")
		if err != nil {
			// 假设 mylog.Logger 是一个已经配置好的 zap.Logger 实例
			mylog.Logger.Error("An error occurred",
				zap.Error(err)) // 记录错误对象

			//return false
		}
		c.Writer.(http.Flusher).Flush()

		return true
	})

	return nil
}

func handleSingleHuoShanRequest(ctx context.Context, c *gin.Context, client *arkruntime.Client, huoshanReq model.ChatCompletionRequest) error {
	resp, err := client.CreateChatCompletion(ctx, huoshanReq)
	if err != nil {
		handleErrorResponse(c, err)
		return err
	}

	// 假设 mylog.Logger 是一个已经配置好的 zap.Logger 实例
	mylog.Logger.Info("Response received",
		zap.Any("response", resp)) // 记录响应对象

	c.JSON(http.StatusOK, resp)
	return nil
}

func handleErrorResponse(c *gin.Context, err error) {
	// 假设 mylog.Logger 是一个已经配置好的 zap.Logger 实例
	mylog.Logger.Error("An error occurred",
		zap.Error(err)) // 记录错误对象

	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}
