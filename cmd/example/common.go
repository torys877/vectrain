package main

import (
	"flag"
	"fmt"
	"github.com/torys877/vectrain/internal/config"
	"os"
)

// LoadConfig принимает путь к файлу и возвращает ошибку,
// если файл не найден или не читается.
func LoadConfig(path string) error {
	// Просто проверим, существует ли файл
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	// Здесь могла бы быть логика парсинга конфига
	fmt.Println("Config loaded from:", path)
	return nil
}

func main() {
	// Парсим аргумент --config
	configPath := flag.String("config", "", "path to config file")
	flag.Parse()

	if *configPath == "" {
		fmt.Println("Error: --config argument is required")
		os.Exit(1)
	}

	appConfig, err := config.LoadConfig(*configPath)

	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	fmt.Println(appConfig)
}
