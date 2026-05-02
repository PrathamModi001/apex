package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

func HealthHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, HealthResponse{
			Status:  "ok",
			Service: "ingestor",
		})
	}
}
