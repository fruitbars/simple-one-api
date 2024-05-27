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
	"simple-one-api/pkg/common"
	"simple-one-api/pkg/config"
	myopenai "simple-one-api/pkg/openai"
)

// validateAndFormatURL checks if the given URL matches the two specified formats and returns the formatted URL
func validateAndFormatURL(rawurl string) (string, bool) {
	parsedURL, err := url.Parse(rawurl)
	if err != nil {
		return "", false
	}

	// Regular expression to match "/v1" or "/v1/chat/completions"
	re := regexp.MustCompile(`^/v1(/chat/completions)?$`)

	// Check if the path matches the regular expression
	if re.MatchString(parsedURL.Path) {
		// Return the formatted URL as "https://domain/v1"
		formattedURL := fmt.Sprintf("%s://%s/v1", parsedURL.Scheme, parsedURL.Host)
		return formattedURL, true
	}

	return "", false
}

func OpenAI2OpenAIHandler(c *gin.Context, s *config.ModelDetails, oaiReq myopenai.OpenAIRequest) {
	apiKey := s.Credentials["api_key"]

	config := openai.DefaultConfig(apiKey)
	if s.ServerURL != "" {
		//serverUrl = defaultUrl
		formattedURL, isOk := validateAndFormatURL(s.ServerURL)
		if isOk {
			config.BaseURL = formattedURL
		}
	}

	log.Println(config.BaseURL)

	if oaiReq.Stream != nil && *oaiReq.Stream {
		common.SetEventStreamHeaders(c)

		openaiClient := openai.NewClientWithConfig(config)
		ctx := context.Background()

		req := adapter.OpenAIRequestToOpenAIRequest(oaiReq)

		stream, err := openaiClient.CreateChatCompletionStream(ctx, *req)
		if err != nil {
			fmt.Printf("ChatCompletionStream error: %v\n", err)
			return
		}
		defer stream.Close()

		fmt.Printf("Stream response: ")
		for {
			response, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				log.Println("Stream finished")
				//fmt.Println("\nStream finished")
				return
			} else if err != nil {
				log.Println("Stream error: ", err)
				return
			}

			respData, err := json.Marshal(&response)
			if err != nil {
				log.Println(err)
				c.JSON(http.StatusUnauthorized, err)
			} else {
				log.Println("response http data", string(respData))

				c.Writer.WriteString("data: " + string(respData) + "\n\n")
				c.Writer.(http.Flusher).Flush()

			}
		}
	}

}
