package qdrant

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"
	"strconv"
)

func (q *Qdrant) StoreOne(ctx context.Context, vector []float32, payload map[string]string) error {
	qdrantPayload, err := q.getPayload(payload)

	if err != nil {
		return fmt.Errorf("failed to get payload: %v", err)
	}

	// добавляем точку
	points := &qdrant.UpsertPoints{
		CollectionName: q.collectionName,
		Points: []*qdrant.PointStruct{
			{
				Id: &qdrant.PointId{
					PointIdOptions: &qdrant.PointId_Uuid{
						Uuid: uuid.New().String(),
					},
				},
				Vectors: &qdrant.Vectors{
					VectorsOptions: &qdrant.Vectors_Vector{
						Vector: &qdrant.Vector{
							Data: vector,
						},
					},
				},
				Payload: qdrantPayload,
			},
		},
	}

	_, err = q.client.GetPointsClient().Upsert(ctx, points)
	if err != nil {
		return fmt.Errorf("failed to upsert point: %v", err)
	}
	return nil
}

func (q *Qdrant) StoreBatch(ctx context.Context, vectors [][]float32, payloads []map[string]string) error {
	return nil
}

func (q *Qdrant) getPayload(payload map[string]string) (map[string]*qdrant.Value, error) {
	qdrantPayload := make(map[string]*qdrant.Value)

	for fieldName, fieldType := range q.payloadFields {
		v, ok := payload[fieldName]
		if !ok || v == "" {
			// сразу берем нулевое значение из карты
			qdrantPayload[fieldName] = zeroValues[fieldType]
			continue
		}

		switch fieldType {
		case QdrantFieldString:
			qdrantPayload[fieldName] = &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: v}}
		case QdrantFieldInt:
			i, err := strconv.Atoi(v)
			if err != nil {
				return nil, err
			}
			qdrantPayload[fieldName] = &qdrant.Value{Kind: &qdrant.Value_IntegerValue{IntegerValue: int64(i)}}
		case QdrantFieldFloat:
			f, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return nil, err
			}
			qdrantPayload[fieldName] = &qdrant.Value{Kind: &qdrant.Value_DoubleValue{DoubleValue: f}}
		case QdrantFieldBool:
			b, err := strconv.ParseBool(v)
			if err != nil {
				return nil, err
			}
			qdrantPayload[fieldName] = &qdrant.Value{Kind: &qdrant.Value_BoolValue{BoolValue: b}}
		}
	}

	return qdrantPayload, nil
}
