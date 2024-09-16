package chat

// 定义结构体
type LastError struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

type Data struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversation_id"`
	BotID          string    `json:"bot_id"`
	CreatedAt      int64     `json:"created_at"` // 将Unix时间戳转换为time.Time类型
	LastError      LastError `json:"last_error"`
	Status         string    `json:"status"`
}

type Response struct {
	Data Data   `json:"data"`
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}
