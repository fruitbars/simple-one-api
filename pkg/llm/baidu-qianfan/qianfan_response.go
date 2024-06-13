package baidu_qianfan

// Response 定义了整个响应体的结构
type QianFanResponse struct {
	ID               string `json:"id"`
	Object           string `json:"object,omitempty"`
	Created          int64  `json:"created,omitempty"`
	SentenceID       *int   `json:"sentence_id,omitempty"` // 在流式接口模式下返回
	IsEnd            *bool  `json:"is_end,omitempty"`      // 在流式接口模式下返回
	IsTruncated      bool   `json:"is_truncated,omitempty"`
	Result           string `json:"result,omitempty"`
	NeedClearHistory bool   `json:"need_clear_history,omitempty"`
	BanRound         *int   `json:"ban_round,omitempty"` // 只有当 need_clear_history 为 true 时返回
	Usage            Usage  `json:"usage,omitempty"`

	ErrorCode int    `json:"error_code,omitempty"`
	ErrorMsg  string `json:"error_msg,omitempty"`
}

// Usage 定义了 token 统计信息的结构
type Usage struct {
	PromptTokens     int `json:"prompt_tokens,omitempty"`
	CompletionTokens int `json:"completion_tokens,omitempty"`
	TotalTokens      int `json:"total_tokens,omitempty"`
}

// ErrorResponse 定义了错误响应的结构
type QianFanErrorResponse struct {
	ErrorCode int    `json:"error_code,omitempty"`
	ErrorMsg  string `json:"error_msg,omitempty"`
	ID        string `json:"id,omitempty"`
}
