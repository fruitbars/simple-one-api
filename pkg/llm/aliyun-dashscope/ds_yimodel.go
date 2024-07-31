package aliyun_dashscope

// 定义请求和响应结构体
type YiModelMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type YiModelRequestBody struct {
	Model      string            `json:"model"`
	Input      YiModelInput      `json:"input"`
	Parameters YiModelParameters `json:"parameters,omitempty"`
}

type YiModelInput struct {
	Messages []YiModelMessage `json:"messages"`
}

type YiModelParameters struct {
	IncrementalOutput bool   `json:"incremental_output,omitempty"`
	ResultFormat      string `json:"result_format,omitempty"`
}

type YiModelResponseBody struct {
	Output struct {
		Text string `json:"text,omitempty"`
	} `json:"output"`
	RequestID string `json:"request_id"`
}
