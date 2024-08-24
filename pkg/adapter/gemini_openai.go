package adapter

import (
	google_gemini "simple-one-api/pkg/llm/google-gemini"
	myopenai "simple-one-api/pkg/openai"
	"strings"
	"time"
)

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
			Delta: myopenai.ResponseDelta{
				Role:    role,
				Content: content,
			},
			//FinishReason: candidate.FinishReason,
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
