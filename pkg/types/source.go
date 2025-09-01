package types

import "context"

type Source interface {
	FetchOne(ctx context.Context) (*Entity, error)
	FetchBatch(ctx context.Context, size int) ([]*Entity, error)
	AfterProcess(ctx context.Context, successMsgs []*Entity, errorMsgs []*ErrorEntity) error
	Name() string
	Close() error
}
