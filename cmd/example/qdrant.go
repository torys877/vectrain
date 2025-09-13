package main

import (
	"context"
	"crypto/tls"
	"github.com/google/uuid"
	"log"

	qdrant "github.com/qdrant/go-client/qdrant"
)

func main() {
	// создаём конфиг для подключения
	config := qdrant.Config{
		Host:      "localhost",
		Port:      6334,
		UseTLS:    false,
		TLSConfig: &tls.Config{},
	}

	client, err := qdrant.NewClient(&config)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}

	ctx := context.Background()
	collectionName := "movies"

	// Создаем запрос на создание коллекции
	createCollection := &qdrant.CreateCollection{
		CollectionName: collectionName,
		VectorsConfig: &qdrant.VectorsConfig{
			Config: &qdrant.VectorsConfig_Params{
				Params: &qdrant.VectorParams{
					Size:     4,
					Distance: qdrant.Distance_Cosine,
				},
			},
		},
	}

	err = client.CreateCollection(ctx, createCollection)
	if err != nil {
		log.Println("collection might already exist:", err)
	}

	// создаём вектор и payload
	vector := []float32{0.1, 0.2, 0.3, 0.4}

	// Создаем payload с правильным типом Value
	payload := make(map[string]*qdrant.Value)
	payload["title"] = &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: "My Movie"}}
	payload["year"] = &qdrant.Value{Kind: &qdrant.Value_IntegerValue{IntegerValue: 2025}}
	payload["genre"] = &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: "comedy"}}

	// добавляем точку
	points := &qdrant.UpsertPoints{
		CollectionName: collectionName,
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
				Payload: payload,
			},
		},
	}

	_, err = client.GetPointsClient().Upsert(ctx, points)
	if err != nil {
		log.Fatalf("failed to upsert point: %v", err)
	}

	log.Println("Point inserted successfully")
}
