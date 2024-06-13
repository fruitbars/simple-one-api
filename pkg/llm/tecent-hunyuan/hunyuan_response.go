package tecent_hunyuan

type HunYuannResponseError struct {
	Code    string `json:"Code,omitempty"`
	Message string `json:"Message,omitempty"`
}

type HunYuanResponse struct {
	Response struct {
		RequestID string `json:"RequestId"`
		Note      string `json:"Note"`
		Choices   []struct {
			Message struct {
				Role    string `json:"Role"`
				Content string `json:"Content"`
			} `json:"Message"`
			FinishReason string `json:"FinishReason"`
		} `json:"Choices"`
		Created int    `json:"Created"`
		ID      string `json:"Id"`
		Usage   struct {
			PromptTokens     int `json:"PromptTokens"`
			CompletionTokens int `json:"CompletionTokens"`
			TotalTokens      int `json:"TotalTokens"`
		} `json:"Usage"`
		Error *HunYuannResponseError `json:"Error"`
	} `json:"Response"`
}

type StreamResponse struct {
	Note    string `json:"Note"`
	Choices []struct {
		Delta struct {
			Role    string `json:"Role"`
			Content string `json:"Content"`
		} `json:"Delta"`
		FinishReason string `json:"FinishReason"`
	} `json:"Choices"`
	Created int64  `json:"Created"`
	ID      string `json:"Id"`
	Usage   struct {
		PromptTokens     int `json:"PromptTokens"`
		CompletionTokens int `json:"CompletionTokens"`
		TotalTokens      int `json:"TotalTokens"`
	} `json:"Usage"`
}
