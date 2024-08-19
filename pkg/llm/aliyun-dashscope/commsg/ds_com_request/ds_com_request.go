package ds_com_request

// Message 代表一个对话消息
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Input 代表一个对话的输入部分
type Input struct {
	Messages []Message `json:"messages"`
}

// Parameters 代表请求的参数
type Parameters struct {
	ResultFormat string `json:"result_format,omitempty"`
}

// ModelRequest 代表一个模型请求，包括模型名称、输入消息和参数
type ModelRequest struct {
	Model      string      `json:"model"`
	Input      Input       `json:"input"`
	Parameters *Parameters `json:"parameters,omitempty"`
}
