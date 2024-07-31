package aliyun_dashscope

// 定义请求和响应结构体
type BaiChuanMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type BaiChuanRequestBody struct {
	Model      string            `json:"model"`
	Input      BaiChuanInput     `json:"input"`
	Parameters BaiChuanParameter `json:"parameters,omitempty"`
}

type BaiChuanInput struct {
	Prompt   string            `json:"prompt,omitempty"`
	Messages []BaiChuanMessage `json:"messages,omitempty"`
}

type BaiChuanParameter struct {
	ResultFormat string `json:"result_format,omitempty"`
}

type BaiChuanResponseBody struct {
	Output struct {
		Text    string           `json:"text,omitempty"`
		Choices []BaiChuanChoice `json:"choices,omitempty"`
	} `json:"output"`
	RequestID string `json:"request_id"`
	Usage     struct {
		OutputTokens int `json:"output_tokens"`
		InputTokens  int `json:"input_tokens"`
	} `json:"usage"`
}

type BaiChuanChoice struct {
	Message struct {
		Role        string `json:"role"`
		ContentType string `json:"content_type"`
		Content     string `json:"content"`
	} `json:"message"`
	FinishReason string `json:"finish_reason"`
}
