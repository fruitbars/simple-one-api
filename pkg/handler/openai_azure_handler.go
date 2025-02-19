package handler

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/sdk/ai/azopenai"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"net/url"
	"simple-one-api/pkg/adapter"
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/utils"
)

func formatAzureURL(inputURL string) (string, error) {
	// 解析URL
	parsedURL, err := url.Parse(inputURL)
	if err != nil {
		return "", err
	}

	// 构建新的URL
	formattedURL := &url.URL{
		Scheme: parsedURL.Scheme,
		Host:   parsedURL.Host,
	}

	return formattedURL.String(), nil
}

// OpenAI2AzureOpenAIHandler handles OpenAI to Azure OpenAI requests
func OpenAI2AzureOpenAIHandler(c *gin.Context, oaiReqParam *OAIRequestParam) error {
	req := oaiReqParam.chatCompletionReq
	s := oaiReqParam.modelDetails

	apiKey, _ := utils.GetStringFromMap(s.Credentials, config.KEYNAME_API_KEY)
	serverURL, err := formatAzureURL(s.ServerURL)
	if err != nil {
		serverURL = s.ServerURL
	}

	clientModel := oaiReqParam.ClientModel

	log.Println(req, apiKey, serverURL, clientModel)

	keyCredential := azcore.NewKeyCredential(apiKey)
	client, err := azopenai.NewClientWithKeyCredential(serverURL, keyCredential, nil)

	azureReq := adapter.OpenAIRequestToAzureRequest(req)

	resp, err := client.GetChatCompletions(context.TODO(), *azureReq, nil)

	myresp := adapter.AzureResponseToOpenAIResponse(&resp)

	c.JSON(http.StatusOK, myresp)

	return nil
}
