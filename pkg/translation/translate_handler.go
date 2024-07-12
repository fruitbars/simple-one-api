package translation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"io"
	"net/http"
	"regexp"
	"simple-one-api/pkg/mylog"
	"simple-one-api/pkg/simple_client"
	"simple-one-api/pkg/utils"
	"strconv"
	"strings"
)

type TranslationRequest struct {
	Text       []string `json:"text" binding:"required"`
	TargetLang string   `json:"target_lang" binding:"required"`
	Stream     bool     `json:"stream,omitempty"`
}

type TranslationResponse struct {
	Translations []TranslationResult `json:"translations"`
}

type TranslationResult struct {
	DetectedSourceLanguage string `json:"detected_source_language"`
	Text                   string `json:"text"`
}

// multiUnescapeJSON 尝试多次解码被多次转义的JSON字符串。
func multiUnescapeJSON(escapedJSON string) (string, error) {
	current := strings.Trim(escapedJSON, "`")
	for {
		decoded, err := strconv.Unquote("\"" + current + "\"")
		if err != nil {
			return current, err
		}
		if decoded == current {
			return decoded, nil
		}
		current = decoded
	}
}

// extractJSONFromMarkdown 提取并解码Markdown代码块中的JSON字符串。
func extractJSONFromMarkdown(input string) ([]string, error) {
	var results []string
	r := regexp.MustCompile("(?s)```(?:json)?\\s*(.*?)\\s*```")
	matches := r.FindAllStringSubmatch(input, -1)

	for _, match := range matches {
		jsonStr := match[1]
		cleaned := strings.ReplaceAll(jsonStr, "\\n", "")
		cleaned = strings.ReplaceAll(cleaned, "\\t", "")
		cleaned = strings.TrimSpace(cleaned)

		current := cleaned
		for {
			decoded, err := strconv.Unquote("\"" + current + "\"")
			if err != nil {
				break
			}
			if decoded == current {
				current = decoded
				break
			}
			current = decoded
		}
		results = append(results, current)
	}
	return results, nil
}

func createTranslationPrompt(reqJsonstr string) string {
	return fmt.Sprintf("你是一个机器翻译接口，遵循以下输入输出协议，当接收到输入，直接给出输出即可，不要任何多余的回复\n输入协议(json格式)：\n```\n{\"text\":[\"Hello world!\",\"Good morning!\"],\"target_lang\":\"DE\"}\n```\n\n输出协议(json格式)：\n```\n{\n  \"translations\": [\n    {\n      \"detected_source_language\": \"EN\",\n      \"text\": \"Hallo, Welt!\"\n    },\n    {\n      \"detected_source_language\": \"EN\",\n      \"text\": \"Guten Morgen!\"\n    }\n  ]\n}\n```\n现在我的输入是：\n```%s```", reqJsonstr)
}

func createTranslationPromptStream(reqJsonstr string) string {
	return fmt.Sprintf("你是一个机器翻译接口，遵循以下输入输出协议，当接收到输入，直接给出输出即可，不要任何多余的回复\n输入协议(json格式)：\n```\n{\"text\":[\"Hello world!\"],\"target_lang\":\"DE\"}\n```\n\n翻译结果直接输出：\n\nHallo, Welt!\n\n现在我的输入是：\n```\n%s\n```\n输出：\n", reqJsonstr)
}

func handleTranslationResponse(responseContent string) (*TranslationResponse, error) {
	var response TranslationResponse
	finalJSON, err := extractJSONFromMarkdown(responseContent)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal([]byte(finalJSON[0]), &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func translateStream(c *gin.Context, transReq *TranslationRequest) error {
	reqJsonstr, err := json.Marshal(transReq)
	if err != nil {
		mylog.Logger.Error("Error marshalling request:", zap.Error(err))
		return err
	}

	prompt := createTranslationPromptStream(string(reqJsonstr))

	var req openai.ChatCompletionRequest
	req.Stream = true
	req.Model = "random"

	message := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: prompt,
	}

	req.Messages = append(req.Messages, message)

	client := simple_client.NewSimpleClient("")

	chatStream, err := client.CreateChatCompletionStream(context.Background(), req)
	if err != nil {
		mylog.Logger.Error("Error creating chat completion stream:", zap.Error(err))
		return err
	}

	utils.SetEventStreamHeaders(c)
	for {
		chatResp, err := chatStream.Recv()
		if errors.Is(err, io.EOF) {
			mylog.Logger.Info("Stream finished")
			return nil
		}
		if err != nil {
			mylog.Logger.Error("Error receiving chat response:", zap.Error(err))
			return err
		}

		if chatResp == nil {
			continue
		}

		mylog.Logger.Info("Received chat response", zap.Any("chatResp", chatResp))
		if len(chatResp.Choices) > 0 {

			translatedText := chatResp.Choices[0].Delta.Content
			tr := TranslationResponse{
				Translations: []TranslationResult{
					{
						DetectedSourceLanguage: "",
						Text:                   translatedText,
					},
				},
			}

			trJsonData, _ := json.Marshal(tr)
			_, err = c.Writer.WriteString("data: " + string(trJsonData) + "\n\n")
			c.Writer.(http.Flusher).Flush()
			//c.JSON(http.StatusOK, tr)
		}
	}
}

func translate(transReq *TranslationRequest) (*TranslationResponse, error) {
	reqJsonstr, err := json.Marshal(transReq)
	if err != nil {
		mylog.Logger.Error("Error marshalling request:", zap.Error(err))
		return nil, err
	}

	prompt := createTranslationPrompt(string(reqJsonstr))

	var req openai.ChatCompletionRequest
	req.Stream = false
	req.Model = "random"

	message := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: prompt,
	}

	req.Messages = append(req.Messages, message)

	client := simple_client.NewSimpleClient("")

	resp, err := client.CreateChatCompletion(context.Background(), req)
	if err != nil {
		mylog.Logger.Error("Error creating chat completion:", zap.Error(err))
		return nil, err
	}

	if len(resp.Choices) > 0 {
		mylog.Logger.Info("Received chat response", zap.String("content", resp.Choices[0].Message.Content))
		return handleTranslationResponse(resp.Choices[0].Message.Content)
	}

	return nil, nil
}

func TranslateHandler(c *gin.Context) {
	var request TranslationRequest
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
		transResp, err := translate(&request)
		if err != nil {
			mylog.Logger.Error("Error translating:", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, transResp)
	}
}
