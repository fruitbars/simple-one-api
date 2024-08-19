package common_btype

// Message 代表一个对话消息
type Message struct {
	Content string `json:"content"`
	Role    string `json:"role"`
}

// Input 代表一个对话的输入部分
type Input struct {
	Prompt string `json:"prompt"`
}

// ModelRequest 代表一个模型请求，包括模型名称和输入消息
type DSBtypeRequestBody struct {
	Model string `json:"model"`
	Input Input  `json:"input"`
}

type DSBtypeResponseBody struct {
	Output struct {
		Text string `json:"text"`
	} `json:"output"`
	Usage struct {
		OutputTokens int `json:"output_tokens"`
		InputTokens  int `json:"input_tokens"`
	} `json:"usage"`
	RequestID string `json:"request_id"`
}
