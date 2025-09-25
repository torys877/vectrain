package handlers

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

type HealthResponse struct {
	Status  string                 `json:"status"`
	Details *HealthResponseDetails `json:"details,omitempty"`
}

type HealthResponseDetails struct {
	Application string `json:"application"`
	Database    string `json:"database"`
}

func HealthCheck() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, &HealthResponse{
			Status: "ok",
			Details: &HealthResponseDetails{
				Application: "healthy",
				Database:    "connected",
			},
		})
	}
}
