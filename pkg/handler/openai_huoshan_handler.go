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
	"time"
)

const DefaultHuoShanServerURL = "https://ark.cn-beijing.volces.com/api/v3"

func configureClient(oaiReqParam *OAIRequestParam, model string) (*arkruntime.Client, error) {
	s := oaiReqParam.modelDetails
	credentials := oaiReqParam.creds

	serverURL := s.ServerURL
	if serverURL == "" {
		serverURL = DefaultHuoShanServerURL
	}

	accessKey, _ := utils.GetStringFromMap(credentials, config.KEYNAME_ACCESS_KEY)
	secretKey, _ := utils.GetStringFromMap(credentials, config.KEYNAME_SECRET_KEY)

	// 创建自定义 HTTP client 使用上述 transport
	httpHSClient := &http.Client{
		//Transport: transport,
		Timeout: 30 * time.Second,
	}

	if oaiReqParam.httpTransport != nil {
		httpHSClient.Transport = oaiReqParam.httpTransport
	}

	// 定义一个 configOption 来设置自定义的 HTTP client
	withCustomHTTPClient := func(config *arkruntime.ClientConfig) {
		config.HTTPClient = httpHSClient
	}

	// 使用 NewClientWithAkSk 创建 Client，并应用自定义的 HTTP client 和其他配置
	client := arkruntime.NewClientWithAkSk(
		accessKey,
		secretKey,
		arkruntime.WithBaseUrl(serverURL),
		arkruntime.WithRegion("cn-beijing"),
		withCustomHTTPClient, // 应用自定义 HTTP client 的配置
	)

	return client, nil
}

func OpenAI2HuoShanHandler(c *gin.Context, oaiReqParam *OAIRequestParam) error {
	oaiReq := oaiReqParam.chatCompletionReq
	s := oaiReqParam.modelDetails
	//credentials := oaiReqParam.creds

	client, err := configureClient(oaiReqParam, oaiReq.Model)
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

func prepareHuoshanRequest(oaiReq *openai.ChatCompletionRequest, s *config.ModelDetails) model.ChatCompletionRequest {
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
