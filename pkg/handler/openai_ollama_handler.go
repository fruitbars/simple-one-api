package handler

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"io"
	"net/http"
	"simple-one-api/pkg/adapter"
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/llm/ollama"
	"simple-one-api/pkg/mylog"
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

	// 假设 mylog.Logger 是一个已经配置好的 zap.Logger 实例
	mylog.Logger.Info("JSON String",
		zap.String("json_str", string(jsonStr))) // 记录 JSON 字符串

	// 创建POST请求
	req, err := http.NewRequest("POST", serverUrl, bytes.NewBuffer(jsonStr))
	if err != nil {
		// 假设 mylog.Logger 是一个已经配置好的 zap.Logger 实例
		mylog.Logger.Error("Error creating request",
			zap.Error(err)) // 记录错误对象

		return err
	}

	// 添加必要的请求头
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		// 假设 mylog.Logger 是一个已经配置好的 zap.Logger 实例
		mylog.Logger.Error("Error sending request",
			zap.Error(err)) // 记录错误对象

		return err
	}
	defer resp.Body.Close()

	// 检查HTTP响应状态码
	if resp.StatusCode != http.StatusOK {
		// 假设 mylog.Logger 是一个已经配置好的 zap.Logger 实例
		mylog.Logger.Info("HTTP Error Response",
			zap.String("status", resp.Status)) // 记录 HTTP 响应状态

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			// 假设 mylog.Logger 是一个已经配置好的 zap.Logger 实例
			mylog.Logger.Error("An error occurred",
				zap.Error(err)) // 记录错误对象

			return err
		}

		// 假设 mylog.Logger 是一个已经配置好的 zap.Logger 实例
		mylog.Logger.Info("Response body",
			zap.String("body", string(body))) // 记录响应体内容

		return fmt.Errorf(string(body))
	}
	//	mylog.Println(ollamaRequest.Stream)

	if ollamaRequest.Stream {
		utils.SetEventStreamHeaders(c)
		// 流式处理
		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					mylog.Logger.Error("Error reading stream",
						zap.Error(err)) // 记录错误对象

				}

				break
			}
			mylog.Logger.Info("ollama response",
				zap.String("line", line)) // 记录流响应行

			var ollamaStreamResp ollama.ChatResponse
			err = json.Unmarshal([]byte(line), &ollamaStreamResp)
			if err != nil {
				mylog.Logger.Error("An error occurred",
					zap.Error(err)) // 记录错误对象

			}

			oaiRespStream := adapter.OllamaResponseToOpenAIStreamResponse(&ollamaStreamResp)
			respData, err := json.Marshal(&oaiRespStream)
			if err != nil {
				mylog.Logger.Error("Error marshaling response",
					zap.Error(err)) // 记录错误对象

				return err
			}

			mylog.Logger.Info("Response data",
				zap.String("resp_data", string(respData))) // 记录响应数据

			_, err = c.Writer.WriteString("data: " + string(respData) + "\n\n")
			if err != nil {
				mylog.Logger.Error("An error occurred",
					zap.Error(err)) // 记录错误对象
				//return err
			}
			c.Writer.(http.Flusher).Flush()

		}
	} else {
		// 一次性读取完整响应
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			mylog.Logger.Error("Error reading response body",
				zap.Error(err)) // 记录错误对象
			return err
		}

		mylog.Logger.Info("Response",
			zap.String("body", string(body))) // 记录响应体内容

		var ollamaResp ollama.ChatResponse
		json.Unmarshal(body, &ollamaResp)

		myresp := adapter.OllamaResponseToOpenAIResponse(&ollamaResp)

		c.JSON(http.StatusOK, myresp)
		return nil
	}

	return nil
}
