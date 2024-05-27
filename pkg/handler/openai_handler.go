package handler

import (
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/openai"
)

func DebugInfo(c *gin.Context) {
	// 打印请求方法
	log.Printf("HTTP请求方法：%s\n", c.Request.Method)

	// 打印请求路径
	log.Printf("请求路径：%s\n", c.Request.URL.Path)

	// 打印请求参数
	queryParam := c.Request.URL.Query()
	log.Println("请求参数：")
	for key, value := range queryParam {
		log.Printf("%s: %s\n", key, value)
	}

	// 打印请求头部信息
	log.Println("请求头部信息：")
	for key, value := range c.Request.Header {
		log.Printf("%s: %s\n", key, value)
	}
}

// OpenAIHandler 处理 /v1/chat/completions 路径上的 POST 请求
func OpenAIHandler(c *gin.Context) {
	DebugInfo(c)
	// 检查请求方法是否为POST
	if c.Request.Method != "POST" {
		log.Println("not post")
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "Only POST method is accepted"})
		return
	}

	var oaiReq openai.OpenAIRequest
	// 从请求中解析 JSON 到 reqBody
	if err := c.ShouldBindJSON(&oaiReq); err != nil {
		log.Println(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Println(oaiReq)

	s, err := config.GetModelService(oaiReq.Model)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Println(*s)

	switch s.ServiceName {
	case "qianfan":
		OpenAI2QianFanHander(c, s, oaiReq)
	case "hunyuan":
		OpenAI2HunYuanHander(c, s, oaiReq)
	case "xinghuo":
		OpenAI2XingHuoHander(c, s, oaiReq)
	case "openai":
		OpenAI2OpenAIHandler(c, s, oaiReq)
	case "minimax":
		OpenAI2MinimaxHander(c, s, oaiReq)

	}

}
