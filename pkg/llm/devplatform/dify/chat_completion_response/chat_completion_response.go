package chat_completion_response

import "time"

// RetrieverResource represents a retriever resource metadata
type RetrieverResource struct {
	Position     int     `json:"position"`
	DatasetID    string  `json:"dataset_id"`
	DatasetName  string  `json:"dataset_name"`
	DocumentID   string  `json:"document_id"`
	DocumentName string  `json:"document_name"`
	SegmentID    string  `json:"segment_id"`
	Score        float64 `json:"score"`
	Content      string  `json:"content"`
}

// Usage represents the usage metadata
type Usage struct {
	PromptTokens        int     `json:"prompt_tokens"`
	PromptUnitPrice     string  `json:"prompt_unit_price"`
	PromptPriceUnit     string  `json:"prompt_price_unit"`
	PromptPrice         string  `json:"prompt_price"`
	CompletionTokens    int     `json:"completion_tokens"`
	CompletionUnitPrice string  `json:"completion_unit_price"`
	CompletionPriceUnit string  `json:"completion_price_unit"`
	CompletionPrice     string  `json:"completion_price"`
	TotalTokens         int     `json:"total_tokens"`
	TotalPrice          string  `json:"total_price"`
	Currency            string  `json:"currency"`
	Latency             float64 `json:"latency"`
}

// Metadata represents the metadata of the event
type Metadata struct {
	Usage              Usage               `json:"usage"`
	RetrieverResources []RetrieverResource `json:"retriever_resources"`
}

// Event represents the main event structure
type ChatCompletionResponse struct {
	Event          string    `json:"event"`
	MessageID      string    `json:"message_id"`
	ConversationID string    `json:"conversation_id"`
	Mode           string    `json:"mode"`
	Answer         string    `json:"answer"`
	Metadata       Metadata  `json:"metadata"`
	CreatedAt      time.Time `json:"created_at"`
}
