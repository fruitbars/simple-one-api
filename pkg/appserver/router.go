package appserver

import (
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"simple-one-api/internal/webui"
	"simple-one-api/pkg/apis"
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/embedding"
	"simple-one-api/pkg/handler"
	"simple-one-api/pkg/mywebui"
	"simple-one-api/pkg/translation"
)

type Options struct {
	EnableWeb                  bool
	TrustedLocalAdminBootstrap bool
}

const maxAPIRequestBodyBytes int64 = 8 << 20

func NewRouter() *gin.Engine {
	enableWeb := config.CurrentConfiguration().EnableWeb
	return NewRouterWithOptions(Options{EnableWeb: enableWeb})
}

func NewRouterWithOptions(options Options) *gin.Engine {
	if options.EnableWeb {
		announceAdminBootstrap()
	}
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "x-api-key", "anthropic-version", "anthropic-beta"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}))

	registerAPI(router, options)
	if options.EnableWeb {
		registerFrontend(router)
	}
	return router
}

func registerAPI(router *gin.Engine, options Options) {
	router.GET("/v1/models", requireAPIAccess(), apis.ModelsHandler)
	router.GET("/v1/models/:model", requireAPIAccess(), apis.RetrieveModelHandler)
	router.POST("/v2/translate", requireAPIAccess(), translation.TranslateV2Handler)
	router.POST("/translate", requireAPIAccess(), translation.TranslateV1Handler)
	if options.EnableWeb {
		router.GET("/multimodelcall", requireAPIAccess(), mywebui.WSMultiModelCallHandler)
	}

	admin := router.Group("/api/admin", requireAdminAccess(AdminAccessOptions{
		TrustLocalBootstrap: options.TrustedLocalAdminBootstrap,
	}))
	admin.GET("/status", apis.AdminStatusHandler)
	admin.GET("/config", apis.AdminConfigHandler)
	admin.GET("/config/draft", apis.AdminConfigDraftHandler)
	admin.POST("/config/validate", apis.AdminConfigValidateHandler)
	admin.POST("/config/revisions", apis.AdminConfigPublishHandler)
	admin.GET("/config/revisions", apis.AdminConfigRevisionsHandler)
	admin.GET("/logs", apis.AdminLogsHandler)
	admin.POST("/config/revisions/:id/activate", apis.AdminConfigActivateHandler)

	v1 := router.Group("/v1", requireAPIAccess(), limitRequestBody(maxAPIRequestBodyBytes))
	v1.POST("/*path", func(context *gin.Context) {
		requestPath := context.Request.URL.Path
		switch {
		case strings.HasSuffix(requestPath, "/v1/responses"), strings.HasSuffix(requestPath, "/responses"):
			handler.ResponsesHandler(context)
		case strings.HasSuffix(requestPath, "/v1/messages"), strings.HasSuffix(requestPath, "/messages"):
			handler.AnthropicMessagesHandler(context)
		case strings.HasSuffix(requestPath, "/v1/chat/completions"), strings.HasSuffix(requestPath, "/chat/completions"), strings.HasSuffix(requestPath, "/v1"):
			handler.OpenAIHandler(context)
		case strings.HasSuffix(requestPath, "/v1/translate"):
			translation.TranslateV1Handler(context)
		case strings.HasSuffix(requestPath, "/v1/embeddings"):
			embedding.EmbeddingsHandler(context)
		default:
			context.JSON(http.StatusNotFound, gin.H{"error": "Path not found"})
		}
	})
}

func limitRequestBody(maxBytes int64) gin.HandlerFunc {
	return func(context *gin.Context) {
		if context.Request.Body != nil {
			context.Request.Body = http.MaxBytesReader(context.Writer, context.Request.Body, maxBytes)
		}
		context.Next()
	}
}

func registerFrontend(router *gin.Engine) {
	webHandler := gin.WrapH(webui.Handler())
	router.GET("/", webHandler)
	router.HEAD("/", webHandler)
	router.GET("/assets/*filepath", webHandler)
	router.HEAD("/assets/*filepath", webHandler)
	router.NoRoute(func(context *gin.Context) {
		if shouldServeSPA(context.Request.URL.Path, context.Request.Method) {
			webHandler(context)
			return
		}
		context.JSON(http.StatusNotFound, gin.H{"error": "not found", "path": context.Request.URL.Path})
	})
}

func shouldServeSPA(requestPath string, method string) bool {
	if method != http.MethodGet {
		return false
	}
	if strings.HasPrefix(requestPath, "/v1/") || strings.HasPrefix(requestPath, "/api/") {
		return false
	}
	extension := path.Ext(requestPath)
	return extension == "" || extension == ".html"
}
