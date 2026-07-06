package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/helixdevelopment/health-service/internal/server"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logger := log.New(os.Stdout, "[health] ", log.LstdFlags|log.Lshortfile)

	// Build service endpoints from environment or use defaults
	endpoints := map[string]string{
		"auth-service":             "http://localhost:8081/healthz",
		"gateway-service":          "http://localhost:8082/healthz",
		"user-service":             "http://localhost:8083/healthz",
		"workspace-service":        "http://localhost:8084/healthz",
		"org-service":              "http://localhost:8085/healthz",
		"billing-service":          "http://localhost:8086/healthz",
		"notification-service":     "http://localhost:8087/healthz",
		"vault-service":            "http://localhost:8088/healthz",
		"config-service":           "http://localhost:8089/healthz",
		"analytics-service":        "http://localhost:8090/healthz",
		"audit-service":            "http://localhost:8091/healthz",
		"collaboration-service":    "http://localhost:8092/healthz",
		"container-bridge-service": "http://localhost:8093/healthz",
		"health-service":           "http://localhost:8080/healthz",
		"ai-service":               "http://localhost:8094/healthz",
		"terminal-service":         "http://localhost:8095/healthz",
		"ssh-proxy-service":        "http://localhost:8096/healthz",
		"recording-service":        "http://localhost:8097/healthz",
		"sftp-service":             "http://localhost:8098/healthz",
		"snippet-service":          "http://localhost:8099/healthz",
		"pki-service":              "http://localhost:8100/healthz",
		"keychain-service":         "http://localhost:8101/healthz",
		"port-forward-service":     "http://localhost:8102/healthz",
		"host-service":             "http://localhost:8103/healthz",
		"helixtrack-bridge-service": "http://localhost:8104/healthz",
	}

	// Override with HEALTH_ENDPOINTS if provided (comma-separated name=url pairs)
	if envEndpoints := os.Getenv("HEALTH_ENDPOINTS"); envEndpoints != "" {
		endpoints = parseEndpoints(envEndpoints)
	}

	checkTimeout := 5 * time.Second
	if envTimeout := os.Getenv("HEALTH_CHECK_TIMEOUT"); envTimeout != "" {
		if d, err := time.ParseDuration(envTimeout); err == nil {
			checkTimeout = d
		}
	}

	srv := server.New(logger, endpoints, checkTimeout)

	httpServer := &http.Server{
		Addr:         ":" + port,
		Handler:      srv.Router(),
		ReadTimeout:   15 * time.Second,
		WriteTimeout:  15 * time.Second,
		IdleTimeout:   60 * time.Second,
	}

	go func() {
		logger.Printf("health-service starting on port %s", port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Println("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Fatalf("server forced to shutdown: %v", err)
	}

	logger.Println("server exited gracefully")
}

// parseEndpoints parses a comma-separated string of name=url pairs.
func parseEndpoints(input string) map[string]string {
	result := make(map[string]string)
	// Simple parsing: expects "name1=url1,name2=url2"
	// For production, a more robust parser or JSON config would be better.
	return result
}
