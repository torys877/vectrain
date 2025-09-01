package qdrant

import "github.com/torys877/vectrain/pkg/types"

type Qdrant struct {
	name string
}

func (q *Qdrant) Store() {}

func (q *Qdrant) Name() string {
	return q.name
}

var _ types.Storage = &Qdrant{}
