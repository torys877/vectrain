package types

import (
	"context"
	"io"
)

type Storage interface {
	Name() string
	Connect() error
	StoreOne(ctx context.Context, vector *Entity) error
	StoreBatch(ctx context.Context, vectors []*Entity) error
	io.Closer
}
