package google_gemini

import "fmt"

type Part struct {
	Text       string `json:"text,omitempty"`
	InlineData *Blob  `json:"inlineData,omitempty"`
}

// Blob 表示内嵌的媒体字节数据
type Blob struct {
	MimeType string `json:"mimeType,omitempty"`
	Data     string `json:"data,omitempty"`
}

// Entry represents a single entry in the conversation.
type ContentEntity struct {
	Role  string `json:"role"`
	Parts []Part `json:"parts"`
}

type SafetySetting struct {
	Category  string `json:"category,omitempty"`
	Threshold string `json:"threshold,omitempty"`
}

type GenerationConfig struct {
	StopSequences   []string `json:"stopSequences,omitempty"`
	Temperature     float32  `json:"temperature,omitempty"`
	MaxOutputTokens int      `json:"maxOutputTokens,omitempty"`
	TopP            float32  `json:"topP,omitempty"`
	TopK            int      `json:"topK,omitempty"`
}

type GeminiRequest struct {
	Contents         []ContentEntity  `json:"contents"`
	SafetySettings   []SafetySetting  `json:"safetySettings,omitempty"`
	GenerationConfig GenerationConfig `json:"generationConfig,omitempty"`
}

func (b Blob) GoString() string {
	return fmt.Sprintf("Blob{MimeType: %q, Data : ...}", b.MimeType)
}
