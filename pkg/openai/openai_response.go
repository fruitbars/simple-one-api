package myopenai

import "encoding/json"

type OpenAIResponse struct {
	ID                string   `json:"id"`
	Object            string   `json:"object,omitempty"`
	Created           int64    `json:"created,omitempty"`
	Model             string   `json:"model,omitempty"`
	SystemFingerprint string   `json:"system_fingerprint,omitempty"`
	Choices           []Choice `json:"choices,omitempty"`
	Usage             *Usage   `json:"usage,omitempty"`

	Error *ErrorDetail `json:"error,omitempty"`
}

// Choice 定义了响应中的选择项结构
type Choice struct {
	Index        int              `json:"index"`
	Message      ResponseMessage  `json:"message"`
	LogProbs     *json.RawMessage `json:"logprobs"` // 使用 RawMessage 以便处理可能为 null 的情况
	FinishReason string           `json:"finish_reason"`
}

// ResponseMessage Message 定义了对话中的消息结构
type ResponseMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Usage 定义了使用统计的结构
type Usage struct {
	PromptTokens     int `json:"prompt_tokens,omitempty"`
	CompletionTokens int `json:"completion_tokens,omitempty"`
	TotalTokens      int `json:"total_tokens,omitempty"`
}

// ErrorDetail 包含具体的错误详情
type ErrorDetail struct {
	Message string      `json:"message,omitempty"` // 错误消息
	Type    string      `json:"type,omitempty"`    // 错误类型
	Param   interface{} `json:"param,omitempty"`   // 参数，可能为 null，所以使用 interface{}
	Code    interface{} `json:"code,omitempty"`    // 错误代码，可能为 null，同样使用 interface{}
}

type OpenAIStreamResponse struct {
	ID                string                       `json:"id,omitempty"`
	Object            string                       `json:"object,omitempty"`
	Created           int64                        `json:"created,omitempty"`
	Model             string                       `json:"model,omitempty"`
	SystemFingerprint string                       `json:"system_fingerprint,omitempty"`
	Choices           []OpenAIStreamResponseChoice `json:"choices,omitempty"`
	Usage             *Usage                       `json:"usage,omitempty"`
	Error             *ErrorDetail                 `json:"error,omitempty"`
}

type OpenAIStreamResponseChoice struct {
	Index int `json:"index,omitempty"`
	Delta struct {
		Role    string `json:"role,omitempty"`
		Content string `json:"content,omitempty"`
	} `json:"delta,omitempty"`
	Logprobs     any `json:"logprobs,omitempty"`
	FinishReason any `json:"finish_reason,omitempty"`
}
