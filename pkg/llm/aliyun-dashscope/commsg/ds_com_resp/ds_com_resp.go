package ds_com_resp

// Message 代表返回的消息内容
type Message struct {
	Role        string `json:"role"`
	ContentType string `json:"content_type"`
	Content     string `json:"content"`
}

// Choice 代表返回的一个选择项
type Choice struct {
	FinishReason string  `json:"finish_reason"`
	Message      Message `json:"message"`
}

// Output 代表输出部分，包括多个选择项
type Output struct {
	Choices []Choice `json:"choices"`
}

// Usage 代表请求的使用情况，包括输入和输出的token数量
type Usage struct {
	OutputTokens int `json:"output_tokens"`
	InputTokens  int `json:"input_tokens"`
}

// ModelResponse 代表整个响应结构体，包括输出、使用情况和请求ID
type ModelResponse struct {
	Output    Output `json:"output"`
	Usage     Usage  `json:"usage"`
	RequestID string `json:"request_id"`
}
