package aliyun_dashscope

// 定义请求和响应结构体
type LlamaRequestBody struct {
	Model string `json:"model"`
	Input struct {
		Prompt string `json:"prompt"`
	} `json:"input"`
}

type LlamaResponseBody struct {
	Output struct {
		Text string `json:"text"`
	} `json:"output"`
	Usage struct {
		OutputTokens int `json:"output_tokens"`
		InputTokens  int `json:"input_tokens"`
	} `json:"usage"`
	RequestID string `json:"request_id"`
}
