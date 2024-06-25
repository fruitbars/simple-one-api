package myopenai

import (
	"encoding/json"
	"simple-one-api/pkg/mycommon"
)

// RequestBody 定义 API 请求的主体结构
type OpenAIRequest struct {
	Model            string             `json:"model"`
	Messages         []mycommon.Message `json:"messages"`
	FrequencyPenalty *float32           `json:"frequency_penalty,omitempty"`
	LogitBias        map[int]int        `json:"logit_bias,omitempty"`
	LogProbs         *bool              `json:"logprobs,omitempty"`
	TopLogProbs      *int               `json:"top_logprobs,omitempty"`
	MaxTokens        *int               `json:"max_tokens,omitempty"`
	N                *int               `json:"n,omitempty"`
	PresencePenalty  *float32           `json:"presence_penalty,omitempty"`
	ResponseFormat   *ResponseFormat    `json:"response_format,omitempty"`
	Seed             *int               `json:"seed,omitempty"`
	Stop             []string           `json:"stop,omitempty"`
	Stream           *bool              `json:"stream,omitempty"`
	StreamOptions    *StreamOptions     `json:"stream_options,omitempty"`
	Temperature      *float32           `json:"temperature,omitempty"`
	TopP             *float32           `json:"top_p,omitempty"`
	Tools            []Tool             `json:"tools,omitempty"`
	ToolChoice       json.RawMessage    `json:"tool_choice,omitempty"`
	User             *string            `json:"user,omitempty"`
}

// ResponseFormat 定义响应格式的结构
type ResponseFormat struct {
	Type string `json:"type"`
}

// StreamOptions 定义流选项的结构
type StreamOptions struct {
	// 详细字段可以根据具体实现需求添加
}

// Tool 定义工具，如函数
type Tool struct {
	Type     string    `json:"type"`
	Function *Function `json:"function,omitempty"`
}

// Function 定义函数的结构
type Function struct {
	Name string `json:"name"`
}

// ToolChoice 定义工具选择的结构
type ToolChoiceFunction struct {
	Type     string    `json:"type"`
	Function *Function `json:"function,omitempty"`
}
