package adapter

import (
	"testing"

	google_gemini "simple-one-api/pkg/llm/google-gemini"
)

func TestGeminiResponseUsesOpenAIAssistantRole(t *testing.T) {
	response := geminiResponseWithModelRole()

	nonStream := GeminiResponseToOpenAIResponse(response)
	if got := nonStream.Choices[0].Message.Role; got != "assistant" {
		t.Fatalf("non-stream role = %q, want assistant", got)
	}

	stream := GeminiResponseToOpenAIStreamResponse(response)
	if got := stream.Choices[0].Delta.Role; got != "assistant" {
		t.Fatalf("stream role = %q, want assistant", got)
	}
}

func geminiResponseWithModelRole() *google_gemini.GeminiResponse {
	return &google_gemini.GeminiResponse{
		Candidates: []google_gemini.Candidate{{
			Content: google_gemini.ContentEntity{
				Role:  "model",
				Parts: []google_gemini.Part{{Text: "hello"}},
			},
		}},
	}
}
