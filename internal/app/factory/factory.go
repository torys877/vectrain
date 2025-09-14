package factory

import (
	"fmt"
	"github.com/torys877/vectrain/internal/app/embedders/ollama"
	"github.com/torys877/vectrain/internal/app/sources/kafka"
	vecQdrant "github.com/torys877/vectrain/internal/app/storages/qdrant"
	"github.com/torys877/vectrain/internal/constants"
	"github.com/torys877/vectrain/pkg/types"
)

func NewSource(cfg types.TypedConfig) (types.Source, error) {
	switch cfg.Type() {
	case constants.SourceKafka:
		return kafka.NewKafkaClient(cfg)
	default:
		return nil, fmt.Errorf("invalid source type: %s", cfg.Type())
	}
}
func NewEmbedder(cfg types.TypedConfig) (types.Embedder, error) {
	switch cfg.Type() {
	case constants.EmbedderOllama:
		return ollama.NewOllamaClient(cfg)
	default:
		return nil, fmt.Errorf("invalid embedder type: %s", cfg.Type())
	}
}

func NewStorage(cfg types.TypedConfig) (types.Storage, error) {
	switch cfg.Type() {
	case constants.StorageQdrant:
		return vecQdrant.NewQdrantClient(cfg)
	default:
		return nil, fmt.Errorf("invalid storage type: %s", cfg.Type())
	}
}
