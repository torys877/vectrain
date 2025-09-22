package types

import (
	"context"
	"io"
)

type Source interface {
	Name() string
	Connect() error
	Fetch(ctx context.Context, size int) ([]*Entity, error)
	BeforeProcessHook(ctx context.Context, entities []*Entity) error
	AfterProcessHook(ctx context.Context, entities []*Entity) error
	io.Closer
}
