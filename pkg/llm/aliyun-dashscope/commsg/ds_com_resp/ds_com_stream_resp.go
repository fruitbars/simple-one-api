package ds_com_resp

type StreamResponseOutput struct {
	Choices []StreamResponseChoice `json:"choices"`
}

type StreamResponseChoice struct {
	Message      StreamResponseMessage `json:"message"`
	FinishReason string                `json:"finish_reason"`
}

type StreamResponseMessage struct {
	Content string `json:"content"`
	Role    string `json:"role"`
}

type StreamResponseUsage struct {
	Kens         int `json:"kens"`
	OutputTokens int `json:"output_tokens"`
}

type ModelStreamResponse struct {
	Output    StreamResponseOutput `json:"output"`
	Usage     StreamResponseUsage  `json:"usage"`
	RequestID string               `json:"request_id"`
}
