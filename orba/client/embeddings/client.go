package embeddings

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/openai/openai-go"
	openaiOption "github.com/openai/openai-go/option"
)

// Client is an interface defining the embeddings client.
type Client interface {
	Init(cfg json.RawMessage) error
	Generate(context.Context, string) ([]float64, error)
}

var _ Client = (*OpenAIEmbeddingsClient)(nil)

type OpenAIEmbeddingsConfig struct {
	BaseURL url.URL `json:"base_url"`
	APIKey  string  `json:"api_key"`
	Model   string  `json:"model"`
}

type OpenAIEmbeddingsClient struct {
	cfg *OpenAIEmbeddingsConfig
	c   openai.Client
}

func (c *OpenAIEmbeddingsClient) Init(cfgJson json.RawMessage) error {
	c.cfg = new(OpenAIEmbeddingsConfig)
	if err := json.Unmarshal(cfgJson, c.cfg); err != nil {
		return fmt.Errorf("could not load openai embeddings config: %w", err)
	}

	clientOptions := []openaiOption.RequestOption{}
	if baseUrl := c.cfg.BaseURL.String(); baseUrl != "" {
		clientOptions = append(clientOptions, openaiOption.WithBaseURL(baseUrl))
	}

	if apiKey := c.cfg.APIKey; apiKey != "" {
		clientOptions = append(clientOptions, openaiOption.WithAPIKey(apiKey))
	}

	c.c = openai.NewClient(clientOptions...)

	return nil
}

// Generate uses the OpenAI API to generate embeddings for the given text
func (c *OpenAIEmbeddingsClient) Generate(ctx context.Context, text string) ([]float64, error) {
	resp, err := c.c.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Input: openai.EmbeddingNewParamsInputUnion{
			OfString: openai.String(text),
		},
		Model: c.cfg.Model,
	})
	if err != nil {
		return []float64{}, fmt.Errorf("could not get embeddings: %w", err)
	}

	// TODO: What do if len(resp.Data) > 1?
	return resp.Data[0].Embedding, nil
}
