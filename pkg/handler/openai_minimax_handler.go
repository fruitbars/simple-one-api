package handler

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"io"
	"log"
	"net/http"
	"simple-one-api/pkg/adapter"
	mycommon "simple-one-api/pkg/common"
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/llm/minimax"
	"simple-one-api/pkg/openai"
	"strings"
)

func OpenAI2MinimaxHander(c *gin.Context, s *config.ModelDetails, oaiReq openai.OpenAIRequest) {
	apiKey := s.Credentials["api_key"]
	groupID := s.Credentials["group_id"]

	if s.ServerURL == "" {
		//serverUrl = defaultUrl
		s.ServerURL = "https://api.minimax.chat/v1/text/chatcompletion_pro"
	}

	serverUrl := fmt.Sprintf("%s?GroupId=%s", s.ServerURL, groupID)
	bearerToken := fmt.Sprintf("Bearer %s", apiKey)

	if oaiReq.Stream != nil && *oaiReq.Stream {
		minimaxReq := adapter.OpenAIRequestToMinimaxRequest(oaiReq)

		jsonData, err := json.Marshal(minimaxReq)
		if err != nil {
			log.Println("Error marshalling JSON:", err)
			return
		}

		log.Println(string(jsonData))

		request, err := http.NewRequest("POST", serverUrl, bytes.NewBuffer(jsonData))
		if err != nil {
			log.Println("Error creating request:", err)
			return
		}

		request.Header.Add("Authorization", bearerToken)
		request.Header.Add("Content-Type", "application/json")

		// 使用http.Client发送请求
		client := &http.Client{}
		response, err := client.Do(request)
		if err != nil {
			log.Println("Error sending request:", err)
			return
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
				return
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

			respData, err := json.Marshal(&oaiRespStream)
			if err != nil {
				log.Println(err)
			} else {
				log.Println("response http data", string(respData))

				if oaiRespStream.Error != nil {

					c.JSON(http.StatusUnauthorized, oaiRespStream)
				} else {
					c.Writer.WriteString("data: " + string(respData) + "\n\n")
					c.Writer.(http.Flusher).Flush()
				}

			}

		}
	}
}
