package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helixdevelopment/terminal-service/internal/handler"
	"github.com/helixdevelopment/terminal-service/internal/model"
	"github.com/helixdevelopment/terminal-service/internal/recorder"
	"github.com/helixdevelopment/terminal-service/internal/repository"
)

func setupTestRouter() (*gin.Engine, *handler.Handler) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handler.New(nil, recorder.NewRecorder("", nil))
	return r, h
}

func TestHealthCheck(t *testing.T) {
	r, h := setupTestRouter()
	r.GET("/healthz", h.HealthCheck)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/healthz", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.HealthResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "healthy", resp.Status)
	assert.Equal(t, "terminal-service", resp.Service)
}

func TestReadinessCheck(t *testing.T) {
	r, h := setupTestRouter()
	r.GET("/healthz/ready", h.ReadinessCheck)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/healthz/ready", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.ReadyResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp.Ready)
	assert.Equal(t, "terminal-service", resp.Service)
}

func TestCreateTerminalSessionValidation(t *testing.T) {
	r, h := setupTestRouter()
	r.POST("/api/v1/terminal/sessions", h.CreateTerminalSession)

	body := map[string]interface{}{
		"user_id": "not-a-uuid",
		"host_id": "also-not-uuid",
	}
	b, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/terminal/sessions", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListTerminalSessions(t *testing.T) {
	r, h := setupTestRouter()
	r.GET("/api/v1/terminal/sessions", h.ListTerminalSessions)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/terminal/sessions?limit=10&offset=0", nil)
	r.ServeHTTP(w, req)

	// Should succeed even with nil repo (returns empty list)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	sessions, ok := resp["sessions"].([]interface{})
	require.True(t, ok)
	assert.Empty(t, sessions)
}

func TestGetTerminalSessionInvalidID(t *testing.T) {
	r, h := setupTestRouter()
	r.GET("/api/v1/terminal/sessions/:id", h.GetTerminalSession)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/terminal/sessions/invalid-id", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCloseTerminalSessionInvalidID(t *testing.T) {
	r, h := setupTestRouter()
	r.POST("/api/v1/terminal/sessions/:id/close", h.CloseTerminalSession)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/terminal/sessions/bad-id/close", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWriteTerminalOutputValidation(t *testing.T) {
	r, h := setupTestRouter()
	r.POST("/api/v1/terminal/sessions/:id/output", h.WriteTerminalOutput)

	body := model.WriteOutputRequest{
		Outputs: []model.OutputChunk{
			{OutputType: "badtype", Data: "hello"},
		},
	}
	b, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/terminal/sessions/550e8400-e29b-41d4-a716-446655440000/output", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestStartRecordingValidation(t *testing.T) {
	r, h := setupTestRouter()
	r.POST("/api/v1/terminal/sessions/:id/recording", h.StartRecording)

	body := map[string]string{"format": "mp4"}
	b, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/terminal/sessions/550e8400-e29b-41d4-a716-446655440000/recording", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPlaybackBadSessionID(t *testing.T) {
	r, h := setupTestRouter()
	r.GET("/api/v1/terminal/sessions/:id/playback", h.GetPlayback)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/terminal/sessions/bad-id/playback", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
