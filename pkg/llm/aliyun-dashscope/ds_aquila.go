package aliyun_dashscope

// 定义请求和响应结构体
type AquilaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AquilaRequestBody struct {
	Model      string           `json:"model"`
	Input      AquilaInput      `json:"input"`
	Parameters AquilaParameters `json:"parameters,omitempty"`
}

type AquilaInput struct {
	Messages []AquilaMessage `json:"messages"`
}

type AquilaParameters struct {
	IncrementalOutput bool   `json:"incremental_output,omitempty"`
	ResultFormat      string `json:"result_format,omitempty"`
}

type AquilaResponseBody struct {
	Output struct {
		Text string `json:"text,omitempty"`
	} `json:"output"`
	RequestID string `json:"request_id"`
}
