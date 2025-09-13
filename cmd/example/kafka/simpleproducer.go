package main

import (
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"log"
	"time"
)

func main() {
	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": "127.0.0.1:9092",
	})
	if err != nil {
		log.Fatalf("Failed to create producer: %s", err)
	}
	defer producer.Close()

	topic := "raw-data"

	for id := 1; id <= 200; id++ {
		message := fmt.Sprintf(
			"it works or not",
		)

		err := producer.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
			Value:          []byte(message),
		}, nil)
		if err != nil {
			log.Printf("Failed to produce message: %s", err)
		}

		time.Sleep(100 * time.Millisecond)
	}

	// wait all messages to be sent
	producer.Flush(5 * 1000)
	fmt.Println("All messages sent")
}
