package chat_message_list

// 定义响应数据中的Message结构体
type Message struct {
	BotID          string `json:"bot_id"`
	ChatID         string `json:"chat_id"`
	Content        string `json:"content"`
	ContentType    string `json:"content_type"`
	ID             string `json:"id"`
	ConversationID string `json:"conversation_id"`
	Role           string `json:"role"`
	Type           string `json:"type"`
}

// 定义响应结构体
type MessageListResponse struct {
	Code int       `json:"code"`
	Data []Message `json:"data"`
	Msg  string    `json:"msg"`
}
