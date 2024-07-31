package aliyun_dashscope

// 定义请求和响应结构体
type ChatGlMMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatGlMRequestBody struct {
	Model      string           `json:"model"`
	Input      ChatGlMInput     `json:"input"`
	Parameters ChatGlMParameter `json:"parameters,omitempty"`
}

type ChatGlMInput struct {
	Prompt   string           `json:"prompt,omitempty"`
	History  []string         `json:"history,omitempty"`
	Messages []ChatGlMMessage `json:"messages,omitempty"`
}

type ChatGlMParameter struct {
	IncrementalOutput bool   `json:"incremental_output,omitempty"`
	ResultFormat      string `json:"result_format,omitempty"`
}

type ChatGlMResponseBody struct {
	Output struct {
		Text    string          `json:"text,omitempty"`
		Choices []ChatGlMChoice `json:"choices,omitempty"`
	} `json:"output"`
	RequestID string `json:"request_id"`
	Usage     struct {
		OutputTokens int `json:"output_tokens"`
		InputTokens  int `json:"input_tokens"`
	} `json:"usage"`
}

type ChatGlMChoice struct {
	Message struct {
		Role        string `json:"role"`
		ContentType string `json:"content_type"`
		Content     string `json:"content"`
	} `json:"message"`
	FinishReason string `json:"finish_reason"`
}
