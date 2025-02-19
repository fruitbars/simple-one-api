package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"simple-one-api/pkg/adapter"
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/mycommon"
	"simple-one-api/pkg/mylog"
	"simple-one-api/pkg/utils"
	"strings"
)

// validateAndFormatURL checks if the given URL matches the specified formats and returns the formatted URL
func validateAndFormatURL(rawurl string) (string, bool) {
	parsedURL, err := url.Parse(rawurl)
	if err != nil {
		return "", false
	}

	re := regexp.MustCompile(`/v([1-9]|[1-4][0-9]|50)(/chat/completions|/)?$`)
	if re.MatchString(parsedURL.Path) {
		if submatch := re.FindStringSubmatch(parsedURL.Path); submatch[2] == "/chat/completions" {
			formattedURL := fmt.Sprintf("%s://%s%s", parsedURL.Scheme, parsedURL.Host, parsedURL.Path[:len(parsedURL.Path)-len("/chat/completions")])
			return formattedURL, true
		}
		return rawurl, true
	}
	return rawurl, false
}

// getDefaultServerURL returns the default server URL based on the model prefix
func getDefaultServerURL(model string) string {
	model = strings.ToLower(model)
	switch {
	case strings.HasPrefix(model, "glm-"):
		return "https://open.bigmodel.cn/api/paas/v4/chat/completions"
	case strings.HasPrefix(model, "deepseek-"):
		return "https://api.deepseek.com/v1"
	case strings.HasPrefix(model, "yi-"):
		return "https://api.lingyiwanwu.com/v1/chat/completions"
	case strings.HasPrefix(model, "gpt-"):
		return "https://api.openai.com/v1/chat/completions"
	default:
		return ""
	}
}

// getConfig generates the OpenAI client configuration based on model details and request
func getConfig(s *config.ModelDetails, oaiReqParam *OAIRequestParam) (openai.ClientConfig, error) {
	req := oaiReqParam.chatCompletionReq
	credentials := oaiReqParam.creds
	apiKey, _ := utils.GetStringFromMap(credentials, config.KEYNAME_API_KEY)
	conf := openai.DefaultConfig(apiKey)

	serverURL := s.ServerURL
	if serverURL == "" {
		serverURL = getDefaultServerURL(req.Model)
		mylog.Logger.Info("Using default server URL",
			zap.String("server_url", serverURL)) // 记录默认服务器 URL
	}

	if serverURL != "" {
		formattedURL, ok := validateAndFormatURL(serverURL)

		conf.BaseURL = formattedURL
		if ok {
			mylog.Logger.Info("Formatted server URL is valid",
				zap.String("formatted_url", formattedURL))
		} else {
			mylog.Logger.Warn("Formatted server URL is invalid",
				zap.String("formatted_url", formattedURL))
		}
	} else {
		return conf, errors.New("server URL is empty")
	}

	return conf, nil
}

// handleOpenAIRequest handles OpenAI requests, supporting both streaming and non-streaming modes
func handleOpenAIOpenAIRequest(conf openai.ClientConfig, c *gin.Context, req *openai.ChatCompletionRequest, clientModel string) error {

	logOpenAIChatCompletionRequest(req)

	openaiClient := openai.NewClientWithConfig(conf)

	ctx := context.Background()

	if req.Stream {
		return handleOpenAIOpenAIStreamRequest(c, openaiClient, ctx, req, clientModel)
	}

	return handleOpenAIStandardRequest(c, openaiClient, ctx, req, clientModel)
}

// handleStreamRequest handles streaming OpenAI requests
func handleOpenAIOpenAIStreamRequest(c *gin.Context, client *openai.Client, ctx context.Context, req *openai.ChatCompletionRequest, clientModel string) error {
	utils.SetEventStreamHeaders(c)
	stream, err := client.CreateChatCompletionStream(ctx, *req)
	if err != nil {
		mylog.Logger.Error("An error occurred",
			zap.Error(err))
		return fmt.Errorf("ChatCompletionStream error: %w", err)
	}
	defer stream.Close()

	backIdStr := uuid.New().String()
	for {
		response, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			mylog.Logger.Info(err.Error())
			return nil
		} else if err != nil {
			mylog.Logger.Error("An error occurred",
				zap.Error(err))
			return err
		}

		mylog.Logger.Debug("CheckOpenAIStreamRespone1",
			zap.Any("response", response))

		if response.ID == "" {
			response.ID = backIdStr
		}

		adapter.CheckOpenAIStreamRespone(&response)

		response.Model = clientModel
		respData, err := json.Marshal(&response)
		if err != nil {
			mylog.Logger.Error("An error occurred",
				zap.Error(err))
			return err
		}

		mylog.Logger.Debug("Response data",
			zap.String("resp_data", string(respData))) // 记录响应数据

		_, err = c.Writer.WriteString("data: " + string(respData) + "\n\n")
		if err != nil {
			mylog.Logger.Error("An error occurred",
				zap.Error(err))
			return err
		}
		c.Writer.(http.Flusher).Flush()
	}
}

// handleStandardRequest handles non-streaming OpenAI requests
func handleOpenAIStandardRequest(c *gin.Context, client *openai.Client, ctx context.Context, req *openai.ChatCompletionRequest, clientModel string) error {
	resp, err := client.CreateChatCompletion(ctx, *req)
	if err != nil {
		mylog.Logger.Error("An error occurred",
			zap.Any("req", req),
			zap.Error(err))
		return err
	}

	myResp := adapter.OpenAIResponseToOpenAIResponse(&resp)
	myResp.Model = clientModel

	respJsonStr, err := json.Marshal(*myResp)
	if err != nil {
		mylog.Logger.Error("An error occurred",
			zap.Error(err)) // 记录错误对象
	}

	mylog.Logger.Info("Response JSON String",
		zap.String("resp_json_str", string(respJsonStr))) // 记录响应 JSON 字符串

	c.JSON(http.StatusOK, myResp)
	return nil
}

// OpenAI2OpenAIHandler handles OpenAI to OpenAI requests
func OpenAI2OpenAIHandler(c *gin.Context, oaiReqParam *OAIRequestParam) error {
	//oaiReq := oaiReqParam.chatCompletionReq
	s := oaiReqParam.modelDetails
	//credentials := oaiReqParam.creds
	conf, err := getConfig(s, oaiReqParam)
	if err != nil {
		return err
	}

	if strings.HasPrefix(s.ServerURL, "https://api.groq.com/openai/v1") {
		adjustGroqReq(oaiReqParam.chatCompletionReq)
	} else if strings.HasPrefix(s.ServerURL, "https://open.bigmodel.cn") {
		mycommon.AdjustOpenAIRequestParams(oaiReqParam.chatCompletionReq)

		if strings.Contains(oaiReqParam.chatCompletionReq.Model, "glm-4v") {
			AdjustChatCompletionRequestForZhiPu(oaiReqParam.chatCompletionReq)
		}
	}

	defaultTransport := http.DefaultTransport
	if oaiReqParam.httpTransport != nil {
		defaultTransport = oaiReqParam.httpTransport
	}

	scTransport := &utils.SimpleCustomTransport{
		Transport: defaultTransport,
	}
	conf.HTTPClient = &http.Client{
		Transport: scTransport,
	}

	mylog.Logger.Debug("OpenAI2OpenAIHandler", zap.Any("req", oaiReqParam.chatCompletionReq), zap.Any("scTransport", scTransport.Transport))

	clientModel := oaiReqParam.ClientModel
	return handleOpenAIOpenAIRequest(conf, c, oaiReqParam.chatCompletionReq, clientModel)
}
