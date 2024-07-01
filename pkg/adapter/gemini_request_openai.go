package adapter

import (
	"encoding/json"
	"fmt"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	googlegemini "simple-one-api/pkg/llm/google-gemini"
	"simple-one-api/pkg/mycomdef"
	"simple-one-api/pkg/mycommon"
	"simple-one-api/pkg/mylog"
	"strings"
)

func DeepCopyGeminiRequest(original *googlegemini.GeminiRequest) (*googlegemini.GeminiRequest, error) {
	originalJSON, err := json.Marshal(original)
	if err != nil {
		return nil, err
	}

	var copyReq googlegemini.GeminiRequest
	err = json.Unmarshal(originalJSON, &copyReq)
	if err != nil {
		return nil, err
	}

	for i, _ := range copyReq.Contents {
		for j, _ := range copyReq.Contents[i].Parts {
			if copyReq.Contents[i].Parts[j].InlineData != nil {
				d := "..."
				copyReq.Contents[i].Parts[j].InlineData.Data = d
			}

		}

	}

	return &copyReq, nil
}

// OpenAIRequestToGeminiRequest converts OpenAI chat completion request to a Gemini request.
func OpenAIRequestToGeminiRequest(oaiReq *openai.ChatCompletionRequest) *googlegemini.GeminiRequest {
	contents := convertMessagesToContents(oaiReq.Messages)

	//mylog.Logger.Debug("convertMessagesToContents", zap.Any("contents", contents))

	geminiReq := &googlegemini.GeminiRequest{
		Contents: contents,
		//	SafetySettings: []googlegemini.SafetySetting{},
		GenerationConfig: googlegemini.GenerationConfig{
			StopSequences:   oaiReq.Stop,
			Temperature:     oaiReq.Temperature,
			MaxOutputTokens: oaiReq.MaxTokens,
			TopP:            oaiReq.TopP,
			TopK:            oaiReq.TopLogProbs,
		},
	}

	return geminiReq
}

// convertMessagesToContents converts messages from OpenAI format to Gemini content entities.
func convertMessagesToContents(messages []openai.ChatCompletionMessage) []googlegemini.ContentEntity {
	var contents []googlegemini.ContentEntity
	hisMessages := mycommon.ConvertSystemMessages2NoSystem(messages)

	//mylog.Logger.Debug("ConvertSystemMessages2NoSystem", zap.Any("hisMessages", hisMessages))

	for _, msg := range hisMessages {
		role := getRole(msg)
		contentEntity := createContentEntity(msg, role)
		if contentEntity != nil {
			contents = append(contents, *contentEntity)
		}
	}

	return contents
}

// getRole determines the role for the message.
func getRole(msg openai.ChatCompletionMessage) string {
	if strings.ToLower(msg.Role) == mycomdef.KEYNAME_ASSISTANT {
		return mycomdef.KEYNAME_MODEL
	}
	return msg.Role
}

// createContentEntity creates a Gemini content entity from an OpenAI message.
func createContentEntity(msg openai.ChatCompletionMessage, role string) *googlegemini.ContentEntity {
	if len(msg.Content) > 0 {
		return &googlegemini.ContentEntity{
			Role:  role,
			Parts: []googlegemini.Part{{Text: msg.Content}},
		}
	}

	return createMultiPartContentEntity(msg, role)
}

// createMultiPartContentEntity creates a content entity with multiple parts.
func createMultiPartContentEntity(msg openai.ChatCompletionMessage, role string) *googlegemini.ContentEntity {
	if len(msg.MultiContent) == 0 {
		return nil
	}

	var parts []googlegemini.Part
	for _, mc := range msg.MultiContent {
		part, err := createPartFromMessageContent(mc)
		if err != nil {
			mylog.Logger.Error("Failed to create part from message content: ", zap.Error(err))
			continue
		}
		parts = append(parts, *part)
	}

	return &googlegemini.ContentEntity{
		Role:  role,
		Parts: parts,
	}
}

// createPartFromMessageContent creates a part from message content based on type.
func createPartFromMessageContent(mc openai.ChatMessagePart) (*googlegemini.Part, error) {
	switch mc.Type {
	case openai.ChatMessagePartTypeText:
		return &googlegemini.Part{Text: mc.Text}, nil
	case openai.ChatMessagePartTypeImageURL:
		imgData, mimeType, err := mycommon.GetImageURLData(mc.ImageURL.URL)
		if err != nil {
			mylog.Logger.Error(err.Error())
		}
		blob := googlegemini.Blob{
			MimeType: mimeType,
			Data:     imgData,
		}
		return &googlegemini.Part{InlineData: &blob}, nil
	default:
		return nil, fmt.Errorf("unsupported message content type: %s", mc.Type)
	}
}
