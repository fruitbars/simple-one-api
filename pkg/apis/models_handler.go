package apis

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"simple-one-api/pkg/config"
	"sort"
	"time"
)

type Model struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

func ModelsHandler(c *gin.Context) {
	var models []Model
	keys := make([]string, 0, len(config.ModelToService))

	for k := range config.SupportModels {
		keys = append(keys, k)
	}
	sort.Strings(keys) // 对keys进行排序

	t := time.Now()
	for _, k := range keys {
		models = append(models, Model{
			ID:      k,
			Object:  "model",
			Created: t.Unix(),
			OwnedBy: "openai",
		})
	}

	if len(models) > 0 {
		models = append(models, Model{
			ID:      "random",
			Object:  "model",
			Created: t.Unix(),
			OwnedBy: "openai",
		})
	}

	if len(models) == 0 {
		c.IndentedJSON(http.StatusNotFound, gin.H{"error": "No models found"})
		return
	}
	c.IndentedJSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   models,
	})
}

// RetrieveModelHandler RetrieveModelHandler用于根据模型ID检索模型信息
func RetrieveModelHandler(c *gin.Context) {
	modelID := c.Param("model") // 从路径中获取模型ID

	if _, found := config.ModelToService[modelID]; found {
		model := Model{
			ID:      "gpt-3.5-turbo-instruct",
			Object:  "model",
			Created: time.Now().Unix(),
			OwnedBy: "openai",
		}
		c.IndentedJSON(http.StatusOK, model)
		return
	}

	c.IndentedJSON(http.StatusNotFound, gin.H{"error": "Model not found"})
}
