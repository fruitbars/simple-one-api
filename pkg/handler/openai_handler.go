package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"io"
	"net/http"
	"simple-one-api/pkg/adapter"
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/mycommon"
	"simple-one-api/pkg/mylog"
	"simple-one-api/pkg/statistics"
	"simple-one-api/pkg/utils"
	"strings"
	"time"
)

var defaultReqTimeout = 120

// 定义 ReasoningMode 枚举类型
type ReasoningMode int

const (
	ReasoningNone ReasoningMode = iota // 0
	ReasoningR1
	ReasoningOpenrouterR1 // 1
)

type OAIRequestParam struct {
	ctx               context.Context
	chatCompletionReq *openai.ChatCompletionRequest
	modelDetails      *config.ModelDetails
	creds             map[string]interface{}
	httpTransport     *http.Transport
	ClientModel       string
	RM                ReasoningMode
	extraFields       map[string]json.RawMessage
}

// serviceHandlerMap maps service names to their corresponding handler functions
var serviceHandlerMap = map[string]func(*gin.Context, *OAIRequestParam) error{
	"qianfan":   OpenAI2QianFanHandler,
	"hunyuan":   OpenAI2HunYuanHandler,
	"xinghuo":   OpenAI2XingHuoHandler,
	"openai":    OpenAI2OpenAIHandler,
	"azure":     OpenAI2AzureOpenAIHandler,
	"deepseek":  OpenAI2OpenAIHandler,
	"zhipu":     OpenAI2OpenAIHandler,
	"minimax":   OpenAI2MinimaxHandler,
	"huoshan":   OpenAI2HuoShanHandler,
	"ollama":    OpenAI2OllamaHandler,
	"groq":      OpenAI2GroqOpenAIHandler,
	"gemini":    OpenAI2GeminiHandler,
	"dashscope": OpenAI2AliyunDashScopeHandler,
	"bailian":   OpenAI2AliyunBaiLianHandler,
	"vertexai":  OpenAI2VertexAIHandler,
	"claude":    OpenAI2ClaudeHandler,
	"dify":      OpenAI2DifyHandler,
}

func LogRequestDetails(c *gin.Context) {
	// 使用 zap 的字段记录功能来记录请求细节
	mylog.Logger.Debug("HTTP request details",
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path),
		zap.Any("parameters", c.Request.URL.Query()),
	)
}

func logOpenAIChatCompletionRequest(oaiReq *openai.ChatCompletionRequest) {
	if oaiReq == nil {
		return
	}

	mylog.Logger.Info("logOpenAIChatCompletionRequest", zap.Float32("TopP", oaiReq.TopP),
		zap.Float32("Temperature", oaiReq.Temperature), zap.Int("MaxTokens", oaiReq.MaxTokens),
		zap.String("model", oaiReq.Model), zap.Bool("IncludeReasoning", oaiReq.IncludeReasoning), zap.Int("N", oaiReq.N), zap.Float32("FrequencyPenalty", oaiReq.FrequencyPenalty))
}

func getBodyDataCopy(c *gin.Context) ([]byte, error) {
	body, err := c.GetRawData()
	if err != nil {
		//c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Unable to read request body"})
		return nil, err
	}

	// 将原始数据保存到上下文
	c.Set("rawData", body)

	// 重新设置请求体，以便后续能够读取
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	return body, nil
}

// OpenAIHandler handles POST requests on /v1/chat/completions path
func OpenAIHandler(c *gin.Context) {
	if !validateRequestMethod(c, "POST") {
		return
	}
	LogRequestDetails(c)

	apikey, err := utils.GetAPIKeyFromHeader(c)
	if err != nil {
		mylog.Logger.Error(err.Error())
	}

	isValid := validateAPIKey(apikey)
	if !isValid {
		err = errors.New("key is not valid")
		mylog.Logger.Error("key is not valid")
		sendErrorResponse(c, http.StatusUnauthorized, err.Error())
		return
	}

	bodyData, getBodyerr := getBodyDataCopy(c)
	if isRequestTooLarge(getBodyerr) {
		sendErrorResponse(c, http.StatusRequestEntityTooLarge, "request body too large")
		return
	}

	var oaiReq openai.ChatCompletionRequest
	if err := c.ShouldBindJSON(&oaiReq); err != nil {
		mylog.Logger.Error(err.Error())
		// 尝试重新解析请求体

		if getBodyerr != nil {
			mylog.Logger.Error(err.Error())
			sendErrorResponse(c, http.StatusBadRequest, err.Error())
			return
		}

		parsedReq, parseErr := mycommon.ParseChatCompletionRequest(bodyData)
		if parseErr != nil {
			mylog.Logger.Error("ParseChatCompletionRequest error: " + parseErr.Error())
			sendErrorResponse(c, http.StatusBadRequest, parseErr.Error())
			return
		}

		// 将重新解析的结果赋值给 oaiReq
		oaiReq = *parsedReq
	}

	mylog.Logger.Info("logOpenAIChatCompletionRequest", zap.Float32("TopP", oaiReq.TopP))
	logOpenAIChatCompletionRequest(&oaiReq)

	isValid, _ = config.ValidateAPIKeyAndModel(apikey, oaiReq.Model)
	if !isValid {
		err = errors.New("key not valid")
		mylog.Logger.Error(err.Error())
		sendErrorResponse(c, http.StatusUnauthorized, err.Error())
		return
	}

	mycommon.LogChatCompletionRequest(oaiReq)

	HandleOpenAIRequest(c, &oaiReq)

	return
}

func HandleOpenAIRequest(c *gin.Context, oaiReq *openai.ChatCompletionRequest) {

	clientModel := oaiReq.Model
	statistics.SetRoute(c, clientModel, "", "", "")
	extraFields := requestExtraFields(c)

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
	statistics.SetRoute(c, clientModel, s.ServiceID, s.ServiceName, oaiReq.Model)

	mylog.Logger.Info("Service details",
		zap.String("service_name", s.ServiceName),
		zap.String("client_model", clientModel),
		zap.String("g_redirect_model", gRedirectModel),
		zap.String("service_model_name", serviceModelName),
		zap.String("redirect_model", mrModel),
		zap.String("map_model", mpModel),
		zap.String("last_model", oaiReq.Model))

	if mycommon.IsMultiContentMessage(oaiReq.Messages) {
		isSupportMC := config.IsSupportMultiContent(oaiReq.Model)
		if !isSupportMC {
			mylog.Logger.Warn("model support vision", zap.Bool("isSupportMC", isSupportMC))
			//convert message
			adapter.OpenAIMultiContentRequestToOpenAIContentRequest(oaiReq)
			mylog.Logger.Debug("multimodal request normalized", zap.String("model", oaiReq.Model))
		} else {

		}
	}

	tokenCost := estimateRequestTokens(oaiReq)
	credentialCandidates := mycommon.OrderCredentialCandidates(s, clientModel, tokenCost)
	if len(credentialCandidates) == 0 {
		sendErrorResponse(c, http.StatusTooManyRequests, "no healthy credential is available")
		return
	}
	requestTimeout := s.Timeout
	if requestTimeout <= 0 {
		requestTimeout = defaultReqTimeout
	}
	requestCtx, cancelRequest := context.WithTimeout(c.Request.Context(), time.Duration(requestTimeout)*time.Second)
	defer cancelRequest()
	c.Request = c.Request.WithContext(requestCtx)

	oaiReqParam := &OAIRequestParam{
		ctx:               requestCtx,
		chatCompletionReq: oaiReq,
		modelDetails:      s,
		creds:             credentialCandidates[0].Credentials,
		ClientModel:       clientModel,
		extraFields:       extraFields,
	}

	if config.IsProxyEnabled(s) {
		proxyType, proxyAddr, transport, err := config.GetConfProxyTransport()
		if err != nil {
			mylog.Logger.Error("GetConfProxyTransport", zap.Error(err))
		} else {
			mylog.Logger.Debug("GetConfProxyTransport", zap.String("proxyType", proxyType), zap.String("proxyAddr", proxyAddr))
			oaiReqParam.httpTransport = transport
		}
	} else {
		mylog.Logger.Debug("GetConfProxyTransport proxy not enabled")
	}

	keepAllSystem := false

	//moonshot支持system模型，并且system可以放在任何位置并且可以是多个
	if s.Provider == "moonshot" || strings.HasPrefix(s.ServerURL, "https://api.moonshot.cn") {
		keepAllSystem = true
	}
	//mylog.Logger.Debug("oaiReq", zap.Any("oaiReq", oaiReq))
	oaiReq.Messages = mycommon.NormalizeMessages(oaiReq.Messages, keepAllSystem)

	var dispatchErr error
	for index, candidate := range credentialCandidates {
		oaiReqParam.creds = candidate.Credentials
		releaseLimits, limitErr := acquireAttemptLimits(requestCtx, s, candidate, oaiReq, clientModel, serviceModelName, mrModel)
		if limitErr != nil {
			dispatchErr = limitErr
			var waitErr *mycommon.LimitWaitError
			providerLimited := errors.As(limitErr, &waitErr) && (waitErr.Scope == mycommon.LimitScopeProvider || waitErr.Scope == mycommon.LimitScopeModel)
			if providerLimited || index == len(credentialCandidates)-1 {
				break
			}
			continue
		}
		mycommon.ReserveCredentialCapacity(candidate.Credentials, candidate.ID, clientModel, tokenCost)
		dispatchErr = dispatchToServiceHandler(c, oaiReqParam)
		releaseLimits()
		if dispatchErr == nil {
			config.RecordProviderResult(candidate.ID, clientModel, true)
			break
		}
		if isUpstreamRateLimit(dispatchErr) {
			mycommon.MarkCredentialCooldown(candidate.ID, clientModel, 30*time.Second)
		}
		retryable := shouldRetryCredential(dispatchErr)
		if retryable {
			config.RecordProviderResult(candidate.ID, clientModel, false)
		}
		if !retryable || c.Writer.Written() || index == len(credentialCandidates)-1 {
			break
		}
		mylog.Logger.Warn("provider credential failed; trying next pooled credential",
			zap.String("credential_id", candidate.ID), zap.Int("next_index", index+1), zap.Error(dispatchErr))
	}
	if dispatchErr != nil {
		mylog.Logger.Error(dispatchErr.Error())
		sendErrorResponse(c, dispatchErrorStatus(dispatchErr), dispatchErr.Error())
		return
	}

	if oaiReq.Stream {
		utils.SendOpenAIStreamEOFData(c)
	}
}

func isUpstreamRateLimit(err error) bool {
	var statusErr *utils.HTTPStatusError
	return errors.As(err, &statusErr) && statusErr.StatusCode == http.StatusTooManyRequests
}

func dispatchErrorStatus(err error) int {
	var waitErr *mycommon.LimitWaitError
	if errors.As(err, &waitErr) {
		return http.StatusTooManyRequests
	}
	var statusErr *utils.HTTPStatusError
	if errors.As(err, &statusErr) && statusErr.StatusCode >= 400 && statusErr.StatusCode < 600 {
		return statusErr.StatusCode
	}
	return http.StatusInternalServerError
}

func acquireAttemptLimits(ctx context.Context, service *config.ModelDetails, credential mycommon.CredentialSelection, request *openai.ChatCompletionRequest, modelNames ...string) (func(), error) {
	tokenCost := estimateRequestTokens(request)
	if request != nil {
		modelNames = append(modelNames, request.Model)
	}
	return acquireAttemptLimitsWithTokenCost(ctx, service, credential, tokenCost, modelNames...)
}

func acquireAttemptLimitsWithTokenCost(ctx context.Context, service *config.ModelDetails, credential mycommon.CredentialSelection, tokenCost int, modelNames ...string) (func(), error) {
	targets := []mycommon.LimitTarget{
		{Key: credential.ID + ":credential", Scope: mycommon.LimitScopeCredential, Limit: mycommon.GetCredentialLimits(credential.Credentials)},
	}
	if limit, name, ok := mycommon.GetCredentialModelLimit(credential.Credentials, modelNames...); ok {
		targets = append(targets, mycommon.LimitTarget{Key: credential.ID + ":credential:model:" + name, Scope: mycommon.LimitScopeCredentialModel, Limit: limit})
	}
	if limit, name, ok := config.ModelLimitFor(service, modelNames...); ok {
		targets = append(targets, mycommon.LimitTarget{Key: service.ServiceID + ":provider:model:" + name, Scope: mycommon.LimitScopeModel, Limit: limit})
	}
	targets = append(targets, mycommon.LimitTarget{Key: service.ServiceID + ":provider", Scope: mycommon.LimitScopeProvider, Limit: service.Limit})
	return mycommon.AcquireCombinedLimits(ctx, targets, tokenCost, defaultReqTimeout)
}

func estimateRequestTokens(request *openai.ChatCompletionRequest) int {
	if request == nil {
		return 1
	}
	payload, _ := json.Marshal(struct {
		Messages []openai.ChatCompletionMessage `json:"messages"`
		Tools    []openai.Tool                  `json:"tools,omitempty"`
	}{Messages: request.Messages, Tools: request.Tools})
	inputEstimate := (len(payload) + 3) / 4
	outputReservation := request.MaxCompletionTokens
	if outputReservation <= 0 {
		outputReservation = request.MaxTokens
	}
	if inputEstimate+outputReservation < 1 {
		return 1
	}
	return inputEstimate + outputReservation
}

func shouldRetryCredential(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	status := 0
	var statusErr *utils.HTTPStatusError
	if errors.As(err, &statusErr) {
		status = statusErr.StatusCode
	}
	var apiErr *openai.APIError
	if status == 0 && errors.As(err, &apiErr) {
		status = apiErr.HTTPStatusCode
	}
	if status == 0 {
		return true
	}
	return status == http.StatusUnauthorized || status == http.StatusForbidden || status == http.StatusRequestTimeout || status == http.StatusConflict || status == http.StatusTooManyRequests || status >= http.StatusInternalServerError
}

func requestExtraFields(c *gin.Context) map[string]json.RawMessage {
	value, ok := c.Get("rawData")
	if !ok {
		return nil
	}
	body, ok := value.([]byte)
	if !ok || len(body) == 0 {
		return nil
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(body, &fields) != nil {
		return nil
	}
	return fields
}

// dispatchToServiceHandler dispatches the request to the appropriate service handler based on the service name
func dispatchToServiceHandler(c *gin.Context, oaiReqParam *OAIRequestParam) error {
	s := oaiReqParam.modelDetails
	protocol := strings.ToLower(strings.TrimSpace(s.UpstreamProtocol))
	switch protocol {
	case config.UpstreamProtocolResponses:
		return OpenAI2ResponsesHandler(c, oaiReqParam)
	case config.UpstreamProtocolAnthropicMessages:
		return OpenAI2ClaudeHandler(c, oaiReqParam)
	case config.UpstreamProtocolChatCompletions:
		return OpenAI2OpenAIHandler(c, oaiReqParam)
	}
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

func validateAPIKey(apikey string) bool {
	expected := config.CurrentAPIKey()
	if expected == "" {
		return true
	}

	if expected != apikey {
		return false
	}
	return true
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
