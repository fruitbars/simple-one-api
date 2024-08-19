package ds_com_request

// Input 代表输入部分，包括提示词
type ModelPromptInput struct {
	Prompt string `json:"prompt"`
}

// ModelRequest 代表整个请求结构体，包括模型名称、输入和参数
type ModelPromptRequest struct {
	Model      string           `json:"model"`
	Input      ModelPromptInput `json:"input"`
	Parameters *Parameters      `json:"parameters"`
}
