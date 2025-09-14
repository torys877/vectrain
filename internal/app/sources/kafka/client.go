package kafka

import (
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/torys877/vectrain/internal/config"
	"github.com/torys877/vectrain/pkg/types"
	"log"
	"strings"
)

type Kafka struct {
	name     string
	topic    string
	groupId  string
	consumer *kafka.Consumer
	//itemDatas map[[16]byte]ItemData
	itemDatas map[string]ItemData
	cfg       config.KafkaConfig
}

type ItemData struct {
	Partition int32
	Offset    int64
}

func NewKafkaClient(cfg config.Source) (*Kafka, error) {
	kafkaConfig, ok := cfg.(config.KafkaConfig)
	if !ok {
		return nil, fmt.Errorf("invalid config type: expected KafkaConfig")
	}

	return &Kafka{
		name:      "kafka",
		topic:     kafkaConfig.Topic,
		groupId:   kafkaConfig.GroupID,
		itemDatas: make(map[string]ItemData),
		cfg:       kafkaConfig,
	}, nil
}

func (k *Kafka) Connect() error {
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": strings.Join(k.cfg.Brokers, ","),
		"group.id":          k.cfg.GroupID,
		"auto.offset.reset": k.cfg.Offset,
	})
	if err != nil {
		log.Printf("failed to create consumer: %s", err)
		return err
	}
	k.consumer = consumer

	if err = consumer.SubscribeTopics([]string{k.topic}, nil); err != nil {
		log.Printf("Failed to subscribe to topic: %s", err)
		return err
	}

	md, err := consumer.GetMetadata(&k.topic, false, 5000) // FIXME make timeout configurable
	if err != nil {
		log.Printf("Failed to get metadata: %v", err)
	}
	t, ok := md.Topics[k.topic]
	if !ok {
		//log.Fatalf("Topic %s does not exist", topic)
		return fmt.Errorf("topic %s does not exist", k.topic)
	}

	var partitions []kafka.TopicPartition
	for _, p := range t.Partitions { // TODO make partition configurable
		partition := kafka.TopicPartition{
			Topic: &k.topic, Partition: p.ID, Offset: kafka.OffsetBeginning, // TODO make offset configurable
		}

		partitions = append(partitions, partition)
	}

	err = consumer.Assign(partitions)

	if err != nil {
		return err
	}

	return nil
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
