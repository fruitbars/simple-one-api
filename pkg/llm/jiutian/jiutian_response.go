package jiutian

// Usage 使用统计
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`     // 输入的token数量
	CompletionTokens int `json:"completion_tokens"` // 生成的token数量
	TotalTokens      int `json:"total_tokens"`      // 总token数量
}

// ChatCompletionResponse 九天模型的对话响应结构
type ChatCompletionResponse struct {
	Usage    Usage     `json:"Usage"`    // 使用统计
	Response string    `json:"response"` // 模型回答
	Delta    string    `json:"delta"`    // 结束标记
	Finished string    `json:"finished"` // 结束原因
	History  [][]string `json:"history"` // 历史对话记录
}

// ChatCompletionStreamResponse 九天模型的流式响应结构
type ChatCompletionStreamResponse struct {
	Response string    `json:"response"` // 当前生成的内容
	Delta    string    `json:"delta"`    // 结束标记
	Finished string    `json:"finished"` // 结束原因
	History  [][]string `json:"history"` // 历史对话记录
} 