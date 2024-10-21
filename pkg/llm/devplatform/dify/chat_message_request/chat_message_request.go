package chat_message_request

type ChatMessageRequest struct {
	Inputs         map[string]interface{} `json:"inputs"`
	Query          string                 `json:"query"`
	ResponseMode   string                 `json:"response_mode"`
	ConversationID string                 `json:"conversation_id"`
	User           string                 `json:"user"`
}
