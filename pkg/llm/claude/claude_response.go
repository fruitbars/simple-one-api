package claude

type StopReasonType string

const (
	EndTurn      StopReasonType = "end_turn"
	MaxTokens    StopReasonType = "max_tokens"
	StopSequence StopReasonType = "stop_sequence"
	ToolUse      StopReasonType = "tool_use"
)

type RespContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type ResponseBody struct {
	ID           string        `json:"id"`
	Type         string        `json:"type"`
	Role         string        `json:"role"`
	Content      []RespContent `json:"content"`
	Model        string        `json:"model"`
	StopReason   string        `json:"stop_reason"`
	StopSequence string        `json:"stop_sequence"`
	Usage        struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}
