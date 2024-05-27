package minimax

// Response 定义响应的结构体
type MinimaxResponse struct {
	Created             int64    `json:"created"`                         // 请求发起时间
	Model               string   `json:"model"`                           // 请求指定的模型名称
	Reply               string   `json:"reply"`                           // 回复内容
	InputSensitive      bool     `json:"input_sensitive"`                 // 输入命中敏感词
	InputSensitiveType  int64    `json:"input_sensitive_type,omitempty"`  // 输入命中敏感词类型
	OutputSensitive     bool     `json:"output_sensitive"`                // 输出命中敏感词
	OutputSensitiveType int64    `json:"output_sensitive_type,omitempty"` // 输出命中敏感词类型
	Choices             []Choice `json:"choices"`                         // 所有结果
	Usage               Usage    `json:"usage"`                           // tokens数使用情况
	ID                  string   `json:"id"`                              // 本次请求的唯一标识
	BaseResp            BaseResp `json:"base_resp"`                       // 错误状态码和详情
}

// Choice 定义选择结果的结构体
type Choice struct {
	Messages     []Message `json:"messages"`      // 回复结果的具体内容
	Index        int64     `json:"index"`         // 排名
	FinishReason string    `json:"finish_reason"` // 结束原因
}

// Usage 定义tokens使用情况的结构体
type Usage struct {
	TotalTokens int64 `json:"total_tokens"` // 消耗tokens总数
}

// BaseResp 定义错误状态码和详情的结构体
type BaseResp struct {
	StatusCode int64  `json:"status_code"` // 状态码
	StatusMsg  string `json:"status_msg"`  // 错误详情
}
