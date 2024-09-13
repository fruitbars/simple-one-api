package chat_retrieve

// 定义使用情况的结构体
type Usage struct {
	InputCount  int `json:"input_count"`
	OutputCount int `json:"output_count"`
	TokenCount  int `json:"token_count"`
}

// 定义数据结构体
type ChatData struct {
	BotID          string `json:"bot_id"`
	CompletedAt    int64  `json:"completed_at"`
	ConversationID string `json:"conversation_id"`
	CreatedAt      int64  `json:"created_at"`
	ID             string `json:"id"`
	Status         string `json:"status"`
	Usage          Usage  `json:"usage"`
}

// 定义响应结构体
type ChatRetrieveResponse struct {
	Code int      `json:"code"`
	Data ChatData `json:"data"`
	Msg  string   `json:"msg"`
}
