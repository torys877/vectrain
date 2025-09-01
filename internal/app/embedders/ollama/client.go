package ollama

import (
	"github.com/torys877/vectrain/internal/config"
	"github.com/torys877/vectrain/pkg/types"
	"time"
)

type Ollama struct {
	name                       string
	model                      string
	endpoint                   string
	MaxResponseTimeoutDuration time.Duration
}

func NewOllamaClient(config *config.OllamaConfig) (*Ollama, error) {
	return &Ollama{
		name:                       "ollama",
		model:                      config.Model,
		endpoint:                   config.Endpoint,
		MaxResponseTimeoutDuration: config.MaxResponseTimeoutDuration,
	}, nil
}

func (o *Ollama) Name() string { return o.name }

var _ types.Embedder = &Ollama{}
