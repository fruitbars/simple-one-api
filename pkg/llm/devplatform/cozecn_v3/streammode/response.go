package streammode

type EventData struct {
	ID             string `json:"id"`
	ConversationID string `json:"conversation_id"`
	BotID          string `json:"bot_id"`
	Role           string `json:"role"`
	Type           string `json:"type"`
	Content        string `json:"content"`
	ContentType    string `json:"content_type"`
	ChatID         string `json:"chat_id"`
	CompletedAt    int64  `json:"completed_at"`
	LastError      struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	} `json:"last_error"`
	Status string `json:"status"`
	Usage  struct {
		TokenCount  int `json:"token_count"`
		OutputCount int `json:"output_count"`
		InputCount  int `json:"input_count"`
	} `json:"usage"`
}
