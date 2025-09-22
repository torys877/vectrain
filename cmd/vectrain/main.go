package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/torys877/vectrain/internal/app"
	"github.com/torys877/vectrain/internal/config"
	routes "github.com/torys877/vectrain/internal/http"
	"github.com/torys877/vectrain/internal/infra/logger"
	"github.com/torys877/vectrain/internal/infra/monitoring"
	"go.uber.org/zap"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

func main() {
	defer logger.Close()
	logger.Info("=== Vectrain ===")

	// --- Load configuration ---

	appConfig, err := config.LoadConfig()
	if err != nil {
		logger.Error("failed to load config",
			zap.Error(err),
		)
		os.Exit(1)
	}

	// --- Start Prometheus monitoring ---
	monitoring.RunPrometheus(monitoring.PrometheusConfig{
		Active: appConfig.App.Monitoring.Enabled,
		Port:   appConfig.App.Monitoring.Port,
	})

	// --- Setup context for OS signals ---
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// --- Create pipeline ---
	appPipeline, err := app.Pipeline(*appConfig)
	if err != nil {
		logger.Error("pipeline creation failed, check configuration",
			zap.Error(err),
			zap.Any("config", appConfig),
		)
		os.Exit(1)
	}

	// --- Setup HTTP server ---
	e := echo.New()
	if err := routes.SetupRoutes(e, appConfig, appPipeline); err != nil {
		logger.Error("routes setup failed, check configuration",
			zap.Error(err),
			zap.Any("config", appConfig),
		)
		os.Exit(1)
	}

	// --- Channels for errors ---
	srvErrCh := make(chan error, 1)
	pipelineErrCh := make(chan error, 1)

	// --- Start HTTP server ---
	go func() {
		addr := ":" + strconv.Itoa(appConfig.App.Http.Port)
		if err := e.Start(addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", zap.Error(err))
			srvErrCh <- err
		}
		close(srvErrCh)
	}()

	// --- Start pipeline ---
	go func() {
		if err := appPipeline.Run(ctx); err != nil {
			pipelineErrCh <- fmt.Errorf("pipeline run error: %w", err)
		}
		close(pipelineErrCh)
	}()

	// --- Wait for signal or errors ---
	shutdownInitiated := false
	for !shutdownInitiated {
		select {
		case <-ctx.Done():
			logger.Info("signal received, shutting down...")
			shutdownInitiated = true
		case err, ok := <-srvErrCh:
			if ok && err != nil {
				logger.Error("server encountered an error", zap.Error(err))
			}
			shutdownInitiated = true
		case err, ok := <-pipelineErrCh:
			if ok && err != nil {
				logger.Error("pipeline encountered an error", zap.Error(err))
			}
			shutdownInitiated = true
		}
	}

	// --- Stop pipeline first ---
	logger.Info("pipeline stopping")
	appPipeline.Stop()

	// --- Shutdown HTTP server with timeout ---
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := e.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error", zap.Error(err))
	} else {
		logger.Info("HTTP server stopped")
	}

	logger.Info("application shutdown complete")
}
