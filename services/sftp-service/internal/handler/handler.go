package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/helixdevelopment/sftp-service/internal/model"
	"github.com/helixdevelopment/sftp-service/internal/repository"
)

// Handler contains HTTP handlers for SFTP
type Handler struct {
	repo *repository.Repository
}

// New creates a new Handler
func New(repo *repository.Repository) *Handler {
	return &Handler{repo: repo}
}

// CreateSession creates a new SFTP session
func (h *Handler) CreateSession(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}
	var req model.CreateSFTPSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	hostID, _ := uuid.Parse(req.HostID)
	userIDStr := c.GetString("user_id")
	userID, _ := uuid.Parse(userIDStr)
	session := &model.SFTPSession{
		ID:         uuid.New(),
		HostID:     hostID,
		UserID:     userID,
		RemotePath: req.RemotePath,
		LocalPath:  req.LocalPath,
		Direction:  req.Direction,
		Status:     model.SFTPSessionStatusPending,
	}
	if err := h.repo.CreateSession(c.Request.Context(), session); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, session)
}

// GetSession retrieves a session by ID
func (h *Handler) GetSession(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}
	session, err := h.repo.GetSessionByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, session)
}

// ListSessions retrieves sessions with filtering
func (h *Handler) ListSessions(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}
	var hostID uuid.UUID
	if hStr := c.Query("host_id"); hStr != "" {
		id, err := uuid.Parse(hStr)
		if err == nil {
			hostID = id
		}
	}
	status := c.Query("status")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if offset < 0 {
		offset = 0
	}
	sessions, total, err := h.repo.ListSessions(c.Request.Context(), hostID, status, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data":   sessions,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// UpdateSession updates a session
func (h *Handler) UpdateSession(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}
	var req model.UpdateSFTPSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var completedAt interface{}
	if req.Status == model.SFTPSessionStatusCompleted || req.Status == model.SFTPSessionStatusFailed || req.Status == model.SFTPSessionStatusCancelled {
		completedAt = time.Now()
	}
	updates := map[string]interface{}{
		"status":            req.Status,
		"bytes_transferred": req.BytesTransferred,
		"completed_at":      completedAt,
	}
	if err := h.repo.UpdateSession(c.Request.Context(), id, updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "session updated"})
}

// DeleteSession deletes a session
func (h *Handler) DeleteSession(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}
	if err := h.repo.DeleteSession(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "session deleted"})
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
