package types

import (
	"context"
	"io"
)

type Storage interface {
	Name() string
	Connect() error
	Store(ctx context.Context, vectors []*Entity) error
	io.Closer
}
