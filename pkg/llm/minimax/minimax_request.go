package minimax

// RequestBody 定义请求体的结构
type MinimaxRequest struct {
	Model             string           `json:"model"`                         // 模型名称
	Stream            bool             `json:"stream,omitempty"`              // 是否流式返回
	TokensToGenerate  int64            `json:"tokens_to_generate,omitempty"`  // 最大生成token数
	Temperature       float32          `json:"temperature,omitempty"`         // 温度
	TopP              float32          `json:"top_p,omitempty"`               // 采样方法
	MaskSensitiveInfo bool             `json:"mask_sensitive_info,omitempty"` // 是否打码敏感信息
	Messages          []Message        `json:"messages"`                      // 对话内容
	BotSetting        []BotSetting     `json:"bot_setting"`                   // 机器人的设定
	ReplyConstraints  ReplyConstraints `json:"reply_constraints"`             // 模型回复要求
}

// Message 定义对话内容的结构
type Message struct {
	SenderType string `json:"sender_type"` // 发送者类型
	SenderName string `json:"sender_name"` // 发送者名称
	Text       string `json:"text"`        // 消息内容
}

// BotSetting 定义机器人设定的结构
type BotSetting struct {
	BotName string `json:"bot_name"` // 机器人的名字
	Content string `json:"content"`  // 具体机器人的设定
}

// ReplyConstraints 定义模型回复要求的结构
type ReplyConstraints struct {
	SenderType string `json:"sender_type"` // 回复角色类型
	SenderName string `json:"sender_name"` // 回复机器人名称
}
