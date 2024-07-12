package claude

import "encoding/json"

// MarshalJSON 自定义JSON序列化
func (m Message) MarshalJSON() ([]byte, error) {
	type Alias Message
	if len(m.MultiContent) > 0 && m.Content == "" {
		return json.Marshal(&struct {
			Alias
			Content []ContentBlock `json:"content"`
		}{
			Alias:   (Alias)(m),
			Content: m.MultiContent,
		})
	} else {
		return json.Marshal(&struct {
			Alias
		}{
			Alias: (Alias)(m),
		})
	}
}

// ContentBlock 定义内容块结构体
type ContentBlock struct {
	Type  string `json:"type"`
	Text  string `json:"text,omitempty"`
	Image *Image `json:"image,omitempty"`
}

// Image 定义图像结构体
type Image struct {
	Source ImageSource `json:"source"`
}

// ImageSource 定义图像源结构体
type ImageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

// Message 定义消息结构体
type Message struct {
	Role         string         `json:"role"`
	Content      string         `json:"content"`
	MultiContent []ContentBlock `json:"-"`
}

type Metadata struct {
	UserID string `json:"user_id,omitempty"`
}

// ToolInputSchema 定义工具输入的 JSON schema
type ToolInputSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	Required   []string               `json:"required,omitempty"`
}

type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema ToolInputSchema `json:"input_schema"`
}

// ToolChoice 定义工具选择结构体
type ToolChoice struct {
	Type string `json:"type"` // 可选值："tool"
	Name string `json:"name"` // 工具名称
}

// RequestBody 定义请求体结构体
type RequestBody struct {
	Model         string      `json:"model"`
	Messages      []Message   `json:"messages"`
	MaxTokens     int         `json:"max_tokens"`
	Metadata      *Metadata   `json:"metadata,omitempty"`
	StopSequences []string    `json:"stop_sequences,omitempty"`
	Stream        bool        `json:"stream"`
	System        string      `json:"system,omitempty"`
	Temperature   float32     `json:"temperature"`
	ToolChoice    *ToolChoice `json:"tool_choice,omitempty"`
	Tools         []Tool      `json:"tools,omitempty"`
	TopK          int         `json:"top_k,omitempty"`
	TopP          float32     `json:"top_p,omitempty"`
}
