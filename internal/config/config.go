package config

import (
	"fmt"
	"os"

	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

var validate = validator.New()

// --- Source ---
type Source interface {
	SourceType() string
}

type KafkaConfig struct {
	Brokers []string `yaml:"brokers" validate:"required,min=1"`
	Topic   string   `yaml:"topic" validate:"required"`
	GroupID string   `yaml:"group_id" validate:"required"`
}

func (k KafkaConfig) SourceType() string { return "kafka" }

type PostgresSource struct {
	DSN string `yaml:"dsn" validate:"required"`
}

func (p PostgresSource) SourceType() string { return "postgres" }

// --- Embedder ---
type Embedder interface {
	EmbedderType() string
}

type OllamaConfig struct {
	Model  string `yaml:"model" validate:"required"`
	APIKey string `yaml:"api_key" validate:"required"`
}

func (o OllamaConfig) EmbedderType() string { return "openai" }

// --- Storage ---
type Storage interface {
	StorageType() string
}

type QdrantConfig struct {
	URL        string `yaml:"url" validate:"required,url"`
	Collection string `yaml:"collection" validate:"required"`
	VectorSize int    `yaml:"vector_size" validate:"required"`
	Distance   string `yaml:"distance" validate:"required,oneof=cosine euclid dot"`
}

func (q QdrantConfig) StorageType() string { return "qdrant" }

// --- YAML обёртки ---
type rawConfig struct {
	Source struct {
		Type   string      `yaml:"type" validate:"required"`
		Config interface{} `yaml:"config" validate:"required"`
	} `yaml:"source"`

	Embedder struct {
		Type   string      `yaml:"type" validate:"required"`
		Config interface{} `yaml:"config" validate:"required"`
	} `yaml:"embedder"`

	Storage struct {
		Type   string      `yaml:"type" validate:"required"`
		Config interface{} `yaml:"config" validate:"required"`
	} `yaml:"storage"`
}

// --- Основной конфиг ---
type Config struct {
	Source   Source
	Embedder Embedder
	Storage  Storage
}

// --- LoadConfig ---
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	raw := &rawConfig{}
	if err := yaml.Unmarshal(data, raw); err != nil {
		return nil, fmt.Errorf("failed to parse yaml: %w", err)
	}

	if err := validate.Struct(raw); err != nil {
		return nil, fmt.Errorf("invalid base config: %w", err)
	}

	cfg := &Config{}

	// --- Source ---
	switch raw.Source.Type {
	case "kafka":
		b, _ := yaml.Marshal(raw.Source.Config)
		var k KafkaConfig
		if err := yaml.Unmarshal(b, &k); err != nil {
			return nil, fmt.Errorf("failed to parse kafka config: %w", err)
		}
		if err := validate.Struct(k); err != nil {
			return nil, fmt.Errorf("invalid kafka config: %w", err)
		}
		cfg.Source = k
	case "postgres":
		b, _ := yaml.Marshal(raw.Source.Config)
		var p PostgresSource
		if err := yaml.Unmarshal(b, &p); err != nil {
			return nil, fmt.Errorf("failed to parse postgres config: %w", err)
		}
		if err := validate.Struct(p); err != nil {
			return nil, fmt.Errorf("invalid postgres config: %w", err)
		}
		cfg.Source = p
	default:
		return nil, fmt.Errorf("unsupported source type: %s", raw.Source.Type)
	}

	// --- Embedder ---
	switch raw.Embedder.Type {
	case "ollama":
		b, _ := yaml.Marshal(raw.Embedder.Config)
		var o OllamaConfig
		if err := yaml.Unmarshal(b, &o); err != nil {
			return nil, fmt.Errorf("failed to parse openai config: %w", err)
		}
		if err := validate.Struct(o); err != nil {
			return nil, fmt.Errorf("invalid openai config: %w", err)
		}
		cfg.Embedder = o
	default:
		return nil, fmt.Errorf("unsupported embedder type: %s", raw.Embedder.Type)
	}

	// --- Storage ---
	switch raw.Storage.Type {
	case "qdrant":
		b, _ := yaml.Marshal(raw.Storage.Config)
		var q QdrantConfig
		if err := yaml.Unmarshal(b, &q); err != nil {
			return nil, fmt.Errorf("failed to parse qdrant config: %w", err)
		}
		if err := validate.Struct(q); err != nil {
			return nil, fmt.Errorf("invalid qdrant config: %w", err)
		}
		cfg.Storage = q
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", raw.Storage.Type)
	}

	return cfg, nil
}
