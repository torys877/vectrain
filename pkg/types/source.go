package types

import (
	"context"
	"io"
)

type Source interface {
	Name() string
	Connect() error
	FetchOne(ctx context.Context) (*Entity, error)
	FetchBatch(ctx context.Context, size int) ([]*Entity, error)
	AfterProcessHook(ctx context.Context, entities []*Entity) error
	io.Closer
}
