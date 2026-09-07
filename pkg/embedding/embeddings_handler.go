package embedding

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/embedding/baiduqianfan"
	"simple-one-api/pkg/embedding/oai"
	"simple-one-api/pkg/mycommon"
	"simple-one-api/pkg/mylog"
	"simple-one-api/pkg/statistics"
	"simple-one-api/pkg/utils"
	"time"
)

func EmbeddingsHandler(c *gin.Context) {
	var oaiEmbReq oai.EmbeddingRequest
	if err := c.ShouldBindJSON(&oaiEmbReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	mylog.Logger.Info("EmbeddingsHandler", zap.Any("req", oaiEmbReq))

	s, serviceModelName, err := getEmbeddingModelDetails(&oaiEmbReq)
	if err != nil {
		mylog.Logger.Error(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	clientModel := oaiEmbReq.Model
	mrModel := config.GetModelRedirect(s, serviceModelName)

	oaiEmbReq.Model = mrModel
	statistics.SetRoute(c, clientModel, s.ServiceID, s.ServiceName, oaiEmbReq.Model)

	mylog.Logger.Info("Service details",
		zap.String("service_name", s.ServiceName),
		zap.String("client_model", clientModel),
		//zap.String("g_redirect_model", gRedirectModel),
		zap.String("service_model_name", serviceModelName),
		zap.String("redirect_model", mrModel),
		zap.String("last_model", oaiEmbReq.Model))

	candidates := mycommon.GetCredentialCandidates(s, clientModel)
	if len(candidates) == 0 {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "no healthy credential is available"})
		return
	}

	var proxyTransport *http.Transport
	if config.IsProxyEnabled(s) {
		proxyType, proxyAddr, transport, err := config.GetConfProxyTransport()
		if err != nil {
			mylog.Logger.Error("GetConfProxyTransport", zap.Error(err))
		} else {
			proxyTransport = transport
			mylog.Logger.Debug("GetConfProxyTransport", zap.String("proxyType", proxyType), zap.String("proxyAddr", proxyAddr))
		}
	} else {
		mylog.Logger.Debug("GetConfProxyTransport proxy not enabled")
	}

	var oaiResp interface{}
	requestTimeout := s.Timeout
	if requestTimeout <= 0 {
		requestTimeout = 120
	}
	requestCtx, cancelRequest := context.WithTimeout(c.Request.Context(), time.Duration(requestTimeout)*time.Second)
	defer cancelRequest()

	for index, candidate := range candidates {
		release, limitErr := acquireEmbeddingLimits(requestCtx, s, candidate, &oaiEmbReq, clientModel, serviceModelName, mrModel)
		if limitErr != nil {
			err = limitErr
			var waitErr *mycommon.LimitWaitError
			providerLimited := errors.As(limitErr, &waitErr) && (waitErr.Scope == mycommon.LimitScopeProvider || waitErr.Scope == mycommon.LimitScopeModel)
			if providerLimited || index == len(candidates)-1 {
				break
			}
			continue
		}

		apiKey, _ := utils.GetStringFromMap(candidate.Credentials, config.KEYNAME_API_KEY)
		secretKey, _ := utils.GetStringFromMap(candidate.Credentials, config.KEYNAME_SECRET_KEY)
		switch s.ServiceName {
		case "qianfan":
			oaiResp, err = baiduqianfan.BaiduQianfanEmbedding(requestCtx, &oaiEmbReq, apiKey, secretKey, proxyTransport)
		case "openai":
			oaiResp, err = oai.OpenAIEmbedding(requestCtx, &oaiEmbReq, apiKey, s.ServerURL, proxyTransport)
		default:
			release()
			mylog.Logger.Error("Unsupported service", zap.String("service", s.ServiceName))
			c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported service"})
			return
		}
		release()
		if err == nil {
			config.RecordProviderResult(candidate.ID, clientModel, true)
			break
		}
		retryable := shouldRetryEmbeddingCredential(err)
		if retryable {
			config.RecordProviderResult(candidate.ID, clientModel, false)
		}
		if !retryable || index == len(candidates)-1 {
			break
		}
		mylog.Logger.Warn("embedding credential failed; trying next pooled credential",
			zap.String("credential_id", candidate.ID), zap.Int("next_index", index+1), zap.Error(err))
	}

	if err != nil {
		mylog.Logger.Error("Embedding service error", zap.String("service", s.ServiceName), zap.Error(err))
		status := http.StatusInternalServerError
		var waitErr *mycommon.LimitWaitError
		if errors.As(err, &waitErr) {
			status = http.StatusTooManyRequests
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, oaiResp)

}

func acquireEmbeddingLimits(ctx context.Context, service *config.ModelDetails, credential mycommon.CredentialSelection, request *oai.EmbeddingRequest, modelNames ...string) (func(), error) {
	if request != nil {
		modelNames = append(modelNames, request.Model)
	}
	targets := []mycommon.LimitTarget{
		{Key: credential.ID + ":credential", Scope: mycommon.LimitScopeCredential, Limit: mycommon.GetCredentialLimits(credential.Credentials)},
	}
	if limit, name, ok := mycommon.GetCredentialModelLimit(credential.Credentials, modelNames...); ok {
		targets = append(targets, mycommon.LimitTarget{Key: credential.ID + ":credential:model:" + name, Scope: mycommon.LimitScopeCredentialModel, Limit: limit})
	}
	if limit, name, ok := config.ModelLimitFor(service, modelNames...); ok {
		targets = append(targets, mycommon.LimitTarget{Key: service.ServiceID + ":provider:model:" + name, Scope: mycommon.LimitScopeModel, Limit: limit})
	}
	targets = append(targets, mycommon.LimitTarget{Key: service.ServiceID + ":embedding:provider", Scope: mycommon.LimitScopeProvider, Limit: service.EmbeddingLimit})
	return mycommon.AcquireCombinedLimits(ctx, targets, estimateEmbeddingTokens(request), 30)
}

func estimateEmbeddingTokens(request *oai.EmbeddingRequest) int {
	if request == nil {
		return 1
	}
	payload, _ := json.Marshal(request.Input)
	estimate := (len(payload) + 3) / 4
	if estimate < 1 {
		return 1
	}
	return estimate
}

func shouldRetryEmbeddingCredential(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var statusErr *utils.HTTPStatusError
	if !errors.As(err, &statusErr) {
		return true
	}
	status := statusErr.StatusCode
	return status == http.StatusUnauthorized || status == http.StatusForbidden || status == http.StatusRequestTimeout || status == http.StatusConflict || status == http.StatusTooManyRequests || status >= http.StatusInternalServerError
}

func getEmbeddingModelDetails(oaiEmbReq *oai.EmbeddingRequest) (*config.ModelDetails, string, error) {

	s, err := config.GetModelService(oaiEmbReq.Model)
	if err != nil {
		return nil, "", err
	}

	return s, oaiEmbReq.Model, err
}
