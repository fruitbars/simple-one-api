package handler

import (
	"bytes"
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/sashabaranov/go-openai"
	"io"
	"log"
	"net/http"
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/utils"
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
}

func LogRequestBody(c *gin.Context) {
	// 读取请求消息体
	body, err := c.GetRawData()
	if err != nil {
		log.Println("Error reading request body: ", err)
		return
	}

	// 将消息体转换为字符串并记录
	requestBody := string(body)
	log.Println("Request Body: ", requestBody)

	// 重置请求体，以便后续处理程序可以读取它
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
}

// LogRequestDetails logs detailed request information for debugging purposes
func LogRequestDetails(c *gin.Context) {
	log.Printf("HTTP Request Method: %s, Request Path: %s\n", c.Request.Method, c.Request.URL.Path)
	log.Println("Request Parameters: ", c.Request.URL.Query())
	log.Println("Request Headers: ", c.Request.Header)
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
		log.Println(err)
		sendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := validateAPIKey(c); err != nil {
		log.Println(err)
		sendErrorResponse(c, http.StatusUnauthorized, err.Error())
		return
	}

	handleOpenAIRequest(c, oaiReq)
}

func handleOpenAIRequest(c *gin.Context, oaiReq openai.ChatCompletionRequest) {
	s, modelName, err := getModelDetails(oaiReq)
	if err != nil {
		log.Println(err)
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
		log.Printf("qps limit to: %v\n", s.Limit.QPS)
		if err := s.Limiter.Wait(ctx); err != nil {
			log.Println(err)
			waitDuration := time.Since(startWaitTime)
			log.Printf("waited for: %v\n", waitDuration)
			sendErrorResponse(c, http.StatusTooManyRequests, "Request rate limit exceeded")
			return
		}
		log.Printf("waited for: %v", time.Since(startWaitTime))
	} else if s.ConcurrencyLimiter != nil {
		log.Printf("concurrency limit to: %v\n", s.Limit.Concurrency)
		select {
		case <-s.ConcurrencyLimiter: // attempt to get a token from concurrency limiter
			//log.Println("token acquired")
			defer func() {
				waitDuration := time.Since(startWaitTime)
				log.Printf("releasing token, use token time: %v\n", waitDuration)
				s.ConcurrencyLimiter <- struct{}{} // release token upon request completion
			}()
		case <-ctx.Done(): // handle timeout
			waitDuration := time.Since(startWaitTime)
			log.Printf("timeout after waiting for: %v\n", waitDuration)
			sendErrorResponse(c, http.StatusBadRequest, "Concurrency limit exceeded")
			return
		}
		log.Printf("concurrency waited for: %v", time.Since(startWaitTime))
	}

	oaiReq.Model = config.GetModelMapping(s, modelName)

	log.Println("map model:", modelName, "to", oaiReq.Model)
	log.Println(s.ServiceName)

	//oaiReq.Model = modelName

	if err := dispatchToServiceHandler(c, s, oaiReq); err != nil {
		log.Println("Error handling request:", err)
		sendErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	if oaiReq.Stream {
		utils.SendOpenAIStreamEOFData(c)
	}
}

// dispatchToServiceHandler dispatches the request to the appropriate service handler based on the service name
func dispatchToServiceHandler(c *gin.Context, s *config.ModelDetails, oaiReq openai.ChatCompletionRequest) error {
	if handler, ok := serviceHandlerMap[s.ServiceName]; ok {
		return handler(c, s, oaiReq)
	}
	return errors.New("Service handler not found")
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
		return errors.New("Invalid authorization")
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
