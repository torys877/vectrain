package types

import "context"

type Storage interface {
	Name() string
	StoreOne(ctx context.Context, vector *VectorEntity) error
	StoreBatch(ctx context.Context, vectors []*VectorEntity) error
	Close() error
}

type VectorEntity struct {
	Entity
	Vector []float32
}
