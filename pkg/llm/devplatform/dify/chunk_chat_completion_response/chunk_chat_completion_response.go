package chunk_chat_completion_response

// CommonEvent represents the common structure for all events
type CommonEvent struct {
	Event string `json:"event"`
}

// MessageEvent represents a standard message event
type MessageEvent struct {
	Event          string `json:"event"`
	MessageID      string `json:"message_id"`
	ConversationID string `json:"conversation_id"`
	Answer         string `json:"answer"`
	CreatedAt      int64  `json:"created_at"`
}

// MessageEndEvent represents the end of a message event
type MessageEndEvent struct {
	Event          string   `json:"event"`
	ID             string   `json:"id"`
	ConversationID string   `json:"conversation_id"`
	Metadata       Metadata `json:"metadata"`
}

// Metadata represents metadata for the message_end event
type Metadata struct {
	Usage struct {
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
	} `json:"usage"`
	RetrieverResources []struct {
		Position     int     `json:"position"`
		DatasetID    string  `json:"dataset_id"`
		DatasetName  string  `json:"dataset_name"`
		DocumentID   string  `json:"document_id"`
		DocumentName string  `json:"document_name"`
		SegmentID    string  `json:"segment_id"`
		Score        float64 `json:"score"`
		Content      string  `json:"content"`
	} `json:"retriever_resources"`
}

// TTSEvent represents a TTS message event
type TTSEvent struct {
	Event          string `json:"event"`
	ConversationID string `json:"conversation_id"`
	MessageID      string `json:"message_id"`
	CreatedAt      int64  `json:"created_at"`
	TaskID         string `json:"task_id"`
	Audio          string `json:"audio"`
}

// TTSEndEvent represents the end of a TTS message event
type TTSEndEvent struct {
	Event          string `json:"event"`
	ConversationID string `json:"conversation_id"`
	MessageID      string `json:"message_id"`
	CreatedAt      int64  `json:"created_at"`
	TaskID         string `json:"task_id"`
	Audio          string `json:"audio"`
}
