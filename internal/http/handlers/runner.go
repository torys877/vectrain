package handlers

import (
	"github.com/labstack/echo/v4"
	"github.com/torys877/vectrain/internal/app/pipeline"
	"github.com/torys877/vectrain/internal/config"

	"net/http"
)

type RunnerHandler struct {
	cfg      *config.Config
	pipeline *pipeline.Pipeline
}

func NewRunnerHandler(cfg *config.Config, pipelineApp *pipeline.Pipeline) (*RunnerHandler, error) {
	return &RunnerHandler{
		cfg:      cfg,
		pipeline: pipelineApp,
	}, nil
}

func (rh *RunnerHandler) Start(c echo.Context) error {
	rh.pipeline.Start()
	return c.JSON(http.StatusOK, Response{
		Status:     "Started",
		StatusCode: http.StatusOK,
		Data:       "Started",
	})
}

func (rh *RunnerHandler) Stop(c echo.Context) error {
	rh.pipeline.Stop()
	return c.JSON(http.StatusOK, Response{
		Status:     http.StatusText(http.StatusOK),
		StatusCode: http.StatusOK,
		Data:       "Stopped",
	})
}

func (rh *RunnerHandler) Configuration(c echo.Context) error {
	return c.JSON(http.StatusOK, Response{
		Status:     http.StatusText(http.StatusOK),
		StatusCode: http.StatusOK,
		Data:       rh.cfg,
	})
}
