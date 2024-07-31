package translation

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"
	"simple-one-api/pkg/mylog"
	"simple-one-api/pkg/utils"
	"sync"
)

type TranslationV2Request struct {
	Text       []string `json:"text" binding:"required"`
	TargetLang string   `json:"target_lang" binding:"required"`
	SourceLang string   `json:"source_lang,omitempty"`
	Stream     bool     `json:"stream,omitempty"`
}

type TranslationV2Response struct {
	Translations []TranslationV2Result `json:"translations"`
}

type TranslationV2Result struct {
	DetectedSourceLanguage string `json:"detected_source_language"`
	Text                   string `json:"text"`
}

func translateStream(c *gin.Context, transReq *TranslationV2Request) error {
	utils.SetEventStreamHeaders(c)

	cb := func(dstText string) {
		var tr TranslationV2Response
		tResult := TranslationV2Result{
			Text: dstText,
		}
		tr.Translations = append(tr.Translations, tResult)

		trJsonData, _ := json.Marshal(tr)

		_, err := c.Writer.WriteString("data: " + string(trJsonData) + "\n\n")
		if err != nil {
			mylog.Logger.Error("Error binding JSON:", zap.Error(err))
		}
		c.Writer.(http.Flusher).Flush()
	}

	_, err := LLMTranslateStream(transReq.Text[0], transReq.SourceLang, transReq.TargetLang, cb)
	if err != nil {
		mylog.Logger.Error("Error binding JSON:", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return err
	}
	return nil
}

func TranslateV2Handler(c *gin.Context) {
	var request TranslationV2Request
	if err := c.ShouldBindJSON(&request); err != nil {
		mylog.Logger.Error("Error binding JSON:", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if request.Stream {
		err := translateStream(c, &request)
		if err != nil {
			mylog.Logger.Error("Error translating stream:", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {

		var transResp TranslationV2Response
		var wg sync.WaitGroup
		var mu sync.Mutex
		sem := make(chan struct{}, 5)

		for _, srcText := range request.Text {
			wg.Add(1)
			sem <- struct{}{} // 占用一个并发槽

			go func(text string) {
				defer wg.Done()
				defer func() { <-sem }() // 释放一个并发槽

				var trv2 TranslationV2Result
				dstText, err := LLMTranslate(text, "", request.TargetLang)
				if err != nil {
					mylog.Logger.Error("Error translating stream:", zap.Error(err))
					return
				}

				trv2.Text = dstText

				mu.Lock()
				transResp.Translations = append(transResp.Translations, trv2)
				mu.Unlock()
			}(srcText)
		}

		wg.Wait()
		c.JSON(http.StatusOK, transResp)
	}
}
