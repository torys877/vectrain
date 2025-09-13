package storages

import (
	"fmt"
	vecQdrant "github.com/torys877/vectrain/internal/app/storages/qdrant"
	"github.com/torys877/vectrain/internal/config"
	"github.com/torys877/vectrain/internal/constants"
	"github.com/torys877/vectrain/pkg/types"
)

func Storage(storageConfig config.Storage) (types.Storage, error) {
	switch storageConfig.StorageType() {
	case constants.StorageQdrant:
		return vecQdrant.NewQdrantClient(storageConfig)
	default:
		return nil, fmt.Errorf("invalid storage type: %s", storageConfig.StorageType())
	}
}
