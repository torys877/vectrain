package http

import (
	"context"
	"github.com/torys877/vectrain/pkg/types"
)

func (h *HttpClient) Fetch(ctx context.Context, size int) ([]*types.Entity, error) {
	var batch []*types.Entity
	for i := 0; i < size; i++ {
		select {
		case <-ctx.Done():
			return batch, ctx.Err()
		case e := <-h.entities:
			batch = append(batch, e)
		default:
			return batch, nil
		}
	}
	return batch, nil
}
