package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/torys877/vectrain/pkg/types"
	"time"
)

func (k *Kafka) Fetch(ctx context.Context, size int) ([]*types.Entity, error) {
	if size == 1 {
		entity, err := k.FetchOne(ctx)
		return []*types.Entity{entity}, err
	}

	return k.FetchBatch(ctx, size)
}

func (k *Kafka) FetchOne(ctx context.Context) (*types.Entity, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			msg, err := k.consumer.ReadMessage(100 * time.Millisecond)
			if err != nil {
				continue
			}

			var embedResp types.Entity
			if err := json.Unmarshal(msg.Value, &embedResp); err != nil {
				return nil, fmt.Errorf("error unmarshaling response: %v, body: %s", err, string(msg.Value))
			}

			k.itemDatas[embedResp.ID] = ItemData{Partition: msg.TopicPartition.Partition, Offset: int64(msg.TopicPartition.Offset)}

			return &embedResp, nil
		}
	}
}

func (k *Kafka) FetchBatch(ctx context.Context, size int) ([]*types.Entity, error) {
	var res []*types.Entity
	for i := 0; i < size; i++ {
		select {
		case <-ctx.Done():
			return res, ctx.Err()
		default:
			start := time.Now()
			msg, err := k.consumer.ReadMessage(500 * time.Millisecond)
			if err != nil {
				var kafkaErr kafka.Error
				if errors.As(err, &kafkaErr) && kafkaErr.Code() == kafka.ErrTimedOut {
					i--
					continue
				}
				return res, err
			}

			duration := time.Since(start)
			fmt.Printf("Source Request took %v\n", duration)

			var embedResp types.Entity
			if err := json.Unmarshal(msg.Value, &embedResp); err != nil {
				return nil, fmt.Errorf("error unmarshaling response: %v, body: %s", err, string(msg.Value))
			}

			if len(embedResp.ID) == 0 { // need to handle ID correctly
				embedResp.ID = embedResp.UUID
			}

			k.itemDatas[embedResp.ID] = ItemData{
				Partition: msg.TopicPartition.Partition,
				Offset:    int64(msg.TopicPartition.Offset),
			}

			res = append(res, &embedResp)
		}
	}
	return res, nil
}
