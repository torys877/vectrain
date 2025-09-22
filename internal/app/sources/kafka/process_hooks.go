package kafka

import (
	"context"
	"github.com/torys877/vectrain/pkg/types"
)

func (k *Kafka) BeforeProcessHook(ctx context.Context, msgs []*types.Entity) error {
	return nil
}

func (k *Kafka) AfterProcessHook(ctx context.Context, msgs []*types.Entity) error {
	return nil
}
