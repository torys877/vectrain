package http

import (
	"context"
	"errors"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/torys877/vectrain/internal/config"
	"github.com/torys877/vectrain/pkg/types"
	"net/http"
	"time"
)

type HttpClient struct {
	client       *echo.Echo
	cfg          *HttpConfig
	name         string
	entitiesSize int
	entities     chan *types.Entity
}
type HttpConfig struct {
	Port       string `yaml:"port" validate:"required"`
	RequestCap int    `yaml:"request_cap" validate:"required"`
}

func NewHttpClient(cfg types.TypedConfig) (*HttpClient, error) {
	hc, err := config.ParseConfig[HttpConfig](cfg)
	if err != nil {
		return nil, fmt.Errorf("invalid config, type: %s, err: %w", cfg.Type(), err)
	}

	return &HttpClient{
		name:     cfg.Type(),
		client:   echo.New(),
		cfg:      hc,
		entities: make(chan *types.Entity, hc.RequestCap),
	}, nil
}

func (h *HttpClient) Connect() error {
	h.setupRoutes()

	// Start server
	srvErrCh := make(chan error, 1)
	go func() {
		if err := h.client.Start(":" + h.cfg.Port); err != nil && !errors.Is(err, http.ErrServerClosed) {
			srvErrCh <- err
		}
		close(srvErrCh)
	}()

	return nil
}

func (h *HttpClient) Name() string { return h.name }

func (h *HttpClient) Close() error {
	if h.client != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := h.client.Shutdown(shutdownCtx); err != nil {
			h.client.Logger.Fatal(err)
		}

		return h.client.Close()
	}

	return nil
}

func (h *HttpClient) setupRoutes() {
	api := h.client.Group("/source")
	{
		api.POST("/send", h.sendRoute)
	}
}

func (h *HttpClient) sendRoute(c echo.Context) error {
	var entity types.Entity
	if err := c.Bind(&entity); err != nil {
		errorMessage := fmt.Sprintf("Incorrect Request, err: %v", err)
		c.Logger().Error(errorMessage)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error":   "bad_request",
			"message": errorMessage,
		})
	}

	select {
	case h.entities <- &entity:
		return c.JSON(http.StatusAccepted, map[string]string{
			"status": "queued",
		})
	default:
		return c.JSON(http.StatusTooManyRequests, map[string]string{
			"error":   "queue_full",
			"message": "The processing queue is full. Please try again later.",
		})
	}
}

var _ types.Source = &HttpClient{}
