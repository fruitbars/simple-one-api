package cozecn

// 请求数据结构体
type CozeRequest struct {
	ConversationID string    `json:"conversation_id"`
	BotID          string    `json:"bot_id"`
	User           string    `json:"user"`
	Query          string    `json:"query"`
	Stream         bool      `json:"stream"`
	ChatHistory    []Message `json:"chat_history,omitempty"`
}

type Message struct {
	Role        string `json:"role"`
	Type        string `json:"type,omitempty"`
	Content     string `json:"content"`
	ContentType string `json:"content_type"`
}
