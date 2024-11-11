package oai

import (
	"github.com/sashabaranov/go-openai"
)

type EmbeddingRequest struct {
	Input          any                            `json:"input"`
	Model          string                         `json:"model"`
	User           string                         `json:"user,omitempty"`
	EncodingFormat openai.EmbeddingEncodingFormat `json:"encoding_format,omitempty"`
	// Dimensions The number of dimensions the resulting output embeddings should have.
	// Only supported in text-embedding-3 and later models.
	Dimensions int `json:"dimensions,omitempty"`
}

type EmbeddingResponse struct {
	Object string             `json:"object"`
	Data   []openai.Embedding `json:"data"`
	Model  string             `json:"model"`
	Usage  openai.Usage       `json:"usage"`
}
