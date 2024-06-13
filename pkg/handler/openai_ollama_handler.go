package handler

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sashabaranov/go-openai"
	"io"
	"log"
	"net/http"
	"simple-one-api/pkg/adapter"
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/llm/ollama"
	"simple-one-api/pkg/utils"
)

// 设置目标URL
var defaultOllamaUrl = "http://127.0.0.1:11434/api/chat"

func OpenAI2OllamaHandler(c *gin.Context, s *config.ModelDetails, oaiReq openai.ChatCompletionRequest) error {
	ollamaRequest := adapter.OpenAIRequestToOllamaRequest(oaiReq)

	return handleOllamaRequest(c, s, ollamaRequest)
}

func handleOllamaRequest(c *gin.Context, s *config.ModelDetails, ollamaRequest *ollama.ChatRequest) error {

	serverUrl := defaultOllamaUrl
	if s.ServerURL != "" {
		serverUrl = s.ServerURL
	}

	jsonStr, err := json.Marshal(ollamaRequest)

	log.Println(string(jsonStr))

	// 创建POST请求
	req, err := http.NewRequest("POST", serverUrl, bytes.NewBuffer(jsonStr))
	if err != nil {
		log.Println("Error creating request: ", err)
		return err
	}

	// 添加必要的请求头
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error sending request: ", err)
		return err
	}
	defer resp.Body.Close()

	// 检查HTTP响应状态码
	if resp.StatusCode != http.StatusOK {
		log.Println("HTTP Error Response: ", resp.Status)
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Println(err)
			return err
		}

		log.Println(string(body))

		return fmt.Errorf(string(body))
	}
	//	log.Println(ollamaRequest.Stream)

	if ollamaRequest.Stream {
		utils.SetEventStreamHeaders(c)
		// 流式处理
		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					log.Println("Error reading stream: ", err)
				}

				break
			}
			log.Println("ollama response", line)

			var ollamaStreamResp ollama.ChatResponse
			err = json.Unmarshal([]byte(line), &ollamaStreamResp)
			if err != nil {
				log.Println(err)
			}

			oaiRespStream := adapter.OllamaResponseToOpenAIStreamResponse(&ollamaStreamResp)
			respData, err := json.Marshal(&oaiRespStream)
			if err != nil {
				log.Println("Error marshaling response:", err)
				return err
			}

			log.Println(string(respData))

			_, err = c.Writer.WriteString("data: " + string(respData) + "\n\n")
			if err != nil {
				log.Println(err)
				return err
			}
			c.Writer.(http.Flusher).Flush()

		}
	} else {
		// 一次性读取完整响应
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Println("Error reading response body: ", err)
			return err
		}
		log.Println("Response:", string(body))

		var ollamaResp ollama.ChatResponse
		json.Unmarshal(body, &ollamaResp)

		myresp := adapter.OllamaResponseToOpenAIResponse(&ollamaResp)

		c.JSON(http.StatusOK, myresp)
		return nil
	}

	return nil
}
