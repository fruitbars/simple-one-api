package handler

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sashabaranov/go-openai"
	"io"
	"log"
	"net/http"
	"simple-one-api/pkg/adapter"
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/devplatform/cozecn"
	"simple-one-api/pkg/utils"
	"strings"
)

var defaultCozeUrl = "https://api.coze.cn/open_api/v2/chat"

func OpenAI2CozecnHander(c *gin.Context, s *config.ModelDetails, oaiReq openai.ChatCompletionRequest) error {
	secretToken := s.Credentials["api_key"]
	if secretToken == "" {
		secretToken = s.Credentials["token"]
	}

	cozecnReq := adapter.OpenAIRequestToCozecnRequest(oaiReq)

	cozeServerUrl := defaultCozeUrl
	if s.ServerURL != "" {
		cozeServerUrl = s.ServerURL
	}

	// 将请求数据编码为JSON格式
	jsonData, err := json.Marshal(cozecnReq)
	if err != nil {
		return fmt.Errorf("json编码错误: %v", err)
	}

	log.Println(string(jsonData))

	// 创建HTTP请求
	req, err := http.NewRequest("POST", cozeServerUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Println(err)
		return err
	}

	// 设置请求头
	req.Header.Set("Authorization", "Bearer "+secretToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Connection", "keep-alive")

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return err
	}
	defer resp.Body.Close()

	// 处理响应数据
	if oaiReq.Stream {
		return handleStreamResponse(c, oaiReq, resp.Body)
	} else {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Println(err)
			return err
		}

		log.Println(string(body))

		var respJson cozecn.Response
		json.Unmarshal(body, &respJson)
		myresp := adapter.CozecnReponseToOpenAIResponse(&respJson)

		myresp.Model = oaiReq.Model

		log.Println("响应：", *myresp)

		if respJson.Code != 0 {
			log.Println("错误信息：", respJson)
			return errors.New(respJson.Msg)
		}

		c.JSON(http.StatusOK, myresp)
	}

	return nil
}

func handleStreamResponse(c *gin.Context, oaiReq openai.ChatCompletionRequest, body io.Reader) error {
	scanner := bufio.NewScanner(body)
	utils.SetEventStreamHeaders(c)

	for scanner.Scan() {
		line := scanner.Text()
		//log.Println(line)
		if strings.HasPrefix(line, "data:") {
			log.Println(line)
			line = strings.TrimPrefix(line, "data:")
			var response cozecn.StreamResponse
			if err := json.Unmarshal([]byte(line), &response); err != nil {
				log.Println(err)
				return fmt.Errorf("解析响应数据错误: %v", err)
			}
			//log.Println(response)
			switch response.Event {
			case "message":
				if response.Message.Type == "verbose" {
					continue
				}
				oaiRespStream := adapter.CozecnReponseToOpenAIResponseStream(&response)
				oaiRespStream.Model = oaiReq.Model
				respData, err := json.Marshal(&oaiRespStream)
				if err != nil {
					log.Println(err)
					return err
				}

				log.Println(string(respData))
				_, err = c.Writer.WriteString("data: " + string(respData) + "\n\n")
				if err != nil {
					log.Println(err)
				}
				c.Writer.(http.Flusher).Flush()

			case "done":

				return nil
			case "error":
				log.Printf("Chat 错误结束: %s\n", response.ErrorInformation.Msg)
				return fmt.Errorf("错误码: %d, 错误信息: %s", response.ErrorInformation.Code, response.ErrorInformation.Msg)
			default:
				fmt.Printf("未知事件: %s\n", line)
				return errors.New("message error:" + line)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("读取流式响应数据错误: %v", err)
	}

	return nil
}
