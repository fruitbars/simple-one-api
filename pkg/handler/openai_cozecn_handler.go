package handler

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"io"
	"net/http"
	"simple-one-api/pkg/adapter"
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/llm/devplatform/cozecn"
	"simple-one-api/pkg/llm/devplatform/cozecn_v3/nonestream"
	"simple-one-api/pkg/llm/devplatform/cozecn_v3/streammode"
	"simple-one-api/pkg/mycommon"
	"simple-one-api/pkg/mylog"
	"simple-one-api/pkg/utils"
	"strings"
	"time"
)

var defaultCozecnV2URL = "https://api.coze.cn/open_api/v2/chat"
var defaultCozecomV2URL = "https://api.coze.com/open_api/v2/chat"

func getSecretToken(credentials map[string]interface{}, model string) string {
	//credentials := mycommon.GetACredentials(s, model)
	// 使用统一的api_key获取
	secretToken, _ := utils.GetStringFromMap(credentials, config.KEYNAME_API_KEY)
	if secretToken == "" {
		secretToken, _ = utils.GetStringFromMap(credentials, config.KEYNAME_TOKEN)
	}

	return secretToken
}

func OpenAI2CozecnHandler(c *gin.Context, oaiReqParam *OAIRequestParam) error {
	oaiReq := oaiReqParam.chatCompletionReq
	s := oaiReqParam.modelDetails
	credentials := oaiReqParam.creds

	secretToken := getSecretToken(credentials, oaiReq.Model)
	cozecnReq := adapter.OpenAIRequestToCozecnRequest(oaiReq)
	cozeServerURL := s.ServerURL

	apiVersion := "v3"
	if strings.Contains(cozeServerURL, "v2/chat") {
		apiVersion = "v2"
	}

	mylog.Logger.Info("apiVersion", zap.String("apiVersion", apiVersion))

	if apiVersion == "v2" {
		if cozeServerURL == "" {
			switch s.ServiceName {
			case "cozecn":
				cozeServerURL = defaultCozecnV2URL
			case "cozecom":
				cozeServerURL = defaultCozecomV2URL
			default:
				cozeServerURL = defaultCozecnV2URL
			}
		}
		client := &http.Client{
			Timeout: 3 * time.Minute,
		}
		if oaiReqParam.httpTransport != nil {
			client.Transport = oaiReqParam.httpTransport
		}

		mylog.Logger.Info(cozeServerURL)
		mylog.Logger.Info("oaiReq", zap.Any("oaiReq", oaiReq))
		mylog.Logger.Info("cozecnReq", zap.Any("cozecnReq", cozecnReq))
		// 使用统一的错误处理函数
		if err := sendRequest(c, client, secretToken, cozeServerURL, cozecnReq, oaiReq, oaiReqParam); err != nil {
			mylog.Logger.Error(err.Error(), zap.String("cozeServerURL", cozeServerURL),
				zap.Any("cozecnReq", cozecnReq), zap.Any("oaiReq", oaiReq))
			return err
		}

	} else {
		cozeChatReq := adapter.OpenAIRequestToCozecnV3Request(oaiReq)
		mylog.Logger.Info("cozeChatReq", zap.Any("cozeChatReq", cozeChatReq))
		if oaiReq.Stream == false {

			cozeChatResp, err := nonestream.ChatWithNoneStream(secretToken, cozeChatReq, oaiReqParam.httpTransport, int(3*time.Minute))
			if err != nil {
				mylog.Logger.Error(err.Error())
				return err
			}

			oaiResp := adapter.CozecnV3ReponseToOpenAIResponse(cozeChatResp)
			oaiResp.Model = oaiReqParam.ClientModel

			c.JSON(http.StatusOK, oaiResp)
		} else {
			cb := func(event, data string) {
				mylog.Logger.Info("event", zap.String("event", event), zap.String("data", data))

				if event == "conversation.message.delta" || event == "conversation.chat.completed" {
					var resp streammode.EventData
					err := json.Unmarshal([]byte(data), &resp)
					if err != nil {
						mylog.Logger.Error(err.Error())
						return
					}

					oaiStreamResp := adapter.CozecnV3ReponseToOpenAIResponseStream(&resp)
					oaiStreamResp.Model = oaiReqParam.ClientModel
					respData, err := json.Marshal(oaiStreamResp)
					if err != nil {
						mylog.Logger.Error(err.Error())
					}

					mylog.Logger.Info(string(respData))

					if _, err := c.Writer.WriteString("data: " + string(respData) + "\n\n"); err != nil {
						mylog.Logger.Warn(err.Error())
					}
					c.Writer.(http.Flusher).Flush()
				}
			}
			err := streammode.Chat(secretToken, cozeChatReq, cb, oaiReqParam.httpTransport)
			if err != nil {
				mylog.Logger.Error(err.Error())
				return err
			}
		}

	}

	return nil
}

func sendRequest(c *gin.Context, client *http.Client, token, url string, request interface{}, oaiReq *openai.ChatCompletionRequest, oaiReqParam *OAIRequestParam) error {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("json编码错误: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		mylog.Logger.Error(err.Error())
		return err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		mylog.Logger.Error(err.Error())
		return err
	}
	defer resp.Body.Close()

	err = mycommon.CheckStatusCode(resp)
	if err != nil {

		return err
	}

	return handleCozecnResponse(c, resp, oaiReq, oaiReqParam)
}

func handleCozecnResponse(c *gin.Context, resp *http.Response, oaiReq *openai.ChatCompletionRequest, oaiReqParam *OAIRequestParam) error {
	if oaiReq.Stream {
		return handleCozecnStreamResponse(c, oaiReq, resp.Body, oaiReqParam)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		mylog.Logger.Error(err.Error())
		return err
	}

	mylog.Logger.Info("response", zap.String("body", string(body)))

	var respJson cozecn.Response
	if err := json.Unmarshal(body, &respJson); err != nil {
		mylog.Logger.Error(err.Error())
		return fmt.Errorf("json解码错误: %v", err)
	}

	if respJson.Code != 0 {
		return fmt.Errorf("错误码: %d, 错误信息: %s", respJson.Code, respJson.Msg)
	}

	myresp := adapter.CozecnReponseToOpenAIResponse(&respJson)
	myresp.Model = oaiReqParam.ClientModel
	c.JSON(http.StatusOK, myresp)

	return nil
}

func handleCozecnStreamResponse(c *gin.Context, oaiReq *openai.ChatCompletionRequest, body io.Reader, oaiReqParam *OAIRequestParam) error {
	scanner := bufio.NewScanner(body)
	utils.SetEventStreamHeaders(c)

	for scanner.Scan() {
		line := scanner.Text()
		//log.Println(line)
		if strings.HasPrefix(line, "data:") {
			mylog.Logger.Info(line)
			line = strings.TrimPrefix(line, "data:")
			var response cozecn.StreamResponse
			if err := json.Unmarshal([]byte(line), &response); err != nil {
				mylog.Logger.Error(err.Error())
				return fmt.Errorf("解析响应数据错误: %v", err)
			}
			//log.Println(response)
			switch response.Event {
			case "message":
				if response.Message.Type == "verbose" {
					continue
				}
				oaiRespStream := adapter.CozecnReponseToOpenAIResponseStream(&response)
				oaiRespStream.Model = oaiReqParam.ClientModel
				respData, err := json.Marshal(&oaiRespStream)
				if err != nil {
					mylog.Logger.Error(err.Error())
					return err
				}

				mylog.Logger.Info(string(respData))
				_, err = c.Writer.WriteString("data: " + string(respData) + "\n\n")
				if err != nil {
					mylog.Logger.Error(err.Error())
				}
				c.Writer.(http.Flusher).Flush()

			case "done":

				return nil
			case "error":
				mylog.Logger.Error(response.ErrorInformation.Msg)
				return fmt.Errorf("错误码: %d, 错误信息: %s", response.ErrorInformation.Code, response.ErrorInformation.Msg)
			default:
				fmt.Printf("未知事件: %s\n", line)
				return errors.New("message error:" + line)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		mylog.Logger.Error(err.Error())
		return fmt.Errorf("读取流式响应数据错误: %v", err)
	}

	return nil
}
