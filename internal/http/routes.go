package routes

import (
	"github.com/labstack/echo/v4"
	"github.com/torys877/vectrain/internal/app/pipeline"
	"github.com/torys877/vectrain/internal/config"
	"github.com/torys877/vectrain/internal/http/handlers"
)

func SetupRoutes(e *echo.Echo, cfg *config.Config, pipelineApp *pipeline.Pipeline) error {
	settingsHandler, err := handlers.NewRunnerHandler(cfg, pipelineApp)

	if err != nil {
		return err
	}

	api := e.Group("/api")
	{
		api.GET("/health", handlers.HealthCheck())
		api.POST("/start", settingsHandler.Start)
		api.POST("/stop", settingsHandler.Stop)
		api.POST("/configuration", settingsHandler.Configuration)
	}

	return nil
}
