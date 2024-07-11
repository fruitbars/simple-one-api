package apis

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

type TranslationRequest struct {
	Text       []string `json:"text" binding:"required"`
	TargetLang string   `json:"target_lang" binding:"required"`
}

type TranslationResponse struct {
	Translations []TranslationResult `json:"translations"`
}

type TranslationResult struct {
	DetectedSourceLanguage string `json:"detected_source_language"`
	Text                   string `json:"text"`
}

func translate(text []string, targetLang string) []TranslationResult {

	return nil
}

// RetrieveModelHandler RetrieveModelHandler
func TranslateHandler(c *gin.Context) {
	var request TranslationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	translations := translate(request.Text, request.TargetLang)
	response := TranslationResponse{Translations: translations}

	c.JSON(http.StatusOK, response)
}
