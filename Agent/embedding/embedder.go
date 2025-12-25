package embedding

import (
	"context"
)

type Embedder interface {
	// Embed converts text to vector
	Embed(ctx context.Context, text string) ([]float32, error)

	// BatchEmbed converts multiple texts to vectors in batch
	BatchEmbed(ctx context.Context, texts []string) ([][]float32, error)

	// GetModelName returns the model name
	GetModelName() string

	// GetDimensions returns the vector dimensions
	GetDimensions() int

	// GetModelID returns the model ID
	GetModelID() string

	EmbedderPooler
}

type EmbedderPooler interface {
	BatchEmbedWithPool(ctx context.Context, model Embedder, texts []string) ([][]float32, error)
}

type EmbedderType string

// Config represents the embedder configuration
type Config struct {
	Source               string `json:"source"`
	BaseURL              string `json:"base_url"`
	ModelName            string `json:"model_name"`
	APIKey               string `json:"api_key"`
	TruncatePromptTokens int    `json:"truncate_prompt_tokens"` //截断的长度
	Dimensions           int    `json:"dimensions"`             //嵌入维度
	ModelID              string `json:"model_id"`
}

func NewEmbedder(config Config, pooler EmbedderPooler) (Embedder, error) {
	var embedder Embedder
	var err error
	embedder, err = NewOpenAIEmbedder(config.APIKey,
		config.BaseURL,
		config.ModelName,
		config.TruncatePromptTokens,
		config.Dimensions,
		config.ModelID,
		pooler)
	if err != nil {
		return nil, err
	}
	return embedder, err
}
