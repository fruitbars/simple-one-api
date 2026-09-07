package apis

import (
	"net/http"
	"simple-one-api/pkg/config"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type Model struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

func ModelsHandler(c *gin.Context) {
	supportModels := config.CurrentSupportModels()
	keys := make([]string, 0, len(supportModels))

	for k := range supportModels {
		keys = append(keys, k)
	}
	sort.Strings(keys) // 对keys进行排序

	t := time.Now()
	models := make([]Model, 0, len(keys))
	for _, k := range keys {
		models = append(models, modelMetadata(k, t.Unix()))
	}
	c.IndentedJSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   models,
	})
}

// RetrieveModelHandler RetrieveModelHandler用于根据模型ID检索模型信息
func RetrieveModelHandler(c *gin.Context) {
	modelID := c.Param("model") // 从路径中获取模型ID

	if _, found := config.CurrentModelToService()[modelID]; found {
		c.IndentedJSON(http.StatusOK, modelMetadata(modelID, time.Now().Unix()))
		return
	}

	c.IndentedJSON(http.StatusNotFound, gin.H{"error": gin.H{
		"message": "The model '" + modelID + "' does not exist.",
		"type":    "invalid_request_error",
		"param":   "model",
		"code":    "model_not_found",
	}})
}

func modelMetadata(modelID string, created int64) Model {
	ownedBy := "simple-one-api"
	if details := config.CurrentModelToService()[modelID]; len(details) > 0 {
		ownedBy = strings.TrimSpace(details[0].Provider)
		if ownedBy == "" {
			ownedBy = strings.TrimSpace(details[0].ServiceName)
		}
		if ownedBy == "" {
			ownedBy = "simple-one-api"
		}
	}
	return Model{ID: modelID, Object: "model", Created: created, OwnedBy: ownedBy}
}
