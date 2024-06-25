package adapter

import (
	"github.com/sashabaranov/go-openai"
	google_gemini "simple-one-api/pkg/llm/google-gemini"
	"simple-one-api/pkg/mycommon"
	myopenai "simple-one-api/pkg/openai"
	"strings"
	"time"
)

func OpenAIRequestToGeminiRequest(oaiReq openai.ChatCompletionRequest) *google_gemini.GeminiRequest {
	// 初始化 GeminiRequest 结构
	var Contents []google_gemini.ContentEntity

	//hisMessagesLen := len(oaiReq.Messages)
	hisMessages := mycommon.ConvertSystemMessages2NoSystem(oaiReq.Messages)

	// 转换聊天消息为 Gemini 的内容条目
	for _, msg := range hisMessages {
		role := msg.Role
		if strings.ToLower(msg.Role) == "assistant" {
			role = "model"
		}
		content := google_gemini.ContentEntity{
			Role:  role,
			Parts: []google_gemini.Part{{Text: msg.Content}},
		}

		Contents = append(Contents, content)
	}

	/*
		geminiReq.SafetySettings = append(geminiReq.SafetySettings, google_gemini.SafetySetting{
			Category:  "HARM_CATEGORY_DANGEROUS_CONTENT",
			Threshold: "BLOCK_ONLY_HIGH",
		})

	*/
	geminiReq := &google_gemini.GeminiRequest{
		Contents:       Contents,
		SafetySettings: []google_gemini.SafetySetting{},
		GenerationConfig: google_gemini.GenerationConfig{
			StopSequences:   oaiReq.Stop,
			Temperature:     float64(oaiReq.Temperature),
			MaxOutputTokens: oaiReq.MaxTokens,
			TopP:            float64(oaiReq.TopP),
			TopK:            oaiReq.TopLogProbs,
		},
	}

	return geminiReq
}

func GeminiResponseToOpenAIResponse(qfResp *google_gemini.GeminiResponse) *myopenai.OpenAIResponse {
	// 创建 OpenAIResponse 实例
	openAIResp := &myopenai.OpenAIResponse{
		Object: "chat.completion",
		Usage: &myopenai.Usage{
			PromptTokens:     qfResp.UsageMetadata.PromptTokenCount,
			CompletionTokens: qfResp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      qfResp.UsageMetadata.TotalTokenCount,
		},
		Choices: make([]myopenai.Choice, len(qfResp.Candidates)),
	}

	// 遍历所有候选项
	for i, candidate := range qfResp.Candidates {

		role := candidate.Content.Role
		if strings.ToLower(role) == "model" {
			role = "assitant"
		}

		var content string
		if len(candidate.Content.Parts) > 0 {
			content = candidate.Content.Parts[0].Text
		}

		openAIResp.Choices[i] = myopenai.Choice{
			Index: candidate.Index,
			Message: myopenai.ResponseMessage{
				Role:    role,
				Content: content,
			},
			FinishReason: candidate.FinishReason,
		}

		// 示例代码，假设不处理 LogProbs
		/*
			var logProbs json.RawMessage = nil
			openAIResp.Choices[i].LogProbs = &logProbs

		*/
	}

	return openAIResp
}

func GeminiResponseToOpenAIStreamResponse(qfResp *google_gemini.GeminiResponse) *myopenai.OpenAIStreamResponse {
	if qfResp == nil {
		return nil
	}

	var Choices []myopenai.OpenAIStreamResponseChoice

	for i, candidate := range qfResp.Candidates {
		role := candidate.Content.Role
		if strings.ToLower(role) == "model" {
			role = "assitant"
		}

		var content string
		if len(candidate.Content.Parts) > 0 {
			content = candidate.Content.Parts[0].Text
		}

		choice := myopenai.OpenAIStreamResponseChoice{
			Index: i,
			Delta: struct {
				Role    string `json:"role,omitempty"`
				Content string `json:"content,omitempty"`
			}{
				Role:    role,
				Content: content,
			},
			FinishReason: candidate.FinishReason,
		}

		Choices = append(Choices, choice)
	}

	openAIResponse := &myopenai.OpenAIStreamResponse{
		ID:      "chatcmpl-" + time.Now().Format("20060102150405"), // 生成一个唯一的ID
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		//Model:   "gpt-3.5-turbo-0613", // 假设模型名称
		Choices: Choices,
		Usage: &myopenai.Usage{
			PromptTokens:     qfResp.UsageMetadata.PromptTokenCount,
			CompletionTokens: qfResp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      qfResp.UsageMetadata.TotalTokenCount,
		},
	}

	return openAIResponse
}
