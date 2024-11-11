package main

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"log"
	"net/http"
	"simple-one-api/pkg/apis"
	"simple-one-api/pkg/embedding"
	"simple-one-api/pkg/initializer"
	"simple-one-api/pkg/mylog"
	"simple-one-api/pkg/mywebui"
	"simple-one-api/pkg/translation"
	"strings"

	//"log"
	"os"
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/handler"
	"time"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// 获取程序的第一个参数作为配置文件名
	var configName string
	if len(os.Args) > 1 {
		configName = os.Args[1]
	} else {
		configName = "config.json"
	}

	if err := initializer.Setup(configName); err != nil {
		return
	}
	defer initializer.Cleanup()

	// 创建一个 Gin 路由器实例
	r := gin.New()
	r.Use(gin.Recovery())

	// 配置 CORS 中间件
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // 允许所有来源，如果需要限制来源，可以将 "*" 替换为具体的 URL
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "Access-Control-Request-Private-Network"},
		ExposeHeaders:    []string{"Content-Length", "Access-Control-Allow-Private-Network"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.OPTIONS("/*path", func(c *gin.Context) {
		if c.GetHeader("Access-Control-Request-Private-Network") == "true" {
			c.Header("Access-Control-Allow-Private-Network", "true")
		}
		if c.GetHeader("Access-Control-Request-Credentials") == "true" {
			c.Header("Access-Control-Request-Credentials", "true")
		}
		c.Status(204)
	})

	mylog.Logger.Info("check EnableWeb config", zap.Bool("config.GSOAConf.EnableWeb", config.GSOAConf.EnableWeb))
	if config.GSOAConf.EnableWeb {
		mylog.Logger.Info("web enabled")
		// 设置静态文件夹
		r.Static("/static", "./static")

		// 设置根路径访问静态文件
		r.StaticFile("/", "./static/index.html")

		// 动态路由处理所有html文件
		r.GET("/:filename", func(c *gin.Context) {
			filename := c.Param("filename")
			if strings.HasSuffix(filename, ".html") {
				c.File("./static/" + filename)
			} else {
				c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
			}
		})
	}
	// 添加POST请求方法处理
	//r.POST("/v1/chat/completions", handler.OpenAIHandler)
	r.GET("/v1/models", apis.ModelsHandler)
	r.GET("/v1/models/:model", apis.RetrieveModelHandler)

	r.POST("/v2/translate", translation.TranslateV2Handler)
	r.POST("/translate", translation.TranslateV1Handler)

	//r.POST("/v1/embeddings", embedding.EmbeddingsHandler)

	r.GET("/multimodelcall", mywebui.WSMultiModelCallHandler)

	// 啥也不错，有些客户端真的很无语，不知道会怎么补全，尽量兼容吧
	v1 := r.Group("/v1")
	{
		// 中间件检查路径是否以 /v1/chat/completions 结尾
		v1.POST("/*path", func(c *gin.Context) {
			if strings.HasSuffix(c.Request.URL.Path, "/v1/chat/completions") || strings.HasSuffix(c.Request.URL.Path, "/chat/completions") || strings.HasSuffix(c.Request.URL.Path, "/v1") {
				handler.OpenAIHandler(c)
				return
			} else if strings.HasSuffix(c.Request.URL.Path, "/v1/translate") {
				translation.TranslateV1Handler(c)
				return
			} else if strings.HasSuffix(c.Request.URL.Path, "/v1/embeddings") {
				embedding.EmbeddingsHandler(c)
				return
			}
			c.JSON(http.StatusNotFound, gin.H{"error": "Path not found"})
		})
	}
	// 启动服务器，使用配置中的端口
	if err := r.Run(config.ServerPort); err != nil {
		mylog.Logger.Error(err.Error())
		return
	}
}
