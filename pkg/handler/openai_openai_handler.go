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

func OpenAI2OpenAIHandler(c *gin.Context, s *config.ModelDetails, req openai.ChatCompletionRequest) error {
	apiKey := s.Credentials["api_key"]

	conf := openai.DefaultConfig(apiKey)
	if s.ServerURL != "" {
		//serverUrl = defaultUrl
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

		//req := adapter.OpenAIRequestToOpenAIRequest(oaiReq)

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
			log.Println(err)
			return err
		}

		myresp := adapter.OpenAIResponseToOpenAIResponse(&resp)
		myresp.Model = req.Model

		log.Println("响应：", *myresp)

		c.JSON(http.StatusOK, myresp)
	}

	return nil
}
