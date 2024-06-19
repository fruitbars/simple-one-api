package google_gemini

// SafetyRating 定义安全等级评分
type SafetyRating struct {
	Category    string `json:"category"`
	Probability string `json:"probability"`
}

// Candidate 定义候选者信息
type Candidate struct {
	Content       ContentEntity  `json:"content"`
	FinishReason  string         `json:"finishReason"`
	Index         int            `json:"index"`
	SafetyRatings []SafetyRating `json:"safetyRatings"`
}

// UsageMetadata 定义使用元数据
type UsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

// GeminiResponse 定义总体响应结构
type GeminiResponse struct {
	Candidates    []Candidate   `json:"candidates"`
	UsageMetadata UsageMetadata `json:"usageMetadata"`
}
