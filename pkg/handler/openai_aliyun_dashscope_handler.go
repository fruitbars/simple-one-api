package handler

import (
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"
	"regexp"
	aliyun_dashscope_adapter "simple-one-api/pkg/adapter/aliyun-dashscope-adapter"
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/llm/aliyun-dashscope/common_btype"
	"simple-one-api/pkg/llm/aliyun-dashscope/commsg/ds_com_resp"
	"simple-one-api/pkg/mylog"
	"simple-one-api/pkg/utils"
)

var dashscopeServerURL string = "https://dashscope.aliyuncs.com/api/v1/services/aigc/text-generation/generation"

// 根据模型名称决定调用哪个服务
// A表示common普通类型
// B表示
func getModelProtocolType(model string) (string, error) {
	// 定义模型与服务的映射
	modelServiceMap := map[string]string{
		"llama.*":       "A",
		"baichuan.*":    "A", //baichuan-7b-v1已经验证
		"chatglm.*":     "A", //chatglm3-6b已经验证
		"ziya-llama.*":  "B",
		"dolly.*":       "A",
		"belle-llama.*": "B",
		"chatyuan.*":    "B",
		"billa.*":       "B",
		"yi.*":          "A", //yi-6b-chat,yi-34b-chat已经验证
		"aquilachat.*":  "A", //aquilachat-7b已经验证
		"moss.*":        "B",
		"deepseek.*":    "A", //deepseek-7b-chat已经验证
		"internlm.*":    "A", //internlm-7b-chat已经验证
		"qwen.*":        "A", //internlm-7b-chat已经验证
	}

	// 根据映射选择服务，使用正则表达式来支持通配符
	for pattern, service := range modelServiceMap {
		matched, _ := regexp.MatchString(pattern, model)
		if matched {
			return service, nil
		}
	}

	return "", errors.New("not support type")
}

func OpenAI2AliyunDashScopeHandler(c *gin.Context, oaiReqParam *OAIRequestParam) error {

	oaiReq := oaiReqParam.chatCompletionReq

	bType, err := getModelProtocolType(oaiReq.Model)
	if err != nil {
		mylog.Logger.Error("OpenAI2AliyunDashScopeHandler|getModelProtocolType", zap.Error(err))

		return err
	}

	credentials := oaiReqParam.creds
	apiKey, _ := utils.GetStringFromMap(credentials, config.KEYNAME_API_KEY)

	clientModel := oaiReqParam.ClientModel

	mylog.Logger.Info("OpenAI2AliyunDashScopeHandler", zap.Any("oaiReq", oaiReq), zap.String("bType", bType))

	if bType == "B" {
		llamaReq := aliyun_dashscope_adapter.OpenAIRequestToDashScopeBTypeRequest(oaiReq)

		reqJsonData, _ := json.Marshal(llamaReq)
		respJson, err := utils.SendHTTPRequest(apiKey, dashscopeServerURL, reqJsonData, oaiReqParam.httpTransport)
		if err != nil {
			mylog.Logger.Error("An error occurred", zap.Error(err))

			return err
		}

		var resp common_btype.DSBtypeResponseBody
		json.Unmarshal(respJson, &resp)

		if oaiReq.Stream {
			utils.SetEventStreamHeaders(c)
			oaiRespStream := aliyun_dashscope_adapter.DashScopeBTypeResponseToOpenAIStreamResponse(&resp)

			oaiRespStream.Model = clientModel
			respData, err := json.Marshal(&oaiRespStream)
			if err != nil {
				mylog.Logger.Error("Error marshaling response:", zap.Error(err))
				return err
			}

			// 假设 mylog.Logger 是一个已经配置好的 zap.Logger 实例
			mylog.Logger.Info("Response HTTP data",
				zap.String("data", string(respData))) // 记录响应数据

			if oaiRespStream.Error != nil {
				// 假设 mylog.Logger 是一个已经配置好的 zap.Logger 实例
				mylog.Logger.Error("Error response",
					zap.Any("error", *oaiRespStream.Error)) // 记录错误对象

				return err
			}

			_, err = c.Writer.WriteString("data: " + string(respData) + "\n\n")
			if err != nil {
				// 假设 mylog.Logger 是一个已经配置好的 zap.Logger 实例
				mylog.Logger.Error("An error occurred",
					zap.Error(err)) // 记录错误对象

				return err
			}
			c.Writer.(http.Flusher).Flush()
		} else {
			oaiResp := aliyun_dashscope_adapter.DashScopeBTypeResponseToOpenAIResponse(&resp)
			oaiResp.Model = clientModel
			//待完成

			mylog.Logger.Info("Standard response",
				zap.Any("response", *oaiResp)) // 记录响应对象

			c.JSON(http.StatusOK, oaiResp)
		}

	} else if bType == "A" {
		if oaiReq.Stream {
			utils.SetEventStreamHeaders(c)
			commReq := aliyun_dashscope_adapter.OpenAIRequestToDashScopeCommonRequest(oaiReq)
			mylog.Logger.Info("OpenAI2AliyunDashScopeHandler", zap.Any("commReq", commReq))

			reqJsonData, _ := json.Marshal(commReq)

			var dsLastestStreamResp *ds_com_resp.ModelStreamResponse
			err := utils.SendSSERequest(apiKey, dashscopeServerURL, reqJsonData, func(data string) {
				mylog.Logger.Debug("OpenAI2AliyunDashScopeHandler|utils.SendSSERequest", zap.String("data", data))

				var dsResp ds_com_resp.ModelStreamResponse
				json.Unmarshal([]byte(data), &dsResp)

				prevContent := aliyun_dashscope_adapter.GetStreamResponseContent(dsLastestStreamResp)
				oaiStreamResp := aliyun_dashscope_adapter.DashScopeCommonResponseToOpenAIStreamResponse(&dsResp, prevContent)

				mylog.Logger.Debug("OpenAI2AliyunDashScopeHandler|utils.SendSSERequest", zap.Any("oaiStreamResp", oaiStreamResp))

				dsLastestStreamResp = &dsResp

				oaiStreamResp.Model = clientModel
				respJsonData, _ := json.Marshal(oaiStreamResp)

				_, err := c.Writer.WriteString("data: " + string(respJsonData) + "\n\n")
				if err != nil {
					// 假设 mylog.Logger 是一个已经配置好的 zap.Logger 实例
					mylog.Logger.Error("An error occurred", zap.Error(err)) // 记录错误对象

					return
				}
				c.Writer.(http.Flusher).Flush()
			}, oaiReqParam.httpTransport)

			if err != nil {
				mylog.Logger.Error("OpenAI2AliyunDashScopeHandler|utils.SendSSERequest", zap.Error(err))

				return err
			}

		} else {
			commReq := aliyun_dashscope_adapter.OpenAIRequestToDashScopeCommonRequest(oaiReq)
			reqJsonData, _ := json.Marshal(commReq)
			respJson, err := utils.SendHTTPRequest(apiKey, dashscopeServerURL, reqJsonData, oaiReqParam.httpTransport)
			if err != nil {
				mylog.Logger.Error("An error occurred", zap.Error(err))

				return err
			}

			var commResp ds_com_resp.ModelResponse
			json.Unmarshal(respJson, &commResp)

			oaiResp := aliyun_dashscope_adapter.DashScopeCommonResponseToOpenAIResponse(&commResp)
			//待完成
			oaiResp.Model = clientModel

			mylog.Logger.Info("Standard response",
				zap.Any("response", *oaiResp)) // 记录响应对象

			c.JSON(http.StatusOK, oaiResp)

		}
	}

	return nil
}
