package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sashabaranov/go-openai"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"simple-one-api/pkg/adapter"
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/utils"
	"strings"
)

// validateAndFormatURL checks if the given URL matches the two specified formats and returns the formatted URL
func validateAndFormatURL(rawurl string) (string, bool) {
	parsedURL, err := url.Parse(rawurl)
	if err != nil {
		return "", false
	}

	// Regular expression to match "/v1" to "/v50" or "/v1/chat/completions" to "/v50/chat/completions"
	re := regexp.MustCompile(`/v([1-9]|[1-4][0-9]|50)(/chat/completions)?$`)

	log.Println(rawurl)
	// Check if the path matches the regular expression
	if re.MatchString(parsedURL.Path) {
		// If the path matches "/v1/chat/completions" to "/v50/chat/completions"
		if re.MatchString(parsedURL.Path) && re.FindStringSubmatch(parsedURL.Path)[2] == "/chat/completions" {
			// Remove "/chat/completions" part
			formattedURL := fmt.Sprintf("%s://%s%s", parsedURL.Scheme, parsedURL.Host, parsedURL.Path[:len(parsedURL.Path)-len("/chat/completions")])
			return formattedURL, true
		}
		// If the path matches "/v1" to "/v50"
		return rawurl, true
	}

	return rawurl, false
}

func getDefaultServerURL(model string) string {
	var serverURL string

	switch {
	case strings.HasPrefix(model, "GLM-"):
		serverURL = "https://open.bigmodel.cn/api/paas/v4/chat/completions"
	case strings.HasPrefix(model, "deepseek-"):
		serverURL = "https://api.deepseek.com/v1"
	}

	return serverURL
}

func OpenAI2OpenAIHandler(c *gin.Context, s *config.ModelDetails, req openai.ChatCompletionRequest) error {
	apiKey := s.Credentials["api_key"]

	conf := openai.DefaultConfig(apiKey)

	var serverURL string
	if s.ServerURL == "" {
		serverURL = getDefaultServerURL(req.Model)
		log.Println("get default server_url", serverURL)
	} else {
		serverURL = s.ServerURL
	}

	if serverURL != "" {
		formattedURL, isOk := validateAndFormatURL(s.ServerURL)
		if isOk {
			conf.BaseURL = formattedURL
		}
	}

	log.Println(conf.BaseURL)

	if req.Stream {
		utils.SetEventStreamHeaders(c)

		openaiClient := openai.NewClientWithConfig(conf)
		ctx := context.Background()

		stream, err := openaiClient.CreateChatCompletionStream(ctx, req)
		if err != nil {
			log.Printf("ChatCompletionStream error: %v\n", err)
			return err
		}
		defer stream.Close()

		fmt.Printf("Stream response: ")
		for {
			response, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				log.Println("Stream finished")
				return nil
			} else if err != nil {
				log.Println(err)
				return err
			}

			response.Model = req.Model
			respData, err := json.Marshal(&response)
			if err != nil {
				log.Println(err)
				return err
			} else {
				log.Println("response http data", string(respData))

				c.Writer.WriteString("data: " + string(respData) + "\n\n")
				c.Writer.(http.Flusher).Flush()
			}
		}

	} else {
		openaiClient := openai.NewClientWithConfig(conf)
		//ctx := context.Background()

		//req := adapter.OpenAIRequestToOpenAIRequest(oaiReq)
		resp, err := openaiClient.CreateChatCompletion(
			context.Background(),
			req,
		)

		if err != nil {
			log.Println(err.Error())
			return err
		}

		myresp := adapter.OpenAIResponseToOpenAIResponse(&resp)
		myresp.Model = req.Model

		log.Println("响应：", *myresp)

		c.JSON(http.StatusOK, myresp)
	}

	return nil
}
