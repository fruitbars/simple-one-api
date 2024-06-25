package mycommon

// Message 定义了对话中的消息结构体
type Message struct {
	Role    string `json:"role"`    // 用户或助手的角色
	Content string `json:"content"` // 对话内容
}
