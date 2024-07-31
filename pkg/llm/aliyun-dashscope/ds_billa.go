package aliyun_dashscope

type BillaRequestBody struct {
	Model string `json:"model"`
	Input struct {
		Prompt string `json:"prompt"`
	} `json:"input"`
}

type BillaResponseBody struct {
	Output struct {
		Text string `json:"text"`
	} `json:"output"`
	RequestID string `json:"request_id"`
}
