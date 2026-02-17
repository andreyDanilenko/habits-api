package main

import (
	"log"

	"backend/internal/app"
	"backend/internal/config"
)

// @title           Backend API
// @version         1.0
// @description     Application backend API
// @host            localhost:8080
// @BasePath        /api/v1
// @schemes         http
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	application, err := app.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}

	if err := application.Run(); err != nil {
		log.Fatalf("Failed to run application: %v", err)
	}
}
