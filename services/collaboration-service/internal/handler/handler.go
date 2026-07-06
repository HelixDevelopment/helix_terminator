package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/helixdevelopment/collaboration-service/internal/model"
	"github.com/helixdevelopment/collaboration-service/internal/repository"
)

// Handler contains HTTP handlers for collaboration
type Handler struct {
	repo *repository.Repository
}

// New creates a new Handler
func New(repo *repository.Repository) *Handler {
	return &Handler{repo: repo}
}

// CreateSessionRequest represents a session creation request
type CreateSessionRequest struct {
	HostID uuid.UUID `json:"host_id" binding:"required"`
	Name   string    `json:"name" binding:"required,max=255"`
	OrgID  uuid.UUID `json:"org_id"`
}

// CreateSession creates a new collaboration session
func (h *Handler) CreateSession(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}
	var req CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userIDStr := c.GetString("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		userID = uuid.Nil
	}
	session := &model.CollaborationSession{
		ID:        uuid.New(),
		HostID:    req.HostID,
		CreatedBy: userID,
		OrgID:     &req.OrgID,
		Name:      req.Name,
		Status:    model.SessionStatusActive,
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
	var hostID, createdBy uuid.UUID
	if hStr := c.Query("host_id"); hStr != "" {
		id, err := uuid.Parse(hStr)
		if err == nil {
			hostID = id
		}
	}
	if cStr := c.Query("created_by"); cStr != "" {
		id, err := uuid.Parse(cStr)
		if err == nil {
			createdBy = id
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
	sessions, total, err := h.repo.ListSessions(c.Request.Context(), hostID, createdBy, limit, offset)
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

// JoinSessionRequest represents a join request
type JoinSessionRequest struct {
	UserID uuid.UUID `json:"user_id" binding:"required"`
}

// JoinSession adds a participant to a session
func (h *Handler) JoinSession(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}
	var req JoinSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.repo.JoinSession(c.Request.Context(), id, req.UserID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "joined session"})
}

// LeaveSession removes a participant from a session
func (h *Handler) LeaveSession(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}
	userIDStr := c.GetString("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}
	if err := h.repo.LeaveSession(c.Request.Context(), id, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "left session"})
}

// EndSession ends a collaboration session
func (h *Handler) EndSession(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}
	if err := h.repo.EndSession(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "session ended"})
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
