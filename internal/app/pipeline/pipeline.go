package pipeline

import "github.com/torys877/vectrain/pkg/types"

type Pipeline struct {
	Source   *types.Source
	Embedder *types.Embedder
	Storage  *types.Storage
}

func NewPipeline() *Pipeline {
	return &Pipeline{}
}

func (p *Pipeline) Run() {}
