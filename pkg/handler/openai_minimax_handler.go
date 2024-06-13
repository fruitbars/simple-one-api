package handler

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sashabaranov/go-openai"
	"io"
	"log"
	"net/http"
	"simple-one-api/pkg/adapter"
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/llm/minimax"
	mycommon "simple-one-api/pkg/utils"
	"strings"
)

func OpenAI2MinimaxHandler(c *gin.Context, s *config.ModelDetails, oaiReq openai.ChatCompletionRequest) error {
	apiKey := s.Credentials[config.KEYNAME_API_KEY]
	groupID := s.Credentials[config.KEYNAME_GROUP_ID]

	if s.ServerURL == "" {
		//serverUrl = defaultUrl
		s.ServerURL = "https://api.minimax.chat/v1/text/chatcompletion_pro"
	}

	serverUrl := fmt.Sprintf("%s?GroupId=%s", s.ServerURL, groupID)
	bearerToken := fmt.Sprintf("Bearer %s", apiKey)

	minimaxReq := adapter.OpenAIRequestToMinimaxRequest(oaiReq)

	jsonData, err := json.Marshal(minimaxReq)
	if err != nil {
		log.Println("Error marshalling JSON:", err)
		return err
	}

	log.Println(string(jsonData))

	if oaiReq.Stream {

		request, err := http.NewRequest("POST", serverUrl, bytes.NewBuffer(jsonData))
		if err != nil {
			log.Println("Error creating request:", err)
			return err
		}

		request.Header.Add("Authorization", bearerToken)
		request.Header.Add("Content-Type", "application/json")

		// 使用http.Client发送请求
		client := &http.Client{}
		response, err := client.Do(request)
		if err != nil {
			log.Println("Error sending request:", err)
			return err
		}
		defer response.Body.Close()

		id := uuid.New()
		mycommon.SetEventStreamHeaders(c)
		// 处理SSE响应
		reader := bufio.NewReader(response.Body)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					break
				}

				log.Println("Error reading response:", err)
				return err
			}

			// 去掉行尾的换行符
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "data: ") {
				line = strings.TrimPrefix(line, "data: ")
			}

			if line == "" {
				// 忽略空行
				continue
			}

			log.Println(line)

			var minimaxresp minimax.MinimaxResponse
			json.Unmarshal([]byte(line), &minimaxresp)

			oaiRespStream := adapter.MinimaxResponseToOpenAIStreamResponse(&minimaxresp)
			oaiRespStream.ID = id.String()
			oaiRespStream.Model = oaiReq.Model
			respData, err := json.Marshal(&oaiRespStream)
			if err != nil {
				log.Println(err)
				return err
			} else {
				log.Println("response http data", string(respData))

				if oaiRespStream.Error != nil {
					log.Println(oaiRespStream.Error)
					errInfo, _ := json.Marshal(oaiRespStream.Error)
					return errors.New(string(errInfo))
				} else {
					c.Writer.WriteString("data: " + string(respData) + "\n\n")
					c.Writer.(http.Flusher).Flush()
				}
			}

		}

	} else {
		request, err := http.NewRequest("POST", serverUrl, bytes.NewBuffer(jsonData))
		if err != nil {
			log.Println("Error creating request:", err)
			return err
		}

		request.Header.Add("Authorization", bearerToken)
		request.Header.Add("Content-Type", "application/json")

		// 使用http.Client发送请求
		client := &http.Client{}
		response, err := client.Do(request)
		if err != nil {
			log.Println("Error sending request:", err)
			return err
		}
		defer response.Body.Close()

		bodyData, err := io.ReadAll(response.Body)
		if err != nil {
			log.Println(err)
			return err
		}

		log.Println(string(bodyData))

		var minimaxresp minimax.MinimaxResponse
		json.Unmarshal(bodyData, &minimaxresp)
		log.Println(minimaxresp)
		myresp := adapter.MinimaxResponseToOpenAIResponse(&minimaxresp)
		myresp.Model = oaiReq.Model
		log.Println("响应：", *myresp)

		respData, _ := json.Marshal(*myresp)
		log.Println("响应", string(respData))

		c.JSON(http.StatusOK, myresp)

		return nil
	}

	return nil
}
