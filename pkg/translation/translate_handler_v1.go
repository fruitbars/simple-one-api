package translation

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"
	"simple-one-api/pkg/mylog"
	"simple-one-api/pkg/utils"
)

// TranslationRequest 定义请求的结构体
type TranslationV1Request struct {
	Text       string `json:"text" binding:"required"`
	SourceLang string `json:"source_lang,omitempty"`
	TargetLang string `json:"target_lang" binding:"required"`
	Stream     bool   `json:"stream,omitempty"`
}

// TranslationResponse 定义响应的结构体
type TranslationV1Response struct {
	Alternatives []string `json:"alternatives,omitempty"`
	Code         int      `json:"code"`
	Data         string   `json:"data"`
	ID           int64    `json:"id,omitempty"`
	Method       string   `json:"method,omitempty"`
	SourceLang   string   `json:"source_lang,omitempty"`
	TargetLang   string   `json:"target_lang"`
}

// translateHandler 处理翻译请求的函数
func TranslateV1Handler(c *gin.Context) {
	// 处理 Authorization 验证
	token := c.GetHeader("Authorization")
	if token == "" {
		token = c.Query("token")
		if token == "" {
			//c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			//return
		}
	}

	// 绑定请求 JSON 数据
	var req TranslationV1Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Stream {
		utils.SetEventStreamHeaders(c)

		cb := func(dstText string) {
			tr := TranslationV1Response{
				Data: dstText,
			}

			trJsonData, _ := json.Marshal(tr)

			_, err := c.Writer.WriteString("data: " + string(trJsonData) + "\n\n")
			if err != nil {
				mylog.Logger.Error("Error binding JSON:", zap.Error(err))
			}
			c.Writer.(http.Flusher).Flush()
		}

		_, err := LLMTranslateStream(req.Text, req.SourceLang, req.TargetLang, cb)
		if err != nil {
			mylog.Logger.Error("Error binding JSON:", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		return
	} else {
		targetText, err := LLMTranslate(req.Text, req.SourceLang, req.TargetLang)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		response := TranslationV1Response{
			Code:       200,
			Data:       targetText,
			ID:         8356681003,
			Method:     "Pro",
			SourceLang: req.SourceLang,
			TargetLang: req.TargetLang,
		}

		c.JSON(http.StatusOK, response)
		return
	}

	return
}
