package ollama

import (
	"fmt"
	"github.com/torys877/vectrain/internal/config"
	"github.com/torys877/vectrain/pkg/types"
)

type Ollama struct {
	cfg      *OllamaConfig
	name     string
	model    string
	endpoint string
}

type OllamaConfig struct {
	Model    string `yaml:"model" validate:"required"`
	APIKey   string `yaml:"api_key" validate:"required"`
	Endpoint string `yaml:"endpoint" validate:"required,url"`
}

func NewOllamaClient(cfg types.TypedConfig) (*Ollama, error) {
	oc, err := config.ParseConfig[OllamaConfig](cfg)

	if err != nil {
		return nil, fmt.Errorf("invalid config, type: %s, err: %w", cfg.Type(), err)
	}

	return &Ollama{
		name:     cfg.Type(),
		model:    oc.Model,
		endpoint: oc.Endpoint,
		cfg:      oc,
	}, nil
}

func (o *Ollama) Name() string { return o.name }

var _ types.Embedder = &Ollama{}
