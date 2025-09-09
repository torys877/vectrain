package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/torys877/vectrain/internal/app/embedders/ollama"
	"github.com/torys877/vectrain/internal/app/pipeline"
	"github.com/torys877/vectrain/internal/app/sources/kafka"
	"github.com/torys877/vectrain/internal/app/storages/qdrant"
	"github.com/torys877/vectrain/internal/config"
	routes "github.com/torys877/vectrain/internal/http"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"
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

	kafkaSource, err := kafka.NewKafkaClient(appConfig.Source)
	if err != nil {
		fmt.Println("Source Error:", err)
		os.Exit(1)
	}
	defer kafkaSource.Close()

	qdrantStorage, err := qdrant.NewQdrantClient(appConfig.Storage)
	if err != nil {
		fmt.Println("Storage Error:", err)
		os.Exit(1)
	}
	defer qdrantStorage.Close()

	ollamaEmbedder, err := ollama.NewOllamaClient(appConfig.Embedder)
	if err != nil {
		fmt.Println("Embedder Error:", err)
		os.Exit(1)
	}

	appPipeline := pipeline.NewPipeline(appConfig, kafkaSource, ollamaEmbedder, qdrantStorage)

	e := echo.New()
	err = routes.SetupRoutes(e, appConfig, appPipeline)
	if err != nil {
		fmt.Printf("Routes not set, check configuration: %v\n", err)
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	// Start server
	srvErrCh := make(chan error, 1)
	go func() {
		if err := e.Start(":" + strconv.Itoa(appConfig.App.Http.Port)); err != nil && !errors.Is(err, http.ErrServerClosed) {
			srvErrCh <- err
		}
		close(srvErrCh)
	}()

	// Start pipeline
	pipelineErrCh := make(chan error, 1)
	go func() {
		if err := appPipeline.Run(ctx); err != nil {
			pipelineErrCh <- err
		}
		close(pipelineErrCh)
	}()

	select {
	case <-ctx.Done():
		fmt.Println("Signal received, shutting down...")
	case err := <-srvErrCh:
		fmt.Println("Server error:", err)
	case err := <-pipelineErrCh:
		fmt.Println("Pipeline error:", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := e.Shutdown(shutdownCtx); err != nil {
		e.Logger.Fatal(err)
	}

	appPipeline.Stop()

	//go func() {
	//	if err := e.Start(":8080"); err != nil && !errors.Is(err, http.ErrServerClosed) {
	//		e.Logger.Fatal("shutting down the server")
	//	}
	//}()

	//err = appPipeline.Run(ctx)
	//if err != nil {
	//	fmt.Println("Pipeline Error:", err)
	//	os.Exit(1)
	//}

	//<-ctx.Done()
	//ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	//fmt.Println("Shutting down server...")
	//defer cancel()
	//if err := e.Shutdown(ctx); err != nil {
	//	e.Logger.Fatal(err)
	//}
}
