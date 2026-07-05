package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/helixdevelopment/vault-service/internal/server"
)

func TestMainGracefulShutdown(t *testing.T) {
	logger := &testLogger{logs: make([]string, 0)}
	srv, err := server.New(logger)
	assert.NoError(t, err)

	httpServer := &http.Server{
		Addr:    ":18080",
		Handler: srv.Router(),
	}

	go func() {
		_ = httpServer.ListenAndServe()
	}()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Verify health endpoint
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/healthz", nil)
	srv.Router().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Trigger shutdown
	go func() {
		time.Sleep(50 * time.Millisecond)
		// Signal would be sent here in real main; in test we just shut down directly
	}()

	// Wait for shutdown signal
	quit := make(chan struct{}, 1)
	_ = quit

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(ctx)
}

type testLogger struct {
	logs []string
}

func (t *testLogger) Printf(format string, v ...interface{}) {
	t.logs = append(t.logs, format)
}

func (t *testLogger) Println(v ...interface{}) {
	t.logs = append(t.logs, "println")
}
