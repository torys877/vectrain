package app

import (
	"fmt"
	"github.com/torys877/vectrain/internal/app/factory"
	"github.com/torys877/vectrain/internal/app/pipeline"
	"github.com/torys877/vectrain/internal/config"
)

func Pipeline(cfg config.Config) (*pipeline.Pipeline, error) {
	source, err := factory.NewSource(cfg.Source)
	if err != nil {
		return nil, fmt.Errorf("source error, err: %w", err)
	}

	storage, err := factory.NewStorage(cfg.Storage)
	if err != nil {
		return nil, fmt.Errorf("storage error, err: %w", err)
	}

	embedder, err := factory.NewEmbedder(cfg.Embedder)
	if err != nil {
		return nil, fmt.Errorf("embedder error, err: %w", err)
	}

	pl := pipeline.NewPipeline(
		pipeline.WithConfig(&cfg.App),
		pipeline.WithSource(source),
		pipeline.WithStorage(storage),
		pipeline.WithEmbedder(embedder),
	)

	return pl, nil
}
