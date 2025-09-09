package ollama

import (
	"fmt"
	"github.com/torys877/vectrain/internal/config"
	"github.com/torys877/vectrain/pkg/types"
)

type Ollama struct {
	name     string
	model    string
	endpoint string
	cfg      config.OllamaConfig
}

func NewOllamaClient(cfg config.Embedder) (*Ollama, error) {
	ollamaConfig, ok := cfg.(config.OllamaConfig)
	if !ok {
		return nil, fmt.Errorf("invalid config type: expected OllamaConfig")
	}

	return &Ollama{
		name:     ollamaConfig.EmbedderType(),
		model:    ollamaConfig.Model,
		endpoint: ollamaConfig.Endpoint,
		cfg:      ollamaConfig,
	}, nil
}

func (o *Ollama) Name() string { return o.name }

var _ types.Embedder = &Ollama{}
