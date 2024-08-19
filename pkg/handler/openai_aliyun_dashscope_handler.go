package handler

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"
	aliyun_dashscope_adapter "simple-one-api/pkg/adapter/aliyun-dashscope-adapter"
	"simple-one-api/pkg/llm/aliyun-dashscope/common_btype"
	"simple-one-api/pkg/llm/aliyun-dashscope/commsg/ds_com_resp"
	"simple-one-api/pkg/mylog"
	"simple-one-api/pkg/utils"
	"strings"
)

var dashscopeServerURL string = "https://dashscope.aliyuncs.com/api/v1/services/aigc/text-generation/generation"

// IsLlamaModel 判断给定的模型名称是否以 "llama" 开头
func IsLlamaModel(model string) bool {
	// 使用 HasPrefix 判断字符串是否以 "llama" 开头
	return strings.HasPrefix(model, "llama")
}

func IsBaiChuanModel(model string) bool {
	return strings.HasPrefix(model, "baichuan")
}

// 根据模型名称决定调用哪个服务
func getModelProtocalType(model string) string {
	// 定义模型与服务的映射
	modelServiceMap := map[string]string{
		"llama*":       "AA",
		"baichuan*":    "A",
		"chatglm*":     "A",
		"ziya-llama*":  "B",
		"dolly*":       "A",
		"belle-llama*": "B",
		"chatyuan*":    "B",
		"billa*":       "B",
		"yi*":          "A",
		"aquilachat*":  "A",
		"moss*":        "B",
		"deepseek*":    "A",
		"internlm*":    "A",
	}

	// 根据映射选择服务
	if service, exists := modelServiceMap[model]; exists {
		return service
	}

	return ""
}

func OpenAI2AliyunDashScopeHandler(c *gin.Context, oaiReqParam *OAIRequestParam) error {
	oaiReq := oaiReqParam.chatCompletionReq

	bType := getModelProtocalType(oaiReq.Model)

	if bType == "B" {
		llamaReq := aliyun_dashscope_adapter.OpenAIRequestToDashScopeBTypeRequest(oaiReq)

		var apiKey string
		reqJsonData, _ := json.Marshal(llamaReq)
		respJson, err := utils.SendHTTPRequest(apiKey, dashscopeServerURL, reqJsonData)
		if err != nil {
			mylog.Logger.Error("An error occurred", zap.Error(err))

			return err
		}

		var resp common_btype.DSBtypeResponseBody
		json.Unmarshal(respJson, &resp)

		oaiResp := aliyun_dashscope_adapter.DashScopeBTypeResponseToOpenAIResponse(&resp)
		//待完成

		mylog.Logger.Info("Standard response",
			zap.Any("response", *oaiResp)) // 记录响应对象

		c.JSON(http.StatusOK, oaiResp)
	} else if bType == "A" {
		if oaiReq.Stream {
			commReq := aliyun_dashscope_adapter.OpenAIRequestToDashScopeCommonRequest(oaiReq)
			var apiKey string
			reqJsonData, _ := json.Marshal(commReq)
			utils.SendSSERequest(apiKey, dashscopeServerURL, reqJsonData, func(data string) {
				_, err := c.Writer.WriteString("data: " + data + "\n\n")
				if err != nil {
					// 假设 mylog.Logger 是一个已经配置好的 zap.Logger 实例
					mylog.Logger.Error("An error occurred",
						zap.Error(err)) // 记录错误对象

					return
				}
				c.Writer.(http.Flusher).Flush()
			})

		} else {
			commReq := aliyun_dashscope_adapter.OpenAIRequestToDashScopeCommonRequest(oaiReq)
			var apiKey string
			reqJsonData, _ := json.Marshal(commReq)
			respJson, err := utils.SendHTTPRequest(apiKey, dashscopeServerURL, reqJsonData)
			if err != nil {
				mylog.Logger.Error("An error occurred", zap.Error(err))

				return err
			}

			var commResp ds_com_resp.ModelResponse
			json.Unmarshal(respJson, &commResp)

			oaiResp := aliyun_dashscope_adapter.DashScopeCommonResponseToOpenAIResponse(&commResp)
			//待完成

			mylog.Logger.Info("Standard response",
				zap.Any("response", *oaiResp)) // 记录响应对象

			c.JSON(http.StatusOK, oaiResp)

		}
	}

	return nil
}
