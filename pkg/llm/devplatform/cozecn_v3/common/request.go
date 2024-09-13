package common

// 对话请求结构体
type ChatRequest struct {
	BotID              string            `json:"bot_id"`
	UserID             string            `json:"user_id"`
	Stream             bool              `json:"stream"`
	AutoSaveHistory    bool              `json:"auto_save_history"`
	AdditionalMessages []Message         `json:"additional_messages"`
	CustomVariables    map[string]string `json:"custom_variables,omitempty"`
	ExtraParams        map[string]string `json:"extra_params,omitempty"`
}

// 消息结构体
type Message struct {
	Role        string `json:"role"`
	Content     string `json:"content"`
	ContentType string `json:"content_type"`
}

type ObjectStringMessage struct {
	Type    string `json:"type"`
	Text    string `json:"text,omitempty"`
	FileID  string `json:"file_id,omitempty"`
	FileURL string `json:"file_url,omitempty"`
}
