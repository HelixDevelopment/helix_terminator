package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/helixdevelopment/recording-service/internal/model"
	"github.com/helixdevelopment/recording-service/internal/repository"
)

// Handler contains HTTP handlers for recordings
type Handler struct {
	repo *repository.Repository
}

// New creates a new Handler
func New(repo *repository.Repository) *Handler {
	return &Handler{repo: repo}
}

// CreateRecording creates a new recording
func (h *Handler) CreateRecording(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}
	var req model.CreateRecordingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	sessionID, _ := uuid.Parse(req.SessionID)
	hostID, _ := uuid.Parse(req.HostID)
	userIDStr := c.GetString("user_id")
	userID, _ := uuid.Parse(userIDStr)
	recording := &model.Recording{
		ID:            uuid.New(),
		SessionID:     sessionID,
		HostID:        hostID,
		UserID:        userID,
		FilePath:      req.FilePath,
		Format:        req.Format,
		Status:        model.RecordingStatusRecording,
		DurationSec:   0,
		FileSizeBytes: 0,
	}
	if err := h.repo.CreateRecording(c.Request.Context(), recording); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, recording)
}

// GetRecording retrieves a recording by ID
func (h *Handler) GetRecording(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid recording ID"})
		return
	}
	recording, err := h.repo.GetRecordingByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, recording)
}

// ListRecordings retrieves recordings with filtering
func (h *Handler) ListRecordings(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}
	var hostID, sessionID uuid.UUID
	if hStr := c.Query("host_id"); hStr != "" {
		id, err := uuid.Parse(hStr)
		if err == nil {
			hostID = id
		}
	}
	if sStr := c.Query("session_id"); sStr != "" {
		id, err := uuid.Parse(sStr)
		if err == nil {
			sessionID = id
		}
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if offset < 0 {
		offset = 0
	}
	recordings, total, err := h.repo.ListRecordings(c.Request.Context(), hostID, sessionID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data":   recordings,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// UpdateRecording updates a recording
func (h *Handler) UpdateRecording(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid recording ID"})
		return
	}
	var req model.UpdateRecordingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updates := map[string]interface{}{
		"status":          req.Status,
		"duration_sec":    req.DurationSec,
		"file_size_bytes": req.FileSizeBytes,
	}
	if err := h.repo.UpdateRecording(c.Request.Context(), id, updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "recording updated"})
}

// DeleteRecording deletes a recording
func (h *Handler) DeleteRecording(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid recording ID"})
		return
	}
	if err := h.repo.DeleteRecording(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "recording deleted"})
}

// HealthCheck returns service health
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}

// ReadinessCheck returns readiness status
func (h *Handler) ReadinessCheck(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready"})
		return
	}
	if err := h.repo.Ping(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready", "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}
