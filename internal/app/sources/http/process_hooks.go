package http

import (
	"context"
	"github.com/torys877/vectrain/pkg/types"
)

func (h *HttpClient) BeforeProcessHook(ctx context.Context, msgs []*types.Entity) error {
	return nil
}

func (h *HttpClient) AfterProcessHook(ctx context.Context, msgs []*types.Entity) error {
	return nil
}
