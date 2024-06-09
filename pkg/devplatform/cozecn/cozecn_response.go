package cozecn

type StreamResponse struct {
	Event            string  `json:"event"`
	Message          Message `json:"message,omitempty"`
	IsFinish         bool    `json:"is_finish,omitempty"`
	Index            int     `json:"index,omitempty"`
	ConversationID   string  `json:"conversation_id,omitempty"`
	ErrorInformation struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	} `json:"error_information,omitempty"`
}

type Response struct {
	Messages       []Message `json:"messages"`
	ConversationID string    `json:"conversation_id"`
	Code           int       `json:"code"`
	Msg            string    `json:"msg"`
}
