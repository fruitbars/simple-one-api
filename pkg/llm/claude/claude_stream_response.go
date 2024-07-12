package claude

// 定义事件类型的通用结构体
type Event struct {
	Type              string             `json:"type"`
	Message           *RespMessage       `json:"message,omitempty"`
	ContentBlock      *ContentBlock      `json:"content_block,omitempty"`
	Delta             *Delta             `json:"delta,omitempty"`
	Usage             *Usage             `json:"usage,omitempty"`
	Index             *int               `json:"index,omitempty"`
	MessageEnd        *MessageEnd        `json:"message_end,omitempty"`
	StopReason        *string            `json:"stop_reason,omitempty"`
	StopSequence      *string            `json:"stop_sequence,omitempty"`
	ContentBlockDelta *ContentBlockDelta `json:"content_block_delta,omitempty"`
	ContentBlockStop  *ContentBlockStop  `json:"content_block_stop,omitempty"`
}

type RespMessage struct {
	ID           string   `json:"id"`
	Type         string   `json:"type"`
	Role         string   `json:"role"`
	Content      []string `json:"content"`
	Model        string   `json:"model"`
	StopReason   *string  `json:"stop_reason"`
	StopSequence *string  `json:"stop_sequence"`
	Usage        Usage    `json:"usage"`
}

type Delta struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type MessageEnd struct {
	StopReason   string  `json:"stop_reason"`
	StopSequence *string `json:"stop_sequence"`
}

type ContentBlockDelta struct {
	Index int   `json:"index"`
	Delta Delta `json:"delta"`
}

type ContentBlockStop struct {
	Index int `json:"index"`
}

type MsgMessageStart struct {
	Type    string `json:"type"`
	Message struct {
		ID           string `json:"id"`
		Type         string `json:"type"`
		Role         string `json:"role"`
		Model        string `json:"model"`
		StopSequence any    `json:"stop_sequence"`
		Usage        struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
		Content    []any   `json:"content"`
		StopReason *string `json:"stop_reason"`
	} `json:"message"`
}

type MsgContentBlockStart struct {
	Type         string `json:"type"`
	Index        int    `json:"index"`
	ContentBlock struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content_block"`
}

type MsgContentBlockDelta struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
	Delta struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"delta"`
}

type MsgContentBlockStop struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
}

type MsgMessageDelta struct {
	Type  string `json:"type"`
	Delta struct {
		StopReason   string `json:"stop_reason"`
		StopSequence any    `json:"stop_sequence"`
	} `json:"delta"`
	Usage struct {
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

type MsgMessageStop struct {
	Type string `json:"type"`
}
