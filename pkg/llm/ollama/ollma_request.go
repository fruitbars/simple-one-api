package ollama

type ChatRequest struct {
	Model     string               `json:"model"`
	Messages  []Message            `json:"messages"`
	Stream    bool                 `json:"stream"`
	Format    string               `json:"format,omitempty"`
	Options   AdvancedModelOptions `json:"options,omitempty"`
	KeepAlive string               `json:"keep_alive,omitempty"`
}

type Message struct {
	Role    string   `json:"role"`
	Content string   `json:"content"`
	Images  []string `json:"images,omitempty"`
}

type AdvancedModelOptions struct {
	Temperature   float32 `json:"temperature,omitempty"`
	Seed          int     `json:"seed,omitempty"`
	Mirostat      int     `json:"mirostat,omitempty"`
	MirostatEta   float32 `json:"mirostat_eta,omitempty"`
	MirostatTau   float32 `json:"mirostat_tau,omitempty"`
	NumCtx        int     `json:"num_ctx,omitempty"`
	RepeatLastN   int     `json:"repeat_last_n,omitempty"`
	RepeatPenalty float32 `json:"repeat_penalty,omitempty"`
	Stop          string  `json:"stop,omitempty"`
	TfsZ          float32 `json:"tfs_z,omitempty"`
	NumPredict    int     `json:"num_predict,omitempty"`
	TopK          int     `json:"top_k,omitempty"`
	TopP          float32 `json:"top_p,omitempty"`
}
