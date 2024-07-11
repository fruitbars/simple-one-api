package handler

import (
	"bytes"
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"io"
	"net/http"
	"simple-one-api/pkg/adapter"
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/mycommon"
	"simple-one-api/pkg/mylimiter"
	"simple-one-api/pkg/mylog"
	"simple-one-api/pkg/utils"
	"strings"
	"time"
)

var defaultReqTimeout = 10

type OAIRequestParam struct {
	chatCompletionReq *openai.ChatCompletionRequest
	modelDetails      *config.ModelDetails
	creds             map[string]interface{}
	httpTransport     *http.Transport
}

// serviceHandlerMap maps service names to their corresponding handler functions
var serviceHandlerMap = map[string]func(*gin.Context, *OAIRequestParam) error{
	"qianfan":  OpenAI2QianFanHandler,
	"hunyuan":  OpenAI2HunYuanHandler,
	"xinghuo":  OpenAI2XingHuoHandler,
	"openai":   OpenAI2OpenAIHandler,
	"azure":    OpenAI2AzureOpenAIHandler,
	"deepseek": OpenAI2OpenAIHandler,
	"zhipu":    OpenAI2OpenAIHandler,
	"minimax":  OpenAI2MinimaxHandler,
	"cozecn":   OpenAI2CozecnHandler,
	"cozecom":  OpenAI2CozecnHandler,
	"coze":     OpenAI2CozecnHandler,
	"huoshan":  OpenAI2HuoShanHandler,
	"ollama":   OpenAI2OllamaHandler,
	"groq":     OpenAI2GroqOpenAIHandler,
	"gemini":   OpenAI2GeminiHandler,
	"aliyun":   OpenAI2OpenAIHandler,
	"vertexai": OpenAI2VertexAIHandler,
	"claude":   OpenAI2ClaudeHandler,
}

func LogRequestBody(c *gin.Context) {
	// 读取请求消息体
	body, err := c.GetRawData()
	if err != nil {
		mylog.Logger.Error(err.Error())
		return
	}

	// 将消息体转换为字符串并记录
	//requestBody := string(body)
	//mylog.Logger.Debug("Request body", zap.String("body", requestBody))

	// 重置请求体，以便后续处理程序可以读取它
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
}

func LogRequestDetails(c *gin.Context) {
	// 使用 zap 的字段记录功能来记录请求细节
	mylog.Logger.Debug("HTTP request details",
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path),
		zap.Any("parameters", c.Request.URL.Query()),
		zap.Any("headers", c.Request.Header),
	)

	LogRequestBody(c)
}

// OpenAIHandler handles POST requests on /v1/chat/completions path
func OpenAIHandler(c *gin.Context) {
	if !validateRequestMethod(c, "POST") {
		return
	}
	LogRequestDetails(c)

	if err := validateAPIKey(c); err != nil {
		mylog.Logger.Error(err.Error())
		sendErrorResponse(c, http.StatusUnauthorized, err.Error())
		return
	}

	var oaiReq openai.ChatCompletionRequest
	if err := c.ShouldBindJSON(&oaiReq); err != nil {
		mylog.Logger.Error(err.Error())
		sendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	HandleOpenAIRequest(c, &oaiReq)

	return
}

func HandleOpenAIRequest(c *gin.Context, oaiReq *openai.ChatCompletionRequest) {

	clientModel := oaiReq.Model

	//全局模型重定向名称
	gRedirectModel := config.GetGlobalModelRedirect(clientModel)

	oaiReq.Model = gRedirectModel

	s, serviceModelName, err := getModelDetails(oaiReq)
	if err != nil {
		mylog.Logger.Error(err.Error())
		sendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	//模型重定向名称
	mrModel := config.GetModelRedirect(s, serviceModelName)
	mpModel := config.GetModelMapping(s, mrModel)

	oaiReq.Model = mpModel

	mylog.Logger.Info("Service details",
		zap.String("service_name", s.ServiceName),
		zap.String("client_model", clientModel),
		zap.String("g_redirect_model", gRedirectModel),
		zap.String("service_model_name", serviceModelName),
		zap.String("redirect_model", mrModel),
		zap.String("map_model", mpModel),
		zap.String("last_model", oaiReq.Model))

	isSupportMC := config.IsSupportMultiContent(oaiReq.Model)
	if len(oaiReq.Messages) > 0 && len(oaiReq.Messages[0].MultiContent) > 0 {
		if !isSupportMC {
			mylog.Logger.Warn("model support vision", zap.Bool("isSupportMC", isSupportMC))
			//convert message
			adapter.OpenAIMultiContentRequestToOpenAIContentRequest(oaiReq)
			mylog.Logger.Info("", zap.Any("oaiReq", oaiReq))
		}
	}

	creds, credsID := mycommon.GetACredentials(s, oaiReq.Model)

	var limiter *mylimiter.Limiter
	lt, ln, timeout := mycommon.GetServiceModelDetailsLimit(s)
	if lt != "" && ln > 0 {
		limiter = mylimiter.GetLimiter(s.ServiceID, lt, ln)
	} else {
		lt, ln, timeout = mycommon.GetCredentialLimit(creds)
		if lt != "" && ln > 0 {
			limiter = mylimiter.GetLimiter(credsID, lt, ln)
		}
	}

	oaiReqParam := &OAIRequestParam{
		chatCompletionReq: oaiReq,
		modelDetails:      s,
		creds:             creds,
	}

	if limiter != nil {
		if timeout <= 0 {
			timeout = defaultReqTimeout
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
		defer cancel()

		startWaitTime := time.Now()

		mylog.Logger.Info("Rate limits and timeout configuration",
			zap.String("limit type:", lt),
			zap.Float64("limit num:", ln),
			zap.Int("timeout", timeout))

		if lt == "qps" {
			err = limiter.Wait(ctx)
			elapsed := time.Since(startWaitTime)

			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) {
					// Log a message if the request could not obtain a token within the specified timeout period.
					// 假设 logger 是一个已经配置好的 zap.Logger 实例
					mylog.Logger.Error("Failed to obtain token within the specified time",
						zap.Error(err),                   // 记录错误对象
						zap.Int("timeout", timeout),      // 假设 timeout 是 time.Duration 类型
						zap.Duration("elapsed", elapsed)) // 假设 elapsed 是 time.Duration 类型

				} else if errors.Is(err, context.Canceled) {
					// Log a message if the operation was canceled.
					mylog.Logger.Error("Operation canceled %v, actual waiting time: %v", zap.Error(err), zap.Duration("elapsed", elapsed))
				} else {
					// Log a message for any other unknown errors that occurred while waiting for a token.
					mylog.Logger.Error("Unknown error occurred while waiting for a token: ", zap.Error(err), zap.Duration("elapsed", elapsed))
				}

				//waitDuration := time.Since(startWaitTime)
				mylog.Logger.Debug("waited for: ", zap.Duration("elapsed", elapsed))
				sendErrorResponse(c, http.StatusTooManyRequests, "Request rate limit exceeded")
				return
			}
			// 假设 logger 是一个已经配置好的 zap.Logger 实例
			mylog.Logger.Debug("Wait duration",
				zap.Duration("waited_for", time.Since(startWaitTime)))

		} else if lt == "concurrency" {

			err := limiter.Acquire(ctx)
			if err != nil {
				mylog.Logger.Error(err.Error())
			}
			defer limiter.Release()

			mylog.Logger.Debug("Concurrency wait time",
				zap.Duration("waited_for", time.Since(startWaitTime)))
		}

	}

	if config.IsProxyEnabled(s) {
		proxyType, proxyAddr, transport, err := config.GetConfProxyTransport()
		if err != nil {
			mylog.Logger.Error("GetConfProxyTransport", zap.Error(err))
		} else {
			mylog.Logger.Debug("GetConfProxyTransport", zap.String("proxyType", proxyType), zap.String("proxyAddr", proxyAddr))
			oaiReqParam.httpTransport = transport
		}
	}

	//mylog.Logger.Debug("oaiReq", zap.Any("oaiReq", oaiReq))
	oaiReq.Messages = mycommon.NormalizeMessages(oaiReq.Messages)

	if err := dispatchToServiceHandler(c, oaiReqParam); err != nil {
		//mylog.Logger.Error(err.Error())
		sendErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	if oaiReq.Stream {
		utils.SendOpenAIStreamEOFData(c)
	}
}

// dispatchToServiceHandler dispatches the request to the appropriate service handler based on the service name
func dispatchToServiceHandler(c *gin.Context, oaiReqParam *OAIRequestParam) error {
	s := oaiReqParam.modelDetails
	serviceName := strings.ToLower(s.ServiceName)
	if handler, ok := serviceHandlerMap[serviceName]; ok {
		return handler(c, oaiReqParam)
	}
	return errors.New("service handler not found")
}

func validateRequestMethod(c *gin.Context, method string) bool {
	if c.Request.Method != method {
		sendErrorResponse(c, http.StatusMethodNotAllowed, "Only "+method+" method is accepted")
		return false
	}
	return true
}

func validateAPIKey(c *gin.Context) error {
	if config.APIKey == "" {
		return nil
	}

	apikey, err := utils.GetAPIKeyFromHeader(c)
	if err != nil || config.APIKey != apikey {
		return errors.New("invalid authorization")
	}
	return nil
}

func getModelDetails(oaiReq *openai.ChatCompletionRequest) (*config.ModelDetails, string, error) {
	if oaiReq.Model == config.KEYNAME_RANDOM {
		return config.GetRandomEnabledModelDetailsV1()
	}
	s, err := config.GetModelService(oaiReq.Model)
	if err != nil {
		return nil, "", err
	}

	return s, oaiReq.Model, err
}

func sendErrorResponse(c *gin.Context, code int, msg string) {
	c.JSON(code, gin.H{"error": msg})
}
