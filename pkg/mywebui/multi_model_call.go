package web

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

type FormData struct {
	Prompt      string   `json:"prompt"`
	Temperature float64  `json:"temperature"`
	MaxTokens   int      `json:"maxTokens"`
	TopK        int      `json:"topK"`
	Models      []string `json:"models"`
	System      string   `json:"system"`
	RunCount    int      `json:"runCount"`
}

func MultiModelCallHandler(c *gin.Context) {
	var formData FormData
	if err := c.ShouldBindJSON(&formData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 在这里处理接收到的数据，例如执行一些逻辑操作
	// 这里仅打印接收到的数据作为示例
	c.JSON(http.StatusOK, gin.H{
		"message": "Success",
		"data":    formData,
	})
}
