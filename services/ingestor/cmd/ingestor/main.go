package main

import (
	"log"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"apex/ingestor/internal/handlers"
)

func main() {
	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Phase 1: health only. Gmail polling and Telegram webhook added in Phase 2.
	e.GET("/health", handlers.HealthHandler())

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	log.Fatal(e.Start(":" + port))
}
