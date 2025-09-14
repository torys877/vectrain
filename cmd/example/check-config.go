package main

import (
	"flag"
	"fmt"
	"github.com/torys877/vectrain/internal/config"
	"os"
)

func main() {
	fmt.Println(" === Vectrain === ")

	configPath := flag.String("config", "", "path to config file")
	flag.Parse()
	fmt.Println("configPath:", *configPath)
	if *configPath == "" {
		fmt.Println("Error: --config argument is required")
		os.Exit(1)
	}

	appConfig, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	//fmt.Printf("Config: %+v\n", *appConfig)

	fmt.Printf("Config: %+v\n", *appConfig)
	fmt.Println(appConfig.App.Pipeline.EmbedderResponseTimeoutDuration)
	fmt.Println(appConfig.App.Pipeline.SourceResponseTimeoutDuration)
	fmt.Println(appConfig.App.Pipeline.SourceResponseTimeoutDuration)
	fmt.Printf("Config: %+v\n", *appConfig.App.Pipeline)
}
