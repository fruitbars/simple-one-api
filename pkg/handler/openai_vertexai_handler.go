package handler

import (
	"cloud.google.com/go/vertexai/genai"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"net/http"
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/mylog"
	"simple-one-api/pkg/utils"
	"time"
)

var vertexAIUrlTemplate = "https://%s-aiplatform.googleapis.com/v1beta1/projects/%s/locations/%s/endpoints/openapi"

func getVertexAIServerUrl(credentials map[string]interface{}) (string, string, string, error) {
	projectID, _ := utils.GetStringFromMap(credentials, config.KEYNAME_GCP_PROJECT_ID)
	location, _ := utils.GetStringFromMap(credentials, config.KEYNAME_GCP_LOCATION)
	//modelID, _ := utils.GetStringFromMap(credentials, config.KEYNAME_GCP_MODEL_ID)

	if projectID == "" || location == "" {
		return "", "", "", errors.New("projectID or location is empty")
	}

	serverURL := fmt.Sprintf(vertexAIUrlTemplate, location, projectID, location)
	return location, projectID, serverURL, nil
}

func OpenAI2VertexAIHandler(c *gin.Context, oaiReqParam *OAIRequestParam) error {
	req := oaiReqParam.chatCompletionReq
	//s := oaiReqParam.modelDetails
	credentials := oaiReqParam.creds

	var clientOption option.ClientOption
	customTransport := &utils.SimpleCustomTransport{
		Transport: http.DefaultTransport,
	}
	if oaiReqParam.httpTransport != nil {
		//client.Client.WithHttpTransport(oaiReqParam.httpTransport)
		customTransport.Transport = oaiReqParam.httpTransport
	}

	customHTTPClient := &http.Client{
		Transport: customTransport,
		Timeout:   30 * time.Second,
	}
	clientOption = option.WithHTTPClient(customHTTPClient)

	authJsonFile, _ := utils.GetStringFromMap(credentials, config.KEYNAME_GCP_JSON_FILE)

	authOption := option.WithCredentialsFile(authJsonFile)

	restOption := genai.WithREST()

	location, projectID, _, err := getVertexAIServerUrl(credentials)
	if err != nil {
		return err
	}

	mylog.Logger.Debug("OpenAI2VertexAIHandler", zap.String("projectID", projectID), zap.String("location", location),
		zap.Any("authOption", authOption), zap.Any("restOption", restOption))

	ctx := context.Background()
	client, err := genai.NewClient(ctx, projectID, location, clientOption, authOption, restOption)
	//client, err := genai.NewClient(ctx, projectID, location, authOption, restOption)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	mylog.Logger.Debug("genai.NewClien", zap.Any("client", client))
	modelCaller := client.GenerativeModel(req.Model)

	img := genai.FileData{
		MIMEType: "image/jpeg",
		FileURI:  "gs://generativeai-downloads/images/scones.jpg",
	}
	prompt := genai.Text("What is in this image?")

	if req.Stream {
		iter := modelCaller.GenerateContentStream(
			ctx,
			genai.FileData{
				MIMEType: "video/mp4",
				FileURI:  "gs://cloud-samples-data/generative-ai/video/animals.mp4",
			},
			genai.FileData{
				MIMEType: "video/jpeg",
				FileURI:  "gs://cloud-samples-data/generative-ai/image/character.jpg",
			},
			genai.Text("Are these video and image correlated?"),
		)
		for {
			resp, err := iter.Next()
			if err == iterator.Done {
				mylog.Logger.Error("Done")
				return nil
			}
			if err != nil {
				mylog.Logger.Error("iter.Next", zap.Error(err))
				return err
			}

			mylog.Logger.Info("iter.Next", zap.Any("resp", resp))

			if resp != nil && (len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0) {
				return errors.New("empty response from model")
			}

			fmt.Println("generated response: ")
			for _, c := range resp.Candidates {
				for _, p := range c.Content.Parts {
					fmt.Printf("%s ", p)
				}
			}
		}
	} else {
		resp, err := modelCaller.GenerateContent(ctx, img, prompt)
		if err != nil {
			return fmt.Errorf("error generating content: %w", err)
		}
		mylog.Logger.Debug("modelCaller.GenerateContent", zap.Any("resp", resp))
		rb, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			return fmt.Errorf("json.MarshalIndent: %w", err)
		}
		fmt.Println(string(rb))
		return nil
	}
}
