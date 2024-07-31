package aliyun_dashscope

// 定义请求和响应结构体
type BelleLLAMARequestBody struct {
	Model string `json:"model"`
	Input struct {
		Prompt string `json:"prompt"`
	} `json:"input"`
}

type BelleLLAMAResponseBody struct {
	Output struct {
		Text string `json:"text"`
	} `json:"output"`
	RequestID string `json:"request_id"`
}
