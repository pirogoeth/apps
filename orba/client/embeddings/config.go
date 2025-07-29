package embeddings

import "encoding/json"

type EmbeddingsProvider string

const (
	ProviderOpenAI EmbeddingsProvider = "openai"
)

type Config struct {
	Provider EmbeddingsProvider `json:"provider"`
	Config   json.RawMessage    `json:"config"`
}
