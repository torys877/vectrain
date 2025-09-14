package config

import (
	"fmt"
	"github.com/torys877/vectrain/pkg/types"
	"os"
	"time"

	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

//var validate = validator.New()

type PipelineConfig struct {
	//Mode                    string `yaml:"mode" validate:"required,oneof=performance reliability"`
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

type Config struct {
	App      AppConfig
	Source   types.TypedConfig `yaml:"source"`
	Embedder types.TypedConfig `yaml:"embedder"`
	Storage  types.TypedConfig `yaml:"storage"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	config := &Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse yaml: %w", err)
	}

	var validate = validator.New()
	if err := validate.Struct(config); err != nil {
		return nil, fmt.Errorf("invalid base config: %w", err)
	}

	err = prepareAppConfig(config)
	if err != nil {
		return nil, fmt.Errorf("invalid app config: %w", err)
	}

	return config, nil
}

func prepareAppConfig(cfg *Config) error {
	sourceTimeout, err := time.ParseDuration(cfg.App.Pipeline.SourceResponseTimeout)
	if err != nil {
		return fmt.Errorf("invalid source_response_timeout: %v", err)
	}
	cfg.App.Pipeline.SourceResponseTimeoutDuration = sourceTimeout

	storageTimeout, err := time.ParseDuration(cfg.App.Pipeline.StorageResponseTimeout)
	if err != nil {
		return fmt.Errorf("invalid storage_response_timeout: %v", err)
	}
	cfg.App.Pipeline.StorageResponseTimeoutDuration = storageTimeout

	embedderTimeout, err := time.ParseDuration(cfg.App.Pipeline.EmbedderResponseTimeout)
	if err != nil {
		return fmt.Errorf("invalid embedder_response_timeout: %v", err)
	}
	cfg.App.Pipeline.EmbedderResponseTimeoutDuration = embedderTimeout

	return nil
}

func ParseConfig[T any](cfg types.TypedConfig) (*T, error) {
	var k T
	data, err := yaml.Marshal(cfg.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config, type :%s, err: %w", cfg.Type(), err)
	}

	if err = yaml.Unmarshal(data, &k); err != nil {
		return nil, fmt.Errorf("failed to parse config, type :%s, err: %w", cfg.Type(), err)
	}

	var validate = validator.New()
	if err = validate.Struct(k); err != nil {
		return nil, fmt.Errorf("invalid config, type :%s, err: %w", cfg.Type(), err)
	}

	return &k, nil
}
