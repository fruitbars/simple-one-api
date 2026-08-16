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
	"simple-one-api/pkg/mycommon"
	"simple-one-api/pkg/mylog"
	"simple-one-api/pkg/utils"
	"strings"
	"time"
)

func OpenAI2MinimaxHandler(c *gin.Context, oaiReqParam *OAIRequestParam) error {
	oaiReq := oaiReqParam.chatCompletionReq
	s := oaiReqParam.modelDetails
	credentials := oaiReqParam.creds

	apiKey := credentials[config.KEYNAME_API_KEY]
	groupID := credentials[config.KEYNAME_GROUP_ID]

	baseURL := s.ServerURL
	if baseURL == "" {
		baseURL = "https://api.minimax.chat/v1/text/chatcompletion_pro"
	}

	serverUrl := fmt.Sprintf("%s?GroupId=%s", baseURL, groupID)
	bearerToken := fmt.Sprintf("Bearer %s", apiKey)

	minimaxReq := adapter.OpenAIRequestToMinimaxRequest(oaiReq)

	jsonData, err := json.Marshal(minimaxReq)
	if err != nil {
		mylog.Logger.Error(err.Error())
		return err
	}

	mylog.Logger.Info(string(jsonData))

	if oaiReq.Stream {

		request, err := http.NewRequestWithContext(oaiReqParam.ctx, http.MethodPost, serverUrl, bytes.NewReader(jsonData))
		if err != nil {
			mylog.Logger.Error(err.Error())
			return err
		}

		request.Header.Add("Authorization", bearerToken)
		request.Header.Add("Content-Type", "application/json")

		client := utils.NewHTTPClient(oaiReqParam.httpTransport, 0)

		response, err := client.Do(request)
		if err != nil {
			mylog.Logger.Error(err.Error())
			return err
		}
		defer response.Body.Close()
		if err := mycommon.CheckStatusCode(response); err != nil {
			return err
		}

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
		request, err := http.NewRequestWithContext(oaiReqParam.ctx, http.MethodPost, serverUrl, bytes.NewReader(jsonData))
		if err != nil {
			mylog.Logger.Error(err.Error())
			return err
		}

		request.Header.Add("Authorization", bearerToken)
		request.Header.Add("Content-Type", "application/json")

		client := utils.NewHTTPClient(oaiReqParam.httpTransport, 120*time.Second)
		response, err := client.Do(request)
		if err != nil {
			mylog.Logger.Error(err.Error())

			return err
		}
		defer response.Body.Close()
		if err := mycommon.CheckStatusCode(response); err != nil {
			return err
		}

		bodyData, err := io.ReadAll(response.Body)
		if err != nil {
			mylog.Logger.Error(err.Error())
			return err
		}

		var minimaxresp minimax.MinimaxResponse
		json.Unmarshal(bodyData, &minimaxresp)
		//mylog.Logger.Info((minimaxresp)
		myresp := adapter.MinimaxResponseToOpenAIResponse(&minimaxresp)
		myresp.Model = oaiReqParam.ClientModel

		c.JSON(http.StatusOK, myresp)

		return nil
	}

	return nil
}
