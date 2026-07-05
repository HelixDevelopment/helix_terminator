package main

import (
	"log"
	"os"

	"github.com/helixdevelopment/health-service/internal/server"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// TODO: initialize config, logger, DB, tracer
	srv := server.New()

	log.Printf("health-service starting on port %s", port)
	if err := srv.Run(":" + port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
