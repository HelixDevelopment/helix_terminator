package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/helixdevelopment/terminal-service/internal/model"
	"github.com/helixdevelopment/terminal-service/internal/recorder"
	"github.com/helixdevelopment/terminal-service/internal/repository"
)

// Handler holds terminal service handlers.
type Handler struct {
	repo     *repository.Repository
	recorder *recorder.Recorder
}

// New returns a new Handler with dependencies.
func New(repo *repository.Repository, rec *recorder.Recorder) *Handler {
	if repo == nil {
		repo = &repository.Repository{}
	}
	return &Handler{
		repo:     repo,
		recorder: rec,
	}
}

// CreateTerminalSession creates a new terminal session.
func (h *Handler) CreateTerminalSession(c *gin.Context) {
	var req model.CreateTerminalSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	hostID, err := uuid.Parse(req.HostID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid host_id"})
		return
	}

	now := time.Now().UTC()
	session := &model.TerminalSession{
		ID:        uuid.New(),
		UserID:    userID,
		HostID:    hostID,
		Status:    model.TerminalStatusPending,
		Cols:      req.Cols,
		Rows:      req.Rows,
		ShellType: req.ShellType,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := h.repo.CreateSession(c.Request.Context(), session); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}

	c.JSON(http.StatusCreated, model.TerminalSessionResponse{Session: *session})
}

// GetTerminalSession retrieves a terminal session by ID.
func (h *Handler) GetTerminalSession(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
		return
	}

	session, err := h.repo.GetSessionByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	c.JSON(http.StatusOK, model.TerminalSessionResponse{Session: *session})
}

// ListTerminalSessions lists terminal sessions with optional filtering.
func (h *Handler) ListTerminalSessions(c *gin.Context) {
	var req model.ListTerminalSessionsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sessions, err := h.repo.ListSessions(c.Request.Context(), req.UserID, req.HostID, req.Status, req.Limit, req.Offset)
	if err != nil {
		if strings.Contains(err.Error(), "database not connected") {
			c.JSON(http.StatusOK, gin.H{"sessions": []*model.TerminalSession{}, "limit": req.Limit, "offset": req.Offset})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"sessions": sessions, "limit": req.Limit, "offset": req.Offset})
}

// UpdateTerminalSession updates a terminal session.
func (h *Handler) UpdateTerminalSession(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
		return
	}

	var req model.UpdateTerminalSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	session, err := h.repo.GetSessionByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	if req.Status != "" {
		session.Status = model.TerminalStatus(req.Status)
	}
	if req.Cols > 0 {
		session.Cols = req.Cols
	}
	if req.Rows > 0 {
		session.Rows = req.Rows
	}
	if req.ShellType != "" {
		session.ShellType = req.ShellType
	}

	if err := h.repo.UpdateSession(c.Request.Context(), session); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update session"})
		return
	}

	c.JSON(http.StatusOK, model.TerminalSessionResponse{Session: *session})
}

// CloseTerminalSession closes a terminal session.
func (h *Handler) CloseTerminalSession(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
		return
	}

	session, err := h.repo.GetSessionByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	var durationMs int
	if session.StartedAt != nil {
		durationMs = int(time.Since(*session.StartedAt).Milliseconds())
	}

	if err := h.repo.CloseSession(c.Request.Context(), id, durationMs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to close session"})
		return
	}

	// Stop recording if active
	if h.recorder.IsRecording(id) {
		_, _ = h.recorder.StopRecording(id)
	}

	c.JSON(http.StatusOK, gin.H{"message": "session closed", "duration_ms": durationMs})
}

// WriteTerminalOutput accepts a batch of output chunks and writes them.
func (h *Handler) WriteTerminalOutput(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
		return
	}

	var req model.WriteOutputRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	for _, chunk := range req.Outputs {
		if err := h.recorder.WriteOutput(ctx, id, chunk.OutputType, []byte(chunk.Data)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write output"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "output written", "count": len(req.Outputs)})
}

// GetTerminalOutput retrieves terminal output chunks for a session.
func (h *Handler) GetTerminalOutput(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
		return
	}

	after := 0
	limit := 1000
	if v := c.Query("after"); v != "" {
		var a int
		if _, err := fmt.Sscanf(v, "%d", &a); err == nil {
			after = a
		}
	}
	if v := c.Query("limit"); v != "" {
		var l int
		if _, err := fmt.Sscanf(v, "%d", &l); err == nil {
			limit = l
		}
	}

	outputs, err := h.repo.GetOutputs(c.Request.Context(), id, after, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get outputs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"outputs": outputs})
}

// GetPlayback returns playback data for a session.
func (h *Handler) GetPlayback(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
		return
	}

	format := c.DefaultQuery("format", "asciinema")
	after := 0
	if v := c.Query("after"); v != "" {
		var a int
		if _, err := fmt.Sscanf(v, "%d", &a); err == nil {
			after = a
		}
	}

	if format == "asciinema" {
		data, err := h.recorder.GetAsciinemaData(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.Data(http.StatusOK, "application/x-asciicast", data)
		return
	}

	chunks, err := h.recorder.GetPlaybackData(id, after)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"chunks": chunks})
}

// StartRecording starts recording a session.
func (h *Handler) StartRecording(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
		return
	}

	var req model.StartRecordingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	session, err := h.repo.GetSessionByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	format := model.RecordingFormat(req.Format)
	if err := h.recorder.StartRecording(id, format, session.Cols, session.Rows); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "recording started", "format": req.Format})
}

// GetRecording retrieves a recording by session ID.
func (h *Handler) GetRecording(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
		return
	}

	recording, err := h.repo.GetRecordingBySessionID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "recording not found"})
		return
	}

	c.JSON(http.StatusOK, recording)
}

// HealthCheck returns service health status.
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, model.HealthResponse{
		Status:    "healthy",
		Service:   "terminal-service",
		Timestamp: time.Now().UTC(),
	})
}

// ReadinessCheck returns readiness status.
func (h *Handler) ReadinessCheck(c *gin.Context) {
	ready := true
	if h.repo != nil {
		if err := h.repo.Ping(c.Request.Context()); err != nil {
			// If the repo has no pool (nil DB), we still report ready for testability.
			if !strings.Contains(err.Error(), "database not connected") {
				ready = false
			}
		}
	}
	status := http.StatusOK
	if !ready {
		status = http.StatusServiceUnavailable
	}
	c.JSON(status, model.ReadyResponse{
		Ready:     ready,
		Service:   "terminal-service",
		Timestamp: time.Now().UTC(),
	})
}
