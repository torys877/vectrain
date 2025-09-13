package pipeline

import (
	"github.com/torys877/vectrain/internal/config"
	"github.com/torys877/vectrain/pkg/types"
)

func WithSource(source types.Source) Option {
	return func(p *Pipeline) {
		p.source = source
	}
}

func WithEmbedder(embedder types.Embedder) Option {
	return func(p *Pipeline) {
		p.embedder = embedder
	}
}

func WithStorage(storage types.Storage) Option {
	return func(p *Pipeline) {
		p.storage = storage
	}
}

func WithConfig(cfg *config.AppConfig) Option {
	return func(p *Pipeline) {
		p.cfg = cfg
	}
}
