package aliyun_dashscope

// 定义请求和响应结构体
type InternLMMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type InternLMRequestBody struct {
	Model      string             `json:"model"`
	Input      InternLMInput      `json:"input"`
	Parameters InternLMParameters `json:"parameters,omitempty"`
}

type InternLMInput struct {
	Messages []InternLMMessage `json:"messages"`
}

type InternLMParameters struct {
	IncrementalOutput bool   `json:"incremental_output,omitempty"`
	ResultFormat      string `json:"result_format,omitempty"`
}

type InternLMResponseBody struct {
	Output struct {
		Text string `json:"text,omitempty"`
	} `json:"output"`
	RequestID string `json:"request_id"`
}
