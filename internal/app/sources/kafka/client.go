package kafka

import (
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/torys877/vectrain/internal/config"
	"github.com/torys877/vectrain/pkg/types"
	"log"
)

type Kafka struct {
	name      string
	topicName string
	groupId   string
	consumer  *kafka.Consumer
	itemDatas map[int64]ItemData
}

type ItemData struct {
	Partition int32
	Offset    int64
}

func NewKafkaClient(config *config.KafkaConfig) (*Kafka, error) {
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": config.Endpoint,
		"group.id":          config.GroupID,
		"auto.offset.reset": config.Offset,
	})
	if err != nil {
		log.Fatalf("Failed to create consumer: %s", err)
		return nil, err
	}
	defer consumer.Close()

	topic := config.Topic
	if err = consumer.SubscribeTopics([]string{topic}, nil); err != nil {
		//log.Fatalf("Failed to subscribe to topic: %s", err)
		return nil, err
	}

	md, err := consumer.GetMetadata(&topic, false, 5000)
	if err != nil {
		log.Fatalf("Failed to get metadata: %v", err)
	}
	t, ok := md.Topics[topic]
	if !ok {
		//log.Fatalf("Topic %s does not exist", topic)
		return nil, fmt.Errorf("topic %s does not exist", topic)
	}

	var partitions []kafka.TopicPartition
	for _, p := range t.Partitions { // TODO make partition configurable
		partition := kafka.TopicPartition{
			Topic: &topic, Partition: p.ID, Offset: kafka.OffsetBeginning, // TODO make offset configurable
		}

		partitions = append(partitions, partition)
	}

	err = consumer.Assign(partitions)

	if err != nil {
		return nil, err
	}

	return &Kafka{
		name:      "kafka",
		topicName: topic,
		groupId:   config.GroupID,
		consumer:  consumer,
		itemDatas: make(map[int64]ItemData),
	}, nil
}

func (k *Kafka) Name() string {
	return k.name
}

func (k *Kafka) Close() error {
	if k.consumer != nil {
		return k.consumer.Close()
	}
	return nil
}

var _ types.Source = &Kafka{}
