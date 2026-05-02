package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// HealthResponse is the JSON shape for GET /health.
type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

// HealthHandler returns a 200 OK with service identity.
// Used by docker-compose healthchecks and load balancers.
func HealthHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, HealthResponse{
			Status:  "ok",
			Service: "api-gateway",
		})
	}
}
