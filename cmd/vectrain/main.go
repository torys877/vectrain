package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/torys877/vectrain/internal/app"
	"github.com/torys877/vectrain/internal/infra/monitoring"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/torys877/vectrain/internal/config"
	routes "github.com/torys877/vectrain/internal/http"
)

func main() {
	fmt.Println(" === Vectrain === ")

	configPath := flag.String("config", "", "path to config file")
	metricsFlag := flag.Bool("metrics", false, "flag to enable metrics(prometheus)")
	metricsPort := flag.Int("metrics-port", 9090, "port for metrics, default: 9090")
	flag.Parse()
	fmt.Println("configPath:", *configPath)
	if *configPath == "" {
		fmt.Println("Error: --config argument is required")
		os.Exit(1)
	}

	monitoring.RunPrometheus(monitoring.PrometheusConfig{Active: *metricsFlag, Port: *metricsPort})

	appConfig, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	appPipeline, err := app.Pipeline(*appConfig)
	if err != nil {
		fmt.Printf("Pipeline not created, check configuration: %v\n", err)
		return
	}

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
}
