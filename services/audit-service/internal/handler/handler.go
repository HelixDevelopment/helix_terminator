package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/helixdevelopment/audit-service/internal/model"
	"github.com/helixdevelopment/audit-service/internal/repository"
)

// Handler holds audit service handlers
type Handler struct {
	repo *repository.Repository
}

// New returns a new Handler with dependencies
func New(repo *repository.Repository) *Handler {
	return &Handler{repo: repo}
}

// CreateAuditLog handles POST /api/v1/audit/logs
func (h *Handler) CreateAuditLog(c *gin.Context) {
	var req model.CreateAuditLogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	detailsBytes, err := json.Marshal(req.Details)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid details"})
		return
	}

	ipAddress := req.IPAddress
	if ipAddress == "" {
		ipAddress = c.ClientIP()
	}
	userAgent := req.UserAgent
	if userAgent == "" {
		userAgent = c.Request.UserAgent()
	}

	log := &model.AuditLog{
		ID:           uuid.New(),
		OrgID:        req.OrgID,
		UserID:       req.UserID,
		Action:       req.Action,
		ResourceType: req.ResourceType,
		ResourceID:   req.ResourceID,
		Details:      detailsBytes,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		Timestamp:    time.Now().UTC(),
		Severity:     req.Severity,
	}

	if err := h.repo.CreateAuditLog(c.Request.Context(), log); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create audit log"})
		return
	}

	c.JSON(http.StatusCreated, toAuditLogResponse(log))
}

// ListAuditLogs handles GET /api/v1/audit/logs
func (h *Handler) ListAuditLogs(c *gin.Context) {
	var req model.ListAuditLogsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var orgID, userID *uuid.UUID
	if req.OrgID != "" {
		id, err := uuid.Parse(req.OrgID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid org_id"})
			return
		}
		orgID = &id
	}
	if req.UserID != "" {
		id, err := uuid.Parse(req.UserID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
			return
		}
		userID = &id
	}

	var startTime, endTime *time.Time
	if req.Start != "" {
		st, err := time.Parse(time.RFC3339, req.Start)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start time format, expected RFC3339"})
			return
		}
		startTime = &st
	}
	if req.End != "" {
		et, err := time.Parse(time.RFC3339, req.End)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end time format, expected RFC3339"})
			return
		}
		endTime = &et
	}

	logs, total, err := h.repo.ListAuditLogs(
		c.Request.Context(),
		orgID,
		userID,
		req.Action,
		req.ResourceType,
		req.Severity,
		startTime,
		endTime,
		req.Limit,
		req.Offset,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list audit logs"})
		return
	}

	resp := &model.ListAuditLogsResponse{
		Logs:   make([]*model.AuditLogResponse, len(logs)),
		Total:  total,
		Limit:  req.Limit,
		Offset: req.Offset,
	}
	for i, log := range logs {
		resp.Logs[i] = toAuditLogResponse(log)
	}

	c.JSON(http.StatusOK, resp)
}

// GetAuditLog handles GET /api/v1/audit/logs/:id
func (h *Handler) GetAuditLog(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	log, err := h.repo.GetAuditLogByID(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "audit log not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "audit log not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get audit log"})
		return
	}

	c.JSON(http.StatusOK, toAuditLogResponse(log))
}

// CountByAction handles GET /api/v1/audit/stats/actions
func (h *Handler) CountByAction(c *gin.Context) {
	var req struct {
		OrgID string `form:"org_id"`
		Start string `form:"start"`
		End   string `form:"end"`
	}
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var orgID *uuid.UUID
	if req.OrgID != "" {
		id, err := uuid.Parse(req.OrgID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid org_id"})
			return
		}
		orgID = &id
	}

	var startTime, endTime *time.Time
	if req.Start != "" {
		st, err := time.Parse(time.RFC3339, req.Start)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start time format, expected RFC3339"})
			return
		}
		startTime = &st
	}
	if req.End != "" {
		et, err := time.Parse(time.RFC3339, req.End)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end time format, expected RFC3339"})
			return
		}
		endTime = &et
	}

	counts, err := h.repo.CountByAction(c.Request.Context(), orgID, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count by action"})
		return
	}

	c.JSON(http.StatusOK, model.CountResponse{Counts: counts})
}

// CountByResourceType handles GET /api/v1/audit/stats/resources
func (h *Handler) CountByResourceType(c *gin.Context) {
	var req struct {
		OrgID string `form:"org_id"`
		Start string `form:"start"`
		End   string `form:"end"`
	}
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var orgID *uuid.UUID
	if req.OrgID != "" {
		id, err := uuid.Parse(req.OrgID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid org_id"})
			return
		}
		orgID = &id
	}

	var startTime, endTime *time.Time
	if req.Start != "" {
		st, err := time.Parse(time.RFC3339, req.Start)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start time format, expected RFC3339"})
			return
		}
		startTime = &st
	}
	if req.End != "" {
		et, err := time.Parse(time.RFC3339, req.End)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end time format, expected RFC3339"})
			return
		}
		endTime = &et
	}

	counts, err := h.repo.CountByResourceType(c.Request.Context(), orgID, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count by resource type"})
		return
	}

	c.JSON(http.StatusOK, model.CountResponse{Counts: counts})
}

// HealthCheck returns service health status
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "audit-service",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// ReadinessCheck returns readiness status (503 if no DB)
func (h *Handler) ReadinessCheck(c *gin.Context) {
	if err := h.repo.Ping(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"ready":     false,
			"service":   "audit-service",
			"error":     err.Error(),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ready":     true,
		"service":   "audit-service",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func toAuditLogResponse(log *model.AuditLog) *model.AuditLogResponse {
	resp := &model.AuditLogResponse{
		ID:           log.ID,
		OrgID:        log.OrgID,
		UserID:       log.UserID,
		Action:       log.Action,
		ResourceType: log.ResourceType,
		ResourceID:   log.ResourceID,
		IPAddress:    log.IPAddress,
		UserAgent:    log.UserAgent,
		Timestamp:    log.Timestamp,
		Severity:     log.Severity,
	}
	if len(log.Details) > 0 {
		var details interface{}
		if err := json.Unmarshal(log.Details, &details); err == nil {
			resp.Details = details
		}
	}
	return resp
}
