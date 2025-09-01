package types

import "context"

type Storage interface {
	Name() string
	StoreOne(ctx context.Context, vector []float32, payload map[string]string) error
	StoreBatch(ctx context.Context, vectors [][]float32, payloads []map[string]string) error
	Close() error
}

type StorageMessage struct {
	Vector  []float32 // Vector holds a slice of float32 values representing numerical data or coordinates.
	Payload string    // JSON with payload for filters
}
