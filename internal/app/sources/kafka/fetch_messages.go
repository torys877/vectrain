package kafka

import (
	"context"
	"errors"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"time"
)

func (k *Kafka) FetchOne(ctx context.Context) (any, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			msg, err := k.consumer.ReadMessage(100 * time.Millisecond)
			if err != nil {
				continue
			}
			return msg.Value, nil
		}
	}
}

func (k *Kafka) FetchBatch(ctx context.Context, size int) ([]any, error) {
	var res []any
	for i := 0; i < size; i++ {
		select {
		case <-ctx.Done():
			return res, ctx.Err()
		default:
			msg, err := k.consumer.ReadMessage(100 * time.Millisecond)
			if err != nil {
				var kafkaErr kafka.Error
				if errors.As(err, &kafkaErr) && kafkaErr.Code() == kafka.ErrTimedOut {
					i--
					continue
				}
				return res, err
			}
			res = append(res, msg.Value)
		}
	}
	return res, nil
}
