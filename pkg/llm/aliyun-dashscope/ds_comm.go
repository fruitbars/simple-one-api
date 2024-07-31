package aliyun_dashscope

// 定义请求和响应结构体
type AliyunComMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AliyunComInput struct {
	Prompt   string             `json:"prompt,omitempty"`
	Messages []AliyunComMessage `json:"messages,omitempty"`
}

type AliyunComParameters struct {
	IncrementalOutput bool   `json:"incremental_output,omitempty"`
	ResultFormat      string `json:"result_format,omitempty"`
}

type AliyunComRequestBody struct {
	Model      string              `json:"model"`
	Input      AliyunComInput      `json:"input"`
	Parameters AliyunComParameters `json:"parameters,omitempty"`
}
type AliyunComResponseBody struct {
	Output struct {
		Text string `json:"text,omitempty"`
	} `json:"output"`
	RequestID string `json:"request_id"`
}
