package types

import "context"

type Embedder interface {
	Name() string
	Embed(ctx context.Context, msg string) ([]float32, error)
}
