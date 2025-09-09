package config

import (
	"fmt"
	"os"
	"time"

	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

var validate = validator.New()

type Source interface {
	SourceType() string
}

type Embedder interface {
	EmbedderType() string
}

type Storage interface {
	StorageType() string
}

type KafkaConfig struct {
	Brokers []string `yaml:"brokers" validate:"required,min=1"`
	Topic   string   `yaml:"topic" validate:"required"`
	GroupID string   `yaml:"group_id" validate:"required"`
	//Endpoint string   `yaml:"endpoint" validate:"required,url"`
	Offset string `yaml:"offset" validate:"required,oneof=earliest latest"`
	//OffsetNumber string   `yaml:"offset_number"`
}

func (k KafkaConfig) SourceType() string { return "kafka" }

type PostgresSource struct {
	DSN string `yaml:"dsn" validate:"required"`
}

func (p PostgresSource) SourceType() string { return "postgres" }

type OllamaConfig struct {
	Model    string `yaml:"model" validate:"required"`
	APIKey   string `yaml:"api_key" validate:"required"`
	Endpoint string `yaml:"endpoint" validate:"required,url"`
}

func (o OllamaConfig) EmbedderType() string { return "ollama" }

// --- Storage ---

type QdrantConfig struct {
	URL            string            `yaml:"url" validate:"required,url"`
	Host           string            `yaml:"host" validate:"required"`
	Port           int               `yaml:"port" validate:"required"`
	CollectionName string            `yaml:"collectionName" validate:"required"`
	VectorSize     uint64            `yaml:"vector_size" validate:"required"`
	Distance       string            `yaml:"distance" validate:"required,oneof=cosine euclid dot"`
	Fields         map[string]string `yaml:"fields" validate:"required"`
}

func (q QdrantConfig) StorageType() string { return "qdrant" }

type PipelineConfig struct {
	Mode                    string `yaml:"mode" validate:"required,oneof=performance reliability"`
	SourceBatchSize         int    `yaml:"source_batch_size"`
	StorageBatchSize        int    `yaml:"storage_batch_size"`
	EmbedderWorkersCnt      int    `yaml:"embedder_workers_cnt"`
	SourceResponseTimeout   string `yaml:"source_response_timeout"`
	StorageResponseTimeout  string `yaml:"storage_response_timeout"`
	EmbedderResponseTimeout string `yaml:"embedder_response_timeout"`
	SkipEmbedderErrors      bool   `yaml:"skip_embedder_errors"`

	SourceResponseTimeoutDuration   time.Duration
	StorageResponseTimeoutDuration  time.Duration
	EmbedderResponseTimeoutDuration time.Duration
}
type AppConfig struct {
	Name     string `yaml:"name" validate:"required"`
	Pipeline *PipelineConfig
	Http     struct {
		Port int `yaml:"port"`
	}
	Logging struct {
		Level string `yaml:"level" validate:"required,oneof=debug info warn error"`
	}
	Monitoring struct {
		Enabled bool `yaml:"enabled"`
		Port    int  `yaml:"port"`
	}
	RetryPolicy struct {
		MaxRetries int    `yaml:"max_retries"`
		Backoff    string `yaml:"backoff"`
	}
}

// --- YAML обёртки ---
type rawConfig struct {
	App    *AppConfig
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
	App      *AppConfig
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
	cfg.App, err = prepareAppConfig(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid app config: %w", err)
	}

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

func prepareAppConfig(cfg *rawConfig) (*AppConfig, error) {

	sourceTimeout, err := time.ParseDuration(cfg.App.Pipeline.SourceResponseTimeout)
	if err != nil {
		return nil, fmt.Errorf("invalid source_response_timeout: %v", err)
	}
	cfg.App.Pipeline.SourceResponseTimeoutDuration = sourceTimeout

	storageTimeout, err := time.ParseDuration(cfg.App.Pipeline.StorageResponseTimeout)
	if err != nil {
		return nil, fmt.Errorf("invalid storage_response_timeout: %v", err)
	}
	cfg.App.Pipeline.StorageResponseTimeoutDuration = storageTimeout

	embedderTimeout, err := time.ParseDuration(cfg.App.Pipeline.EmbedderResponseTimeout)
	if err != nil {
		return nil, fmt.Errorf("invalid embedder_response_timeout: %v", err)
	}
	cfg.App.Pipeline.EmbedderResponseTimeoutDuration = embedderTimeout

	return cfg.App, nil
}
