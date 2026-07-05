package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/helixdevelopment/ssh-proxy-service/internal/server"
)

func TestMainServer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	srv, err := server.New(nil)
	assert.NoError(t, err)
	assert.NotNil(t, srv)

	router := srv.Router()
	assert.NotNil(t, router)
}

func TestHealthEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	srv, err := server.New(nil)
	assert.NoError(t, err)

	router := srv.Router().(*gin.Engine)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/healthz", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "healthy")

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/healthz/ready", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ready")
}
