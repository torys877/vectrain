package types

import "context"

type Source interface {
	FetchOne(ctx context.Context) (any, error)
	FetchBatch(ctx context.Context, size int) ([]any, error)
	Name() string
	Close() error
}

type SourceMessage struct {
	Message string
	Payload string
}
