package google_gemini

type Part struct {
	Text string `json:"text"`
}

/*
type Content struct {
	Parts []Part `json:"parts"`
}

*/

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
	Temperature     float64  `json:"temperature,omitempty"`
	MaxOutputTokens int      `json:"maxOutputTokens,omitempty"`
	TopP            float64  `json:"topP,omitempty"`
	TopK            int      `json:"topK,omitempty"`
}

type GeminiRequest struct {
	Contents         []ContentEntity  `json:"contents"`
	SafetySettings   []SafetySetting  `json:"safetySettings,omitempty"`
	GenerationConfig GenerationConfig `json:"generationConfig,omitempty"`
}
