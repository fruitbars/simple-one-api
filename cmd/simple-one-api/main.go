package main

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"log"
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

	// 初始化配置
	config.InitConfig(configName)

	if config.Debug == false {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建一个 Gin 路由器实例
	r := gin.Default()

	// 配置 CORS 中间件
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // 允许所有来源，如果需要限制来源，可以将 "*" 替换为具体的 URL
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// 添加POST请求方法处理
	r.POST("/v1/chat/completions", handler.OpenAIHandler)

	// 启动服务器，使用配置中的端口
	r.Run(config.ServerPort) // 使用配置文件中指定的端口号
}
