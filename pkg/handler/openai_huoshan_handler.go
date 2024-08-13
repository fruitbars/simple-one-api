package handler

import (
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/sashabaranov/go-openai"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
	huoshanutils "github.com/volcengine/volcengine-go-sdk/service/arkruntime/utils"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
	"go.uber.org/zap"
	"io"
	"net/http"
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/mylog"
	"simple-one-api/pkg/utils"
	"strings"
	"time"
)

const DefaultHuoShanServerURL = "https://ark.cn-beijing.volces.com/api/v3"

func configureClientWithAkSk(oaiReqParam *OAIRequestParam, model string) (*arkruntime.Client, error) {
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

	client, err := configureClientWithAkSk(oaiReqParam, oaiReq.Model)
	if err != nil {
		handleErrorResponse(c, err)
		return err
	}

	huoshanReq := prepareHuoshanRequest(oaiReq, s)
	ctx := context.Background()

	//如果是bot
	if strings.HasPrefix(oaiReq.Model, "bot-") {
		return handleHuoShanBotRequest(ctx, c, huoshanReq, oaiReqParam)
	} else {
		if oaiReq.Stream {
			return handleHuoShanStream(ctx, c, client, huoshanReq, oaiReqParam)
		} else {
			return handleSingleHuoShanRequest(ctx, c, client, huoshanReq, oaiReqParam)
		}
	}

	return nil
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

func handleHuoShanStream(ctx context.Context, c *gin.Context, client *arkruntime.Client, huoshanReq model.ChatCompletionRequest, oaiReqParam *OAIRequestParam) error {
	mylog.Logger.Debug("Entering handleHuoShanStream", zap.Any("huoshanReq", huoshanReq))
	utils.SetEventStreamHeaders(c)

	stream, err := client.CreateChatCompletionStream(ctx, huoshanReq)
	if err != nil {
		mylog.Logger.Error("Failed to create chat completion stream", zap.Error(err))
		handleErrorResponse(c, err)
		return err
	}
	defer stream.Close()

	return streamHuoshanResponses(c, stream, oaiReqParam)
}

func streamHuoshanResponses(c *gin.Context, stream *huoshanutils.ChatCompletionStreamReader, oaiReqParam *OAIRequestParam) error {
	for {
		recv, err := stream.Recv()
		if err == io.EOF {
			return nil // 正常结束流
		}
		if err != nil {
			mylog.Logger.Error("Error receiving stream data", zap.Error(err))
			return err
		}

		recv.Model = oaiReqParam.ClientModel

		jsonData, err := json.Marshal(recv)
		if err != nil {
			mylog.Logger.Error("JSON marshaling error", zap.Error(err))
			return err
		}

		mylog.Logger.Info("Streaming JSON data", zap.ByteString("json_data", jsonData))
		if _, err = c.Writer.WriteString("data: " + string(jsonData) + "\n\n"); err != nil {
			mylog.Logger.Error("Write to client error", zap.Error(err))
			return err
		}

		if flusher, ok := c.Writer.(http.Flusher); ok {
			flusher.Flush()
		} else {
			mylog.Logger.Warn("Response writer does not support flush operation")
		}
	}
}

func handleSingleHuoShanRequest(ctx context.Context, c *gin.Context, client *arkruntime.Client, huoshanReq model.ChatCompletionRequest, oaiReqParam *OAIRequestParam) error {
	resp, err := client.CreateChatCompletion(ctx, huoshanReq)
	if err != nil {
		handleErrorResponse(c, err)
		return err
	}

	resp.Model = oaiReqParam.ClientModel

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
