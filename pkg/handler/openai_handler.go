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
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/mylog"
	"simple-one-api/pkg/utils"
	"strings"
	"time"
)

var defaultReqTimeout = 10

// serviceHandlerMap maps service names to their corresponding handler functions
var serviceHandlerMap = map[string]func(*gin.Context, *config.ModelDetails, openai.ChatCompletionRequest) error{
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
}

func LogRequestBody(c *gin.Context) {
	// 读取请求消息体
	body, err := c.GetRawData()
	if err != nil {
		mylog.Logger.Error(err.Error())
		return
	}

	// 将消息体转换为字符串并记录
	requestBody := string(body)
	mylog.Logger.Debug("Request body", zap.String("body", requestBody))

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

	var oaiReq openai.ChatCompletionRequest
	if err := c.ShouldBindJSON(&oaiReq); err != nil {
		mylog.Logger.Error(err.Error())
		sendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := validateAPIKey(c); err != nil {
		mylog.Logger.Error(err.Error())
		sendErrorResponse(c, http.StatusUnauthorized, err.Error())
		return
	}

	handleOpenAIRequest(c, oaiReq)
}

func handleOpenAIRequest(c *gin.Context, oaiReq openai.ChatCompletionRequest) {

	s, modelName, err := getModelDetails(oaiReq)
	if err != nil {
		mylog.Logger.Error(err.Error())
		sendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	timeout := s.Timeout
	if timeout <= 0 {
		timeout = defaultReqTimeout // default timeout if not specified
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	startWaitTime := time.Now()

	if s.Limiter != nil {
		// 假设 logger 是一个已经配置好的 zap.Logger 实例
		mylog.Logger.Info("Rate limits and timeout configuration",
			zap.Int("qps_limit", s.Limit.QPS), // 假设 QPS 是 float64 类型
			zap.Int("qpm_limit", s.Limit.QPM), // 假设 QPM 也是 float64 类型
			zap.Int("timeout", timeout))       // 假设 timeout 是 time.Duration 类型

		err = s.Limiter.Wait(ctx)
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
			zap.Duration("waited_for", time.Since(startWaitTime))) // 记录从 startWaitTime 到现在的时间差

	} else if s.ConcurrencyLimiter != nil {
		// 假设 logger 是一个已经配置好的 zap.Logger 实例
		mylog.Logger.Debug("Concurrency limit set",
			zap.Int("concurrency_limit", s.Limit.Concurrency)) // 假设 s.Limit.Concurrency 是 int 类型

		select {
		case <-s.ConcurrencyLimiter: // attempt to get a token from concurrency limiter
			//log.Println("token acquired")
			defer func() {
				waitDuration := time.Since(startWaitTime)
				mylog.Logger.Info("Timeout after waiting",
					zap.Duration("wait_duration", waitDuration))
				s.ConcurrencyLimiter <- struct{}{} // release token upon request completion
			}()
		case <-ctx.Done(): // handle timeout
			waitDuration := time.Since(startWaitTime)
			// 假设 logger 是一个已经配置好的 zap.Logger 实例
			mylog.Logger.Info("Timeout after waiting",
				zap.Duration("wait_duration", waitDuration)) // waitDuration 应该是一个 time.Duration 类型

			sendErrorResponse(c, http.StatusBadRequest, "Concurrency limit exceeded")
			return
		}
		// 假设 logger 是一个已经配置好的 zap.Logger 实例
		mylog.Logger.Debug("Concurrency wait time",
			zap.Duration("waited_for", time.Since(startWaitTime))) // 记录从 startWaitTime 到现在所经历的时间

	}

	clientModel := oaiReq.Model
	if clientModel == "random" {
		oaiReq.Model = config.GetModelRedirect(s, clientModel)
	}

	//兼容之前的版本
	if oaiReq.Model == clientModel {
		oaiReq.Model = config.GetModelMapping(s, modelName)
	}

	// 假设 logger 是一个已经配置好的 zap.Logger 实例
	mylog.Logger.Debug("Service details",
		zap.String("service_name", s.ServiceName), // 假设 s.ServiceName 是 string 类型
		zap.String("client_model", clientModel),   // 假设 clientModel 是 string 类型
		zap.String("real_model_name", modelName),  // 假设 modelName 是 string 类型
		zap.String("last_model", oaiReq.Model))    // 假设 oaiReq.Model 是 string 类型

	if err := dispatchToServiceHandler(c, s, oaiReq); err != nil {
		//mylog.Logger.Error(err.Error())
		sendErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	if oaiReq.Stream {
		utils.SendOpenAIStreamEOFData(c)
	}
}

// dispatchToServiceHandler dispatches the request to the appropriate service handler based on the service name
func dispatchToServiceHandler(c *gin.Context, s *config.ModelDetails, oaiReq openai.ChatCompletionRequest) error {
	serviceName := strings.ToLower(s.ServiceName)
	if handler, ok := serviceHandlerMap[serviceName]; ok {
		return handler(c, s, oaiReq)
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

func getModelDetails(oaiReq openai.ChatCompletionRequest) (*config.ModelDetails, string, error) {
	if oaiReq.Model == "random" {
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
