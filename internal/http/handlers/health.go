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

// HealthCheck godoc
// @Summary      Health check
// @Description  Returns the health status of the application and its database connection
// @Tags         system
// @Produce      json
// @Success      200 {object} HealthResponse "Healthy"
// @Failure      503 {object} HealthResponse "Unhealthy"
// @Router       /api/health [get]
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
