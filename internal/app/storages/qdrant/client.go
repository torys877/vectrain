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
	name           string
	collectionName string
	client         *qdrant.Client
	payloadFields  map[string]string
}

func NewQdrantClient(config *config.QdrantConfig) (*Qdrant, error) {
	qdrantConfig := qdrant.Config{
		Host: config.Host,
		Port: config.Port,
	}

	client, err := qdrant.NewClient(&qdrantConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %v", err)
	}

	return &Qdrant{
		name:           "qdrant",
		collectionName: config.CollectionName,
		client:         client,
		payloadFields:  config.Fields,
	}, nil
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
