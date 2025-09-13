package sources

import (
	"fmt"
	"github.com/torys877/vectrain/internal/app/sources/kafka"
	"github.com/torys877/vectrain/internal/config"
	"github.com/torys877/vectrain/internal/constants"
	"github.com/torys877/vectrain/pkg/types"
)

func Source(sourceConfig config.Source) (types.Source, error) {
	switch sourceConfig.SourceType() {
	case constants.SourceKafka:
		return kafka.NewKafkaClient(sourceConfig)
	default:
		return nil, fmt.Errorf("invalid source type: %s", sourceConfig.SourceType())
	}
}
