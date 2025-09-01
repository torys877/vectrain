package qdrant

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"
	"github.com/torys877/vectrain/pkg/types"
	"strconv"
)

func (q *Qdrant) StoreOne(ctx context.Context, vector *types.VectorEntity) error {
	qdrantPayload, err := q.getPayload(vector.Payload)

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
							Data: vector.Vector,
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

func (q *Qdrant) StoreBatch(ctx context.Context, vectors []*types.VectorEntity) error {
	if len(vectors) == 0 {
		return nil
	}

	points := make([]*qdrant.PointStruct, 0, len(vectors))

	for i, vector := range vectors {
		qdrantPayload, err := q.getPayload(vector.Payload)
		if err != nil {
			return fmt.Errorf("failed to get payload for item %d: %v", i, err)
		}

		point := &qdrant.PointStruct{
			Id: &qdrant.PointId{
				PointIdOptions: &qdrant.PointId_Uuid{
					Uuid: uuid.New().String(),
				},
			},
			Vectors: &qdrant.Vectors{
				VectorsOptions: &qdrant.Vectors_Vector{
					Vector: &qdrant.Vector{
						Data: vector.Vector,
					},
				},
			},
			Payload: qdrantPayload,
		}

		points = append(points, point)
	}

	upsertPoints := &qdrant.UpsertPoints{
		CollectionName: q.collectionName,
		Points:         points,
	}

	_, err := q.client.GetPointsClient().Upsert(ctx, upsertPoints) // TODO check dublicates, because they will be overwritten
	if err != nil {
		return fmt.Errorf("failed to upsert batch points: %v", err)
	}
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
