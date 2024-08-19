package aliyun_dashscope_adapter

// Input 代表输入部分，包括提示词
type Input struct {
	Prompt string `json:"prompt"`
}

// Parameters 代表请求的参数部分，可以为空
type Parameters struct {
	// 如果参数有具体字段，可以在这里定义；当前为空
}

// ModelRequest 代表整个请求结构体，包括模型名称、输入和参数
type ModelRequest struct {
	Model      string     `json:"model"`
	Input      Input      `json:"input"`
	Parameters Parameters `json:"parameters"`
}
