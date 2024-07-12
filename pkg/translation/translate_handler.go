package translation

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sashabaranov/go-openai"
	"net/http"
	"regexp"
	"simple-one-api/pkg/handler"
	"simple-one-api/pkg/mocks"
	"simple-one-api/pkg/mylog"
	"strconv"
	"strings"
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

func createGinContextFromLocalContext(lc *mocks.LocalContext) *gin.Context {
	// 创建一个新的 gin.Context
	c, _ := gin.CreateTestContext(lc)

	// 设置请求参数
	for key, value := range lc.Params {
		c.Params = append(c.Params, gin.Param{Key: key, Value: value})
	}

	return c
}

// multiUnescapeJSON 尝试多次解码被多次转义的JSON字符串。
func multiUnescapeJSON(escapedJSON string) (string, error) {
	// 去除可能的外部引号
	current := strings.Trim(escapedJSON, "`")

	// 可能的外部反引号已被去除，现在开始逐层解码
	for {
		// 尝试解码当前字符串
		decoded, err := strconv.Unquote("\"" + current + "\"")
		if err != nil {
			// 如果无法解码，返回当前的字符串和错误
			return current, err
		}
		// 如果解码后的字符串与之前相同，说明已经无法进一步解码
		if decoded == current {
			return decoded, nil
		}
		// 更新当前字符串为解码后的版本，以便进一步解码
		current = decoded
	}
}

// extractJSONFromMarkdown 提取并解码Markdown代码块中的JSON字符串。
func extractJSONFromMarkdown(input string) ([]string, error) {
	var results []string
	// 正则表达式匹配 ```json 或 ``` 后跟任意非贪婪字符，直到下一个 ```
	r := regexp.MustCompile("(?s)```(?:json)?\\s*(.*?)\\s*```")
	matches := r.FindAllStringSubmatch(input, -1)

	for _, match := range matches {
		// 获取捕获组中的JSON字符串
		jsonStr := match[1]
		// 预处理：去除潜在的非JSON字符和空格
		cleaned := strings.ReplaceAll(jsonStr, "\\n", "")
		cleaned = strings.ReplaceAll(cleaned, "\\t", "")
		cleaned = strings.TrimSpace(cleaned)

		// 多次解码直到得到一个有效的JSON字符串
		current := cleaned
		for {
			decoded, err := strconv.Unquote("\"" + current + "\"")
			if err != nil {
				// 如果无法进一步解码，则认为解码已经完成
				break
			}
			if decoded == current {
				// 如果解码后的字符串没有变化，返回解码后的字符串
				current = decoded
				break
			}
			// 更新当前字符串为解码后的版本，继续解码
			current = decoded
		}

		results = append(results, current)
	}

	return results, nil
}

func translate(transReq *TranslationRequest) (*TranslationResponse, error) {

	reqJsonstr, err := json.Marshal(transReq)
	if err != nil {
		mylog.Logger.Error(err.Error())
		return nil, err
	}

	prompt := "你是一个机器翻译接口，遵循以下输入输出协议，当接收到输入，直接给出输出即可，不要任何多余的回复\n输入协议(json格式)：\n```\n{\"text\":[\"Hello world!\",\"Good morning!\"],\"target_lang\":\"DE\"}\n```\n\n输出协议(json格式)：\n```\n{\n  \"translations\": [\n    {\n      \"detected_source_language\": \"EN\",\n      \"text\": \"Hallo, Welt!\"\n    },\n    {\n      \"detected_source_language\": \"EN\",\n      \"text\": \"Guten Morgen!\"\n    }\n  ]\n}\n```\n现在我的输入是：\n" + "```" + string(reqJsonstr) + "```"
	localContext := mocks.NewLocalContext(false)
	ginContext := createGinContextFromLocalContext(localContext)

	var req openai.ChatCompletionRequest
	req.Stream = false
	req.Model = "random"

	message := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: prompt,
	}

	req.Messages = append(req.Messages, message)

	handler.HandleOpenAIRequest(ginContext, &req)

	if !req.Stream {
		var resp openai.ChatCompletionResponse
		if err := json.Unmarshal(localContext.Body.Bytes(), &resp); err != nil {
			mylog.Logger.Error(err.Error())
			return nil, err
		}

		if len(resp.Choices) > 0 {
			mylog.Logger.Info(resp.Choices[0].Message.Content)
			var response TranslationResponse
			finalJSON, err := extractJSONFromMarkdown(resp.Choices[0].Message.Content)
			if err != nil {
				fmt.Println("Error unescaping JSON:", err)
				return nil, err
			}
			mylog.Logger.Info("|" + finalJSON[0] + "|")
			err = json.Unmarshal([]byte(finalJSON[0]), &response)
			if err != nil {
				mylog.Logger.Error(err.Error())
				return nil, err
			}
			return &response, nil
		}

	}
	return nil, nil

}

// RetrieveModelHandler RetrieveModelHandler
func TranslateHandler(c *gin.Context) {
	var request TranslationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	transResp, err := translate(&request)
	if err != nil {
		mylog.Logger.Error(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, transResp)
}
