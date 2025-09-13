package embedders

import (
	"fmt"
	"github.com/torys877/vectrain/internal/app/embedders/ollama"
	"github.com/torys877/vectrain/internal/config"
	"github.com/torys877/vectrain/internal/constants"
	"github.com/torys877/vectrain/pkg/types"
)

func Embedder(embedderConfig config.Embedder) (types.Embedder, error) {
	switch embedderConfig.EmbedderType() {
	case constants.EmbedderOllama:
		return ollama.NewOllamaClient(embedderConfig)
	default:
		return nil, fmt.Errorf("invalid embedder type: %s", embedderConfig.EmbedderType())
	}
}
