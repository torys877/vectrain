package qdrant

import (
	"context"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"
	"github.com/torys877/vectrain/pkg/types"
)

func (q *Qdrant) Store(ctx context.Context, vectors []*types.Entity) error {
	if len(vectors) == 0 {
		return nil
	}

	_, err := q.checkCollection()
	if err != nil {
		return err
	}

	points := make([]*qdrant.PointStruct, 0, len(vectors))

	for i, vector := range vectors {
		qdrantPayload, err := q.getPayload(vector.Payload)
		if err != nil {
			return fmt.Errorf("failed to get payload for item %d: %v", i, err)
		}

		point := &qdrant.PointStruct{
			Id:      qdrant.NewID(uuid.New().String()),
			Vectors: qdrant.NewVectorsDense(vector.Vector),
			Payload: qdrantPayload,
		}

		points = append(points, point)
	}

	upsertPoints := &qdrant.UpsertPoints{
		CollectionName: q.collectionName,
		Points:         points,
	}

	_, err = q.client.Upsert(ctx, upsertPoints) // TODO check res status, check duplicates, because they will be overwritten

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
			qdrantPayload[fieldName] = qdrant.NewValueString(v)
		case QdrantFieldInt:
			i, err := strconv.Atoi(v)
			if err != nil {
				return nil, err
			}
			qdrantPayload[fieldName] = qdrant.NewValueInt(int64(i))
		case QdrantFieldFloat:
			f, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return nil, err
			}
			qdrantPayload[fieldName] = qdrant.NewValueDouble(f)
		case QdrantFieldBool:
			b, err := strconv.ParseBool(v)
			if err != nil {
				return nil, err
			}
			qdrantPayload[fieldName] = qdrant.NewValueBool(b)
		}
	}

	return qdrantPayload, nil
}

func (q *Qdrant) checkCollection() (bool, error) {
	ctx := context.Background()
	collectionExists, err := q.client.CollectionExists(ctx, q.collectionName)
	if err != nil {
		return false, fmt.Errorf("failed to check collection: %v", err)
	}

	if !collectionExists {
		createCollection := &qdrant.CreateCollection{
			CollectionName: q.collectionName,
			VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
				Size:     q.cfg.VectorSize,
				Distance: qdrant.Distance_Cosine,
			}),
		}

		err = q.client.CreateCollection(ctx, createCollection)
		if err != nil {
			return false, fmt.Errorf("collection did not created: %s", err)
		}
	}

	return true, nil
}
