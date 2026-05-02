package main

import (
	"log"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"apex/event-worker/internal/handlers"
)

func main() {
	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Phase 1: health only. Kafka consumer loop added in Phase 3.
	e.GET("/health", handlers.HealthHandler())

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}
	log.Fatal(e.Start(":" + port))
}
