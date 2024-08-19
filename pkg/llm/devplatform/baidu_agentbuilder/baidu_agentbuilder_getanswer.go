package baidu_agentbuilder

// 定义请求消息的结构
type GetAnswerMessageContent struct {
	Type  string            `json:"type"`
	Value map[string]string `json:"value"`
}

type GetAnswerMessage struct {
	Content GetAnswerMessageContent `json:"content"`
}

type GetAnswerRequest struct {
	Message  GetAnswerMessage `json:"message"`
	Source   string           `json:"source"`
	From     string           `json:"from"`
	OpenID   string           `json:"openId"`
	ThreadID string           `json:"threadId,omitempty"`
}

// 定义响应消息的结构
type GetAnswerResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	LogID   string `json:"logid"`
	Data    struct {
		Content []struct {
			DataType string `json:"dataType"`
			Data     string `json:"data"`
		} `json:"content"`
		ThreadID string `json:"threadId"`
		MsgID    string `json:"msgId"`
	} `json:"data"`
}
