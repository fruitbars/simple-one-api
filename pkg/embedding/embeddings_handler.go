package embedding

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/embedding/baiduqianfan"
	"simple-one-api/pkg/embedding/oai"
	"simple-one-api/pkg/mycommon"
	"simple-one-api/pkg/mylimiter"
	"simple-one-api/pkg/mylog"
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

		return
	}

	clientModel := oaiEmbReq.Model
	mrModel := config.GetModelRedirect(s, serviceModelName)

	oaiEmbReq.Model = mrModel

	mylog.Logger.Info("Service details",
		zap.String("service_name", s.ServiceName),
		zap.String("client_model", clientModel),
		//zap.String("g_redirect_model", gRedirectModel),
		zap.String("service_model_name", serviceModelName),
		zap.String("redirect_model", mrModel),
		zap.String("last_model", oaiEmbReq.Model))

	creds, _ := mycommon.GetACredentials(s, oaiEmbReq.Model)

	/// 配置限流器
	limiter, timeout := setupLimiter(s, &s.EmbeddingLimit, oaiEmbReq.Model)
	if limiter != nil {
		handleRateLimiting(limiter, timeout)
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

	apiKey, _ := utils.GetStringFromMap(creds, config.KEYNAME_API_KEY)
	secretKey, _ := utils.GetStringFromMap(creds, config.KEYNAME_SECRET_KEY)

	var oaiResp interface{}

	switch s.ServiceName {
	case "qianfan":
		oaiResp, err = baiduqianfan.BaiduQianfanEmbedding(&oaiEmbReq, apiKey, secretKey, proxyTransport)
	case "openai":
		oaiResp, err = oai.OpenAIEmbedding(&oaiEmbReq, apiKey, proxyTransport)
	default:
		mylog.Logger.Error("Unsupported service", zap.String("service", s.ServiceName))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported service"})
		return
	}

	if err != nil {
		mylog.Logger.Error("Embedding service error", zap.String("service", s.ServiceName), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, oaiResp)

	return

}

func setupLimiter(s *config.ModelDetails, l *config.Limit, model string) (*mylimiter.Limiter, int) {

	lt, ln, timeout := mycommon.GetServiceLimiterDetailsLimit(l)

	if lt != "" && ln > 0 {
		limiterID := s.ServiceID + "_" + model

		return mylimiter.GetLimiter(limiterID, lt, ln), timeout
	}
	return nil, timeout
}

func handleRateLimiting(limiter *mylimiter.Limiter, timeout int) {
	if timeout <= 0 {
		timeout = 30 // 默认超时时间
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	startWaitTime := time.Now()
	if err := limiter.Wait(ctx); err != nil {
		elapsed := time.Since(startWaitTime)
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			mylog.Logger.Error("Failed to obtain token within specified time", zap.Error(err), zap.Int("timeout", timeout), zap.Duration("elapsed", elapsed))
		case errors.Is(err, context.Canceled):
			mylog.Logger.Error("Operation canceled", zap.Error(err), zap.Duration("elapsed", elapsed))
		default:
			mylog.Logger.Error("Unknown error occurred while waiting for token", zap.Error(err), zap.Duration("elapsed", elapsed))
		}
		return
	}

	mylog.Logger.Info("Rate limiting wait duration",
		zap.Duration("waited_for", time.Since(startWaitTime)))
}

func getEmbeddingModelDetails(oaiEmbReq *oai.EmbeddingRequest) (*config.ModelDetails, string, error) {

	s, err := config.GetModelService(oaiEmbReq.Model)
	if err != nil {
		return nil, "", err
	}

	return s, oaiEmbReq.Model, err
}
