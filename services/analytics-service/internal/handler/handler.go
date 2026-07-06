package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/helixdevelopment/analytics-service/internal/model"
	"github.com/helixdevelopment/analytics-service/internal/repository"
)

// Handler contains HTTP handlers for analytics
type Handler struct {
	repo *repository.Repository
}

// New creates a new Handler
func New(repo *repository.Repository) *Handler {
	return &Handler{repo: repo}
}

// CreateEvent creates a new analytics event
func (h *Handler) CreateEvent(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}
	var req model.CreateAnalyticsEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var orgID *uuid.UUID
	if req.OrgID != "" {
		id, err := uuid.Parse(req.OrgID)
		if err == nil {
			orgID = &id
		}
	}
	var hostID *uuid.UUID
	if req.HostID != "" {
		id, err := uuid.Parse(req.HostID)
		if err == nil {
			hostID = &id
		}
	}
	userIDStr := c.GetString("user_id")
	userID, _ := uuid.Parse(userIDStr)
	event := &model.AnalyticsEvent{
		ID:        uuid.New(),
		OrgID:     orgID,
		UserID:    userID,
		HostID:    hostID,
		EventType: req.EventType,
		Payload:   req.Payload,
	}
	if err := h.repo.CreateEvent(c.Request.Context(), event); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, event)
}

// GetEvent retrieves an event by ID
func (h *Handler) GetEvent(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event ID"})
		return
	}
	event, err := h.repo.GetEventByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, event)
}

// ListEvents retrieves events with filtering
func (h *Handler) ListEvents(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}
	var orgID *uuid.UUID
	if oStr := c.Query("org_id"); oStr != "" {
		id, err := uuid.Parse(oStr)
		if err == nil {
			orgID = &id
		}
	}
	eventType := c.Query("event_type")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if offset < 0 {
		offset = 0
	}
	events, total, err := h.repo.ListEvents(c.Request.Context(), orgID, eventType, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data":   events,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// CountByEventType returns event counts grouped by type
func (h *Handler) CountByEventType(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}
	var orgID *uuid.UUID
	if oStr := c.Query("org_id"); oStr != "" {
		id, err := uuid.Parse(oStr)
		if err == nil {
			orgID = &id
		}
	}
	summaries, err := h.repo.CountByEventType(c.Request.Context(), orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": summaries})
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
