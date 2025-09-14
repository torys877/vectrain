package kafka

import (
	"context"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/torys877/vectrain/pkg/types"
)

func (k *Kafka) AfterProcessHook(ctx context.Context, msgs []*types.Entity) error {
	return k.commitOffsets(msgs)
	// TODO handle error messages
	//return nil
}

func (k *Kafka) commitOffsets(items []*types.Entity) error {
	return nil

	if len(items) == 0 || k.consumer == nil {
		return fmt.Errorf("no items to commit, or consumer is nil")
	}

	offsets := make([]kafka.TopicPartition, 0, len(items))
	//var itemsToRemove [][16]byte
	var itemsToRemove []string

	for _, item := range items {
		offsets = append(offsets, kafka.TopicPartition{
			Partition: k.itemDatas[item.ID].Partition,
			Offset:    kafka.Offset(k.itemDatas[item.ID].Offset + 1),
			Topic:     &k.topic,
		})
		itemsToRemove = append(itemsToRemove, item.ID)
	}

	_, err := k.consumer.CommitOffsets(offsets)
	if err != nil {
		return fmt.Errorf("failed to commit offsets: %v", err)
	}

	for _, id := range itemsToRemove {
		delete(k.itemDatas, id)
	}

	return nil
}
