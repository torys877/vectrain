package main

import (
	"fmt"
	"log"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

func main() {
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": "127.0.0.1:9092",
		"group.id":          "movie-group",
		"auto.offset.reset": "earliest",
		//"debug":             "all",
	})
	if err != nil {
		log.Fatalf("Failed to create consumer: %s", err)
	}
	defer consumer.Close()

	topic := "movie-topic"

	// После создания консьюмера
	log.Printf("Consumer connected to %s", consumer.String())

	if err := consumer.SubscribeTopics([]string{topic}, nil); err != nil {
		log.Fatalf("Failed to subscribe to topic: %s", err)
	}

	md, err := consumer.GetMetadata(&topic, false, 5000)
	if err != nil {
		log.Fatalf("Failed to get metadata: %v", err)
	}
	t, ok := md.Topics[topic]
	if !ok {
		log.Fatalf("Topic %s does not exist", topic)
	}

	// assign partitions
	var partitions []kafka.TopicPartition
	for _, p := range t.Partitions {
		fmt.Printf("Partition ID: %d, Leader: %d, Replicas: %v, ISR: %v\n",
			p.ID, p.Leader, p.Replicas, p.Isrs)
		partition := kafka.TopicPartition{
			Topic: &topic, Partition: p.ID, Offset: kafka.OffsetBeginning,
		}

		partitions = append(partitions, partition)
	}

	err = consumer.Assign(partitions)

	if err != nil {
		fmt.Println("Cannot Assign Partitions")
		return
	}
	// При подписке на топик
	log.Printf("Subscribed to topic: %s", topic)

	fmt.Println("Consumer started, waiting for messages...")

	for {
		fmt.Println("In Circular Loop")
		msg, err := consumer.ReadMessage(-1)
		if err != nil {
			// Ошибка при чтении, например, таймаут
			log.Printf("Consumer error: %v (%v)\n", err, msg)
			continue
		}
		fmt.Printf("Received message: %s from partition %d\n", string(msg.Value), msg.TopicPartition.Partition)
	}
}
