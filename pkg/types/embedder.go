package types

import "context"

type Embedder interface {
	Embed(ctx context.Context, msg string) ([]float32, error)
	Name() string
}
