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
		fmt.Println("Source Error:", err)
		return nil, err
	}

	storage, err := factory.NewStorage(cfg.Storage)
	if err != nil {
		fmt.Println("Storage Error:", err)
		return nil, err
	}

	embedder, err := factory.NewEmbedder(cfg.Embedder)
	if err != nil {
		fmt.Println("Embedder Error:", err)
		return nil, err
	}

	pl := pipeline.NewPipeline(
		pipeline.WithConfig(&cfg.App),
		pipeline.WithSource(source),
		pipeline.WithStorage(storage),
		pipeline.WithEmbedder(embedder),
	)

	return pl, nil
}
