package qdrant

import (
	"fmt"
	"github.com/qdrant/go-client/qdrant"
	"github.com/torys877/vectrain/internal/config"
	"github.com/torys877/vectrain/pkg/types"
)

const (
	QdrantFieldString string = "string"
	QdrantFieldInt    string = "int"
	QdrantFieldFloat  string = "float"
	QdrantFieldBool   string = "bool"
)

var zeroValues = map[string]*qdrant.Value{
	QdrantFieldString: {Kind: &qdrant.Value_StringValue{StringValue: ""}},
	QdrantFieldInt:    {Kind: &qdrant.Value_IntegerValue{IntegerValue: 0}},
	QdrantFieldFloat:  {Kind: &qdrant.Value_DoubleValue{DoubleValue: 0}},
	QdrantFieldBool:   {Kind: &qdrant.Value_BoolValue{BoolValue: false}},
}

type Qdrant struct {
	client         *qdrant.Client
	cfg            *QdrantConfig
	name           string
	collectionName string
	payloadFields  map[string]string
}

type QdrantConfig struct {
	Host           string            `yaml:"host" validate:"required"`
	Port           int               `yaml:"port" validate:"required"`
	VectorSize     uint64            `yaml:"vector_size" validate:"required"`
	CollectionName string            `yaml:"collectionName" validate:"required"`
	Distance       string            `yaml:"distance" validate:"required,oneof=cosine euclid dot"`
	Fields         map[string]string `yaml:"fields" validate:"required"`
}

func NewQdrantClient(cfg types.TypedConfig) (*Qdrant, error) {
	qc, err := config.ParseConfig[QdrantConfig](cfg)

	if err != nil {
		return nil, fmt.Errorf("invalid config, type: %s, err: %w", cfg.Type(), err)
	}

	return &Qdrant{
		name:           "qdrant",
		collectionName: qc.CollectionName,
		payloadFields:  qc.Fields,
		cfg:            qc,
	}, nil
}
func (q *Qdrant) Connect() error {
	qdrantClientConfig := qdrant.Config{
		Host: q.cfg.Host,
		Port: q.cfg.Port,
	}

	client, err := qdrant.NewClient(&qdrantClientConfig)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}
	q.client = client

	return nil
}

func (k *Qdrant) Close() error {
	if k.client != nil {
		return k.client.Close()
	}
	return nil
}

func (q *Qdrant) Name() string {
	return q.name
}

var _ types.Storage = &Qdrant{}
