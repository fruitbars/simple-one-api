package handler

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
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

// 封装HTTP请求和错误处理
func sendOllamaJSONRequest(url string, payload []byte) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		mylog.Logger.Error("Error creating request", zap.Error(err))
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := http.DefaultClient // 使用全局的HTTP客户端
	resp, err := client.Do(req)
	if err != nil {
		mylog.Logger.Error("Error sending request", zap.Error(err))
		return nil, err
	}
	return resp, nil
}

// 错误和响应处理封装
func handleResponse(resp *http.Response) error {

	if resp.StatusCode != http.StatusOK {
		mylog.Logger.Info("HTTP Error Response", zap.String("status", resp.Status))
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			mylog.Logger.Error("Failed to read response body", zap.Error(err))
			return err
		}

		mylog.Logger.Info("Response body", zap.String("body", string(body)))
		return fmt.Errorf(string(body))
	}
	return nil
}

func OpenAI2OllamaHandler(c *gin.Context, oaiReqParam *OAIRequestParam) error {
	oaiReq := oaiReqParam.chatCompletionReq
	s := oaiReqParam.modelDetails
	//credentials := oaiReqParam.creds

	ollamaRequest := adapter.OpenAIRequestToOllamaRequest(oaiReq)
	return handleOllamaRequest(c, s, ollamaRequest, oaiReqParam)
}

func handleOllamaRequest(c *gin.Context, s *config.ModelDetails, ollamaRequest *ollama.ChatRequest, oaiReqParam *OAIRequestParam) error {
	jsonStr, err := json.Marshal(ollamaRequest)
	if err != nil {
		mylog.Logger.Error("Error marshaling JSON", zap.Error(err))
		return err
	}

	serverUrl := defaultOllamaUrl
	if s.ServerURL != "" {
		serverUrl = s.ServerURL
	}

	resp, err := sendOllamaJSONRequest(serverUrl, jsonStr)
	if err != nil {
		mylog.Logger.Error("err", zap.Error(err))
		return err
	}
	defer resp.Body.Close()
	err = handleResponse(resp)
	if err != nil {
		mylog.Logger.Error("err", zap.Error(err))
		return err
	}

	return processOllamaResponseBody(c, resp, ollamaRequest.Stream, oaiReqParam)
}

func processOllamaResponseBody(c *gin.Context, resp *http.Response, stream bool, oaiReqParam *OAIRequestParam) error {
	clientModel := oaiReqParam.ClientModel
	if stream {
		utils.SetEventStreamHeaders(c)
		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					mylog.Logger.Error("Error reading stream", zap.Error(err))
				}
				break
			}

			var ollamaStreamResp ollama.ChatResponse
			err = json.Unmarshal([]byte(line), &ollamaStreamResp)
			if err != nil {
				mylog.Logger.Error("An error occurred during unmarshal", zap.Error(err))
				return err
			}

			oaiRespStream := adapter.OllamaResponseToOpenAIStreamResponse(&ollamaStreamResp)
			oaiRespStream.Model = clientModel
			respData, err := json.Marshal(&oaiRespStream)
			if err != nil {
				mylog.Logger.Error("Error marshaling response", zap.Error(err))
				return err
			}

			_, err = c.Writer.WriteString("data: " + string(respData) + "\n\n")
			if err != nil {
				mylog.Logger.Error("Error writing response", zap.Error(err))
				return err
			}
			c.Writer.(http.Flusher).Flush()
		}
	} else {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			mylog.Logger.Error("Error reading response body", zap.Error(err))
			return err
		}

		var ollamaResp ollama.ChatResponse
		err = json.Unmarshal(body, &ollamaResp)
		if err != nil {
			mylog.Logger.Error("Error unmarshal response body", zap.Error(err))
			return err
		}

		myresp := adapter.OllamaResponseToOpenAIResponse(&ollamaResp)
		myresp.Model = clientModel
		c.JSON(http.StatusOK, myresp)
	}
	return nil
}
