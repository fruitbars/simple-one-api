package handler

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"io"
	"net/http"
	"simple-one-api/pkg/adapter"
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/llm/minimax"
	"simple-one-api/pkg/mylog"
	"simple-one-api/pkg/utils"
	"strings"
)

func OpenAI2MinimaxHandler(c *gin.Context, oaiReqParam *OAIRequestParam) error {
	oaiReq := oaiReqParam.chatCompletionReq
	s := oaiReqParam.modelDetails
	credentials := oaiReqParam.creds

	apiKey := credentials[config.KEYNAME_API_KEY]
	groupID := credentials[config.KEYNAME_GROUP_ID]

	if s.ServerURL == "" {
		//serverUrl = defaultUrl
		s.ServerURL = "https://api.minimax.chat/v1/text/chatcompletion_pro"
	}

	serverUrl := fmt.Sprintf("%s?GroupId=%s", s.ServerURL, groupID)
	bearerToken := fmt.Sprintf("Bearer %s", apiKey)

	minimaxReq := adapter.OpenAIRequestToMinimaxRequest(oaiReq)

	jsonData, err := json.Marshal(minimaxReq)
	if err != nil {
		mylog.Logger.Error(err.Error())
		return err
	}

	mylog.Logger.Info(string(jsonData))

	if oaiReq.Stream {

		request, err := http.NewRequest("POST", serverUrl, bytes.NewBuffer(jsonData))
		if err != nil {
			mylog.Logger.Error(err.Error())
			return err
		}

		request.Header.Add("Authorization", bearerToken)
		request.Header.Add("Content-Type", "application/json")

		// 使用http.Client发送请求
		client := &http.Client{}
		if oaiReqParam.httpTransport != nil {
			client.Transport = oaiReqParam.httpTransport
		}

		response, err := client.Do(request)
		if err != nil {
			mylog.Logger.Error(err.Error())
			return err
		}
		defer response.Body.Close()

		id := uuid.New()
		utils.SetEventStreamHeaders(c)
		// 处理SSE响应
		reader := bufio.NewReader(response.Body)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					break
				}

				mylog.Logger.Error(err.Error())
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

			mylog.Logger.Info(line)

			var minimaxresp minimax.MinimaxResponse
			json.Unmarshal([]byte(line), &minimaxresp)

			oaiRespStream := adapter.MinimaxResponseToOpenAIStreamResponse(&minimaxresp)
			oaiRespStream.ID = id.String()
			oaiRespStream.Model = oaiReqParam.ClientModel
			respData, err := json.Marshal(&oaiRespStream)
			if err != nil {
				mylog.Logger.Error(err.Error())
				return err
			} else {
				mylog.Logger.Info(string(respData))

				if oaiRespStream.Error != nil {
					mylog.Logger.Info(oaiRespStream.Error.Message)
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
			mylog.Logger.Error(err.Error())
			return err
		}

		request.Header.Add("Authorization", bearerToken)
		request.Header.Add("Content-Type", "application/json")

		// 使用http.Client发送请求
		client := &http.Client{}
		response, err := client.Do(request)
		if err != nil {
			mylog.Logger.Error(err.Error())

			return err
		}
		defer response.Body.Close()

		bodyData, err := io.ReadAll(response.Body)
		if err != nil {
			mylog.Logger.Error(err.Error())
			return err
		}

		mylog.Logger.Info(string(bodyData))

		var minimaxresp minimax.MinimaxResponse
		json.Unmarshal(bodyData, &minimaxresp)
		//mylog.Logger.Info((minimaxresp)
		myresp := adapter.MinimaxResponseToOpenAIResponse(&minimaxresp)
		myresp.Model = oaiReqParam.ClientModel

		respData, _ := json.Marshal(*myresp)
		mylog.Logger.Info(string(respData))

		c.JSON(http.StatusOK, myresp)

		return nil
	}

	return nil
}
