package aliyun_dashscope

// 定义请求和响应结构体
type DeepSeekMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type DeepSeekRequestBody struct {
	Model      string             `json:"model"`
	Input      DeepSeekInput      `json:"input"`
	Parameters DeepSeekParameters `json:"parameters,omitempty"`
}

type DeepSeekInput struct {
	Messages []DeepSeekMessage `json:"messages"`
}

type DeepSeekParameters struct {
	IncrementalOutput bool   `json:"incremental_output,omitempty"`
	ResultFormat      string `json:"result_format,omitempty"`
}

type DeepSeekResponseBody struct {
	Output struct {
		Text string `json:"text,omitempty"`
	} `json:"output"`
	RequestID string `json:"request_id"`
}
