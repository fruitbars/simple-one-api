package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/sashabaranov/go-openai"
)

// https://console.groq.com/docs/openai
func adjustGroqReq(req *openai.ChatCompletionRequest) {
	req.LogProbs = false
	req.LogitBias = nil
	req.TopLogProbs = 0
	if req.N != 0 {
		req.N = 1
	}

	if req.Temperature <= 0 {
		req.Temperature = 0.1
	}

	if req.Temperature > 2 {
		req.Temperature = 2
	}
}

// OpenAI2GroqOpenAIHandler handles OpenAI to Azure OpenAI requests
func OpenAI2GroqOpenAIHandler(c *gin.Context, oaiReqParam *OAIRequestParam) error {
	req := oaiReqParam.chatCompletionReq
	s := oaiReqParam.modelDetails
	//credentials := oaiReqParam.creds
	conf, err := getConfig(s, oaiReqParam)
	if err != nil {
		return err
	}

	adjustGroqReq(req)

	clientModel := oaiReqParam.ClientModel
	return handleOpenAIOpenAIRequest(conf, c, req, clientModel)
}
