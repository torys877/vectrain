package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"path/to/your/project/internal/config"
)

func main() {
	// Get config path from environment or use default
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = filepath.Join("config", "config.yaml")
	}

	// Load configuration
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Access specific configurations
	kafkaConfig, err := cfg.GetKafkaConfig()
	if err != nil {
		log.Fatalf("Failed to get Kafka config: %v", err)
	}
	fmt.Printf("Kafka topic: %s\n", kafkaConfig.Topic)

	openaiConfig, err := cfg.GetOpenAIConfig()
	if err != nil {
		log.Fatalf("Failed to get OpenAI config: %v", err)
	}
	fmt.Printf("OpenAI model: %s\n", openaiConfig.Model)

	qdrantConfig, err := cfg.GetQdrantConfig()
	if err != nil {
		log.Fatalf("Failed to get Qdrant config: %v", err)
	}
	fmt.Printf("Qdrant collection: %s (vector size: %d)\n",
		qdrantConfig.Collection, qdrantConfig.VectorSize)
}
