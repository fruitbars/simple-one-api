package handler

import (
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/sashabaranov/go-openai"
	"log"
	"net/http"
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/utils"
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

	//var oaiReq openai.OpenAIRequest
	var oaiReq openai.ChatCompletionRequest
	// 从请求中解析 JSON 到 reqBody
	if err := c.ShouldBindJSON(&oaiReq); err != nil {
		log.Println(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if config.APIKey != "" {
		apikey, err := utils.GetAPIKeyFromHeader(c)
		if err != nil {
			log.Println(err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		if config.APIKey != apikey {
			c.JSON(http.StatusUnauthorized, errors.New("invalid authorization"))
			return
		}
	}

	oaiData, _ := json.Marshal(oaiReq)
	log.Println(string(oaiData))

	var s *config.ModelDetails
	var modelName string
	var err error
	if oaiReq.Model == "random" {

		s, modelName, err = config.GetRandomEnabledModelDetailsV1()
		if err != nil {
			log.Println(err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		oaiReq.Model = modelName
	} else {
		s, err = config.GetModelService(oaiReq.Model)
		if err != nil {
			log.Println(err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	log.Println(*s, modelName)

	switch s.ServiceName {
	case "qianfan":
		err = OpenAI2QianFanHander(c, s, oaiReq)
	case "hunyuan":
		err = OpenAI2HunYuanHander(c, s, oaiReq)
	case "xinghuo":
		err = OpenAI2XingHuoHander(c, s, oaiReq)
	case "openai":
		err = OpenAI2OpenAIHandler(c, s, oaiReq)
	case "minimax":
		err = OpenAI2MinimaxHander(c, s, oaiReq)
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
	} else {
		if oaiReq.Stream == true {
			utils.SendOpenAIStreamEOFData(c)
		}
	}
}
