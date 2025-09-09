package qdrant

import (
	"fmt"
	"github.com/qdrant/go-client/qdrant"
	"github.com/torys877/vectrain/internal/config"
	"github.com/torys877/vectrain/pkg/types"
	"io"
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
	name           string
	collectionName string
	client         *qdrant.Client
	payloadFields  map[string]string
	cfg            config.QdrantConfig
}

func NewQdrantClient(cfg config.Storage) (*Qdrant, error) {
	qdrantConfig, ok := cfg.(config.QdrantConfig)
	if !ok {
		return nil, fmt.Errorf("invalid config type: expected QdrantConfig")
	}

	return &Qdrant{
		name:           "qdrant",
		collectionName: qdrantConfig.CollectionName,
		payloadFields:  qdrantConfig.Fields,
		cfg:            qdrantConfig,
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
var _ io.Closer = &Qdrant{}
