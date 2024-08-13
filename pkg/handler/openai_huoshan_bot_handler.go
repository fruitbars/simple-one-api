package handler

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/sashabaranov/go-openai"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
	"go.uber.org/zap"
	"io"
	"net/http"
	"simple-one-api/pkg/adapter"
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/mylog"
	myopenai "simple-one-api/pkg/openai"
	"simple-one-api/pkg/utils"
	"time"
)

func configureClientWithApiKey(oaiReqParam *OAIRequestParam, model string) (*arkruntime.Client, error) {
	s := oaiReqParam.modelDetails
	credentials := oaiReqParam.creds

	serverURL := s.ServerURL
	if serverURL == "" {
		serverURL = DefaultHuoShanServerURL
	}

	apiKey, _ := utils.GetStringFromMap(credentials, config.KEYNAME_API_KEY)

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
	client := arkruntime.NewClientWithApiKey(
		apiKey,
		arkruntime.WithBaseUrl(serverURL),
		arkruntime.WithRegion("cn-beijing"),
		withCustomHTTPClient, // 应用自定义 HTTP client 的配置
	)

	return client, nil
}

func prepareHuoshanBotRequest(oaiReq *openai.ChatCompletionRequest) model.BotChatCompletionRequest {
	huoshanReq := model.BotChatCompletionRequest{
		BotId:    oaiReq.Model,
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

/*
func handleHuoShanBotRequest(ctx context.Context, c *gin.Context, huoshanReq model.ChatCompletionRequest, oaiReqParam *OAIRequestParam) error {

	oaiReq := oaiReqParam.chatCompletionReq
	s := oaiReqParam.modelDetails
	client, _ := configureClientWithApiKey(oaiReqParam, oaiReq.Model)

	clientModel := oaiReqParam.ClientModel

	botReq := prepareHuoshanBotRequest(oaiReq, s)
	mylog.Logger.Info("handleHuoShanBotRequest", zap.Any("botReq", botReq))
	if oaiReq.Stream {
		stream, err := client.CreateBotChatCompletionStream(ctx, botReq)
		if err != nil {
			mylog.Logger.Error("handleHuoShanBotRequest", zap.Error(err))
			return nil
		}
		defer stream.Close()

		for {
			recv, err := stream.Recv()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				mylog.Logger.Error("handleHuoShanBotRequest", zap.Error(err))
				return nil
			}

			oaiRespStream := adapter.HuoShanBotResponseToOpenAIStreamResponse(&recv)

			oaiRespStream.Model = clientModel

			respData, err := json.Marshal(&oaiRespStream)
			if err != nil {
				mylog.Logger.Error("Error marshaling response",
					zap.Error(err)) // 记录错误对象

				return err
			}

			mylog.Logger.Info("Response HTTP data",
				zap.String("http_data", string(respData))) // 记录 HTTP 响应数据

			if oaiRespStream.Error != nil {
				mylog.Logger.Error("Error response",
					zap.Any("error", *oaiRespStream.Error)) // 记录错误对象

				c.JSON(http.StatusBadRequest, recv)
				return errors.New("error")
			}

			c.Writer.WriteString("data: " + string(respData) + "\n\n")
			c.Writer.(http.Flusher).Flush()
		}
	} else {
		resp, err := client.CreateBotChatCompletion(ctx, botReq)
		if err != nil {
			mylog.Logger.Error("handleHuoShanBotRequest", zap.Error(err))
			return nil
		}
		mylog.Logger.Info("", zap.Any("resp", resp))

		myresp := adapter.HuoShanBotResponseToOpenAIResponse(&resp)

		myresp.Model = clientModel

		respData, _ := json.Marshal(*myresp)
		mylog.Logger.Info(string(respData))

		c.JSON(http.StatusOK, myresp)

		return nil
	}
}
*/

func handleHuoShanBotRequest(ctx context.Context, c *gin.Context, huoshanReq model.ChatCompletionRequest, oaiReqParam *OAIRequestParam) error {
	oaiReq := oaiReqParam.chatCompletionReq
	client, err := configureClientWithApiKey(oaiReqParam, oaiReq.Model)
	if err != nil {
		return err
	}

	botReq := prepareHuoshanBotRequest(oaiReq)
	mylog.Logger.Info("handleHuoShanBotRequest", zap.Any("botReq", botReq))

	if oaiReq.Stream {
		return handleHuoshanBotStreamResponse(ctx, c, client, botReq, oaiReqParam.ClientModel)
	} else {
		return handleHuoshanBotNonStreamResponse(ctx, c, client, botReq, oaiReqParam.ClientModel)
	}
}

func handleHuoshanBotStreamResponse(ctx context.Context, c *gin.Context, client *arkruntime.Client, botReq model.BotChatCompletionRequest, clientModel string) error {
	stream, err := client.CreateBotChatCompletionStream(ctx, botReq)
	if err != nil {
		mylog.Logger.Error("Failed to create stream", zap.Error(err))
		return err
	}
	defer stream.Close()

	for {
		recv, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			mylog.Logger.Error("Stream receive error", zap.Error(err))
			return err
		}

		oaiRespStream := adapter.HuoShanBotResponseToOpenAIStreamResponse(&recv)
		oaiRespStream.Model = clientModel

		if err := writeHuoshanBotStreamResponse(c, oaiRespStream); err != nil {
			return err
		}
	}
}

func handleHuoshanBotNonStreamResponse(ctx context.Context, c *gin.Context, client *arkruntime.Client, botReq model.BotChatCompletionRequest, clientModel string) error {
	resp, err := client.CreateBotChatCompletion(ctx, botReq)
	if err != nil {
		mylog.Logger.Error("Failed to create bot chat completion", zap.Error(err))
		return err
	}
	mylog.Logger.Info("Received response", zap.Any("resp", resp))

	myresp := adapter.HuoShanBotResponseToOpenAIResponse(&resp)
	myresp.Model = clientModel

	c.JSON(http.StatusOK, myresp)
	return nil
}

func writeHuoshanBotStreamResponse(c *gin.Context, oaiRespStream *myopenai.OpenAIStreamResponse) error {
	respData, err := json.Marshal(oaiRespStream)
	if err != nil {
		mylog.Logger.Error("Error marshaling response", zap.Error(err))
		return err
	}

	mylog.Logger.Info("Response HTTP data", zap.String("http_data", string(respData)))

	if oaiRespStream.Error != nil {
		mylog.Logger.Error("Error response", zap.Any("error", *oaiRespStream.Error))
		c.JSON(http.StatusBadRequest, oaiRespStream.Error)
		return errors.New("error in response")
	}

	c.Writer.WriteString("data: " + string(respData) + "\n\n")
	c.Writer.(http.Flusher).Flush()

	return nil
}
