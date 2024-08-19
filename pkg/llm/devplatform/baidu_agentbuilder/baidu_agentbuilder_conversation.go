package baidu_agentbuilder

// MessageContent 定义消息主体的结构
type ConversationMessageContent struct {
	Type  string                 `json:"type"`
	Value map[string]interface{} `json:"value"`
}

// Message 定义会话请求消息的结构
type ConversationMessage struct {
	Content ConversationMessageContent `json:"content"`
}

// ConversationRequest 定义 Conversation 请求的结构
type ConversationRequest struct {
	Message  ConversationMessage `json:"message"`
	Source   string              `json:"source"`
	From     string              `json:"from"`
	OpenID   string              `json:"openId"`
	ThreadID string              `json:"threadId,omitempty"`
}

type ConversationResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	LogID   string `json:"logid"`
	Data    struct {
		Message struct {
			Content []struct {
				DataType   string `json:"dataType"`
				IsFinished bool   `json:"isFinished"`
				Data       struct {
					Text string `json:"text"`
				} `json:"data"`
			} `json:"content"`
			ThreadID string `json:"threadId"`
			MsgID    string `json:"msgId"`
			EndTurn  bool   `json:"endTurn"`
		} `json:"message"`
	} `json:"data"`
}
