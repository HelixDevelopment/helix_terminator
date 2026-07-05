package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/helixdevelopment/org-service/internal/model"
	"github.com/helixdevelopment/org-service/internal/repository"
)

// Handler holds org service handlers.
type Handler struct {
	repo *repository.Repository
}

// New returns a new Handler with dependencies.
func New(repo *repository.Repository) *Handler {
	return &Handler{repo: repo}
}

// CreateOrg handles POST /api/v1/orgs.
func (h *Handler) CreateOrg(c *gin.Context) {
	var req model.CreateOrgRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Plan == "" {
		req.Plan = model.PlanFree
	}

	userIDStr, _ := c.Get("userID")
	var ownerID uuid.UUID
	if userIDStr != nil {
		ownerID, _ = uuid.Parse(userIDStr.(string))
	}
	if ownerID == uuid.Nil {
		ownerID = uuid.New()
	}

	org := &model.Organization{
		ID:          uuid.New(),
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		LogoURL:     req.LogoURL,
		OwnerID:     ownerID,
		Plan:        req.Plan,
		Settings:    req.Settings,
		MemberCount: 1,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}

	if err := h.repo.CreateOrg(c.Request.Context(), org); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	// Add owner membership
	membership := &model.Membership{
		ID:        uuid.New(),
		OrgID:     org.ID,
		UserID:    ownerID,
		Role:      model.RoleOwner,
		JoinedAt:  ptr(time.Now().UTC()),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := h.repo.AddMember(c.Request.Context(), membership); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, model.OrgResponse{Organization: *org})
}

// ListOrgs handles GET /api/v1/orgs.
func (h *Handler) ListOrgs(c *gin.Context) {
	var req model.ListOrgsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Limit == 0 {
		req.Limit = 20
	}

	userIDStr, _ := c.Get("userID")
	var userID uuid.UUID
	if userIDStr != nil {
		userID, _ = uuid.Parse(userIDStr.(string))
	}
	if userID == uuid.Nil {
		userID = uuid.MustParse("00000000-0000-0000-0000-000000000000")
	}

	orgs, total, err := h.repo.ListOrgs(c.Request.Context(), userID, req.Search, req.Limit, req.Offset)
	if err != nil {
		if strings.Contains(err.Error(), "database not connected") {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list organizations"})
		return
	}

	c.JSON(http.StatusOK, model.ListOrgsResponse{
		Data:   orgs,
		Total:  total,
		Limit:  req.Limit,
		Offset: req.Offset,
	})
}

// GetOrg handles GET /api/v1/orgs/:id.
func (h *Handler) GetOrg(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization id"})
		return
	}

	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}

	org, err := h.repo.GetOrgByID(c.Request.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "database not connected") {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "organization not found"})
		return
	}

	c.JSON(http.StatusOK, model.OrgResponse{Organization: *org})
}

// GetOrgBySlug handles GET /api/v1/orgs/by-slug/:slug.
func (h *Handler) GetOrgBySlug(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "slug is required"})
		return
	}

	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}

	org, err := h.repo.GetOrgBySlug(c.Request.Context(), slug)
	if err != nil {
		if strings.Contains(err.Error(), "database not connected") {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "organization not found"})
		return
	}

	c.JSON(http.StatusOK, model.OrgResponse{Organization: *org})
}

// UpdateOrg handles PUT /api/v1/orgs/:id.
func (h *Handler) UpdateOrg(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization id"})
		return
	}

	var req model.UpdateOrgRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Slug != "" {
		updates["slug"] = req.Slug
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.LogoURL != "" {
		updates["logo_url"] = req.LogoURL
	}
	if req.Plan != "" {
		updates["plan"] = req.Plan
	}
	if req.Settings != nil {
		updates["settings"] = req.Settings
	}

	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}

	if err := h.repo.UpdateOrg(c.Request.Context(), id, updates); err != nil {
		if strings.Contains(err.Error(), "database not connected") {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update organization"})
		return
	}

	org, err := h.repo.GetOrgByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "updated"})
		return
	}
	c.JSON(http.StatusOK, model.OrgResponse{Organization: *org})
}

// DeleteOrg handles DELETE /api/v1/orgs/:id.
func (h *Handler) DeleteOrg(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization id"})
		return
	}

	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}

	if err := h.repo.DeleteOrg(c.Request.Context(), id); err != nil {
		if strings.Contains(err.Error(), "database not connected") {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete organization"})
		return
	}

	c.Status(http.StatusNoContent)
}

// CreateTeam handles POST /api/v1/orgs/:id/teams.
func (h *Handler) CreateTeam(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization id"})
		return
	}

	var req model.CreateTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	team := &model.Team{
		ID:          uuid.New(),
		OrgID:       orgID,
		Name:        req.Name,
		Description: req.Description,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}

	if err := h.repo.CreateTeam(c.Request.Context(), team); err != nil {
		if strings.Contains(err.Error(), "database not connected") {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create team"})
		return
	}

	c.JSON(http.StatusCreated, model.TeamResponse{Team: *team})
}

// ListTeams handles GET /api/v1/orgs/:id/teams.
func (h *Handler) ListTeams(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization id"})
		return
	}

	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	if limit <= 0 {
		limit = 20
	}

	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}

	teams, err := h.repo.ListTeams(c.Request.Context(), orgID, limit, offset)
	if err != nil {
		if strings.Contains(err.Error(), "database not connected") {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list teams"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": teams})
}

// GetTeam handles GET /api/v1/teams/:id.
func (h *Handler) GetTeam(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team id"})
		return
	}

	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}

	team, err := h.repo.GetTeamByID(c.Request.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "database not connected") {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "team not found"})
		return
	}

	c.JSON(http.StatusOK, model.TeamResponse{Team: *team})
}

// UpdateTeam handles PUT /api/v1/teams/:id.
func (h *Handler) UpdateTeam(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team id"})
		return
	}

	var req model.UpdateTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}

	team, err := h.repo.GetTeamByID(c.Request.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "database not connected") {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "team not found"})
		return
	}

	if req.Name != "" {
		team.Name = req.Name
	}
	if req.Description != "" {
		team.Description = req.Description
	}
	team.UpdatedAt = time.Now().UTC()

	if err := h.repo.UpdateTeam(c.Request.Context(), team); err != nil {
		if strings.Contains(err.Error(), "database not connected") {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update team"})
		return
	}

	c.JSON(http.StatusOK, model.TeamResponse{Team: *team})
}

// DeleteTeam handles DELETE /api/v1/teams/:id.
func (h *Handler) DeleteTeam(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team id"})
		return
	}

	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}

	if err := h.repo.DeleteTeam(c.Request.Context(), id); err != nil {
		if strings.Contains(err.Error(), "database not connected") {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete team"})
		return
	}

	c.Status(http.StatusNoContent)
}

// AddMember handles POST /api/v1/orgs/:id/members.
func (h *Handler) AddMember(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization id"})
		return
	}

	var req model.AddMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	var teamID *uuid.UUID
	if req.TeamID != "" {
		tid, err := uuid.Parse(req.TeamID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team id"})
			return
		}
		teamID = &tid
	}

	var invitedBy *uuid.UUID
	if req.InvitedBy != "" {
		ib, err := uuid.Parse(req.InvitedBy)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invited by id"})
			return
		}
		invitedBy = &ib
	}

	now := time.Now().UTC()
	membership := &model.Membership{
		ID:        uuid.New(),
		OrgID:     orgID,
		UserID:    userID,
		TeamID:    teamID,
		Role:      req.Role,
		InvitedBy: invitedBy,
		InvitedAt: ptr(now),
		JoinedAt:  ptr(now),
		CreatedAt: now,
		UpdatedAt: now,
	}

	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}

	if err := h.repo.AddMember(c.Request.Context(), membership); err != nil {
		if strings.Contains(err.Error(), "database not connected") {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add member"})
		return
	}

	c.JSON(http.StatusCreated, model.MemberResponse{Membership: *membership})
}

// ListMembers handles GET /api/v1/orgs/:id/members.
func (h *Handler) ListMembers(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization id"})
		return
	}

	role := c.Query("role")
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	if limit <= 0 {
		limit = 20
	}

	var teamID *uuid.UUID
	if tidStr := c.Query("team_id"); tidStr != "" {
		tid, err := uuid.Parse(tidStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team id"})
			return
		}
		teamID = &tid
	}

	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}

	members, total, err := h.repo.ListMembers(c.Request.Context(), orgID, teamID, role, limit, offset)
	if err != nil {
		if strings.Contains(err.Error(), "database not connected") {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list members"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   members,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// UpdateMember handles PUT /api/v1/orgs/:id/members/:user_id.
func (h *Handler) UpdateMember(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization id"})
		return
	}

	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	var req model.UpdateMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}

	membership, err := h.repo.GetMember(c.Request.Context(), orgID, userID)
	if err != nil {
		if strings.Contains(err.Error(), "database not connected") {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "membership not found"})
		return
	}

	if req.Role != "" {
		membership.Role = req.Role
	}
	if req.TeamID != "" {
		tid, err := uuid.Parse(req.TeamID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team id"})
			return
		}
		membership.TeamID = &tid
	}
	membership.UpdatedAt = time.Now().UTC()

	if err := h.repo.UpdateMember(c.Request.Context(), membership); err != nil {
		if strings.Contains(err.Error(), "database not connected") {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update member"})
		return
	}

	c.JSON(http.StatusOK, model.MemberResponse{Membership: *membership})
}

// RemoveMember handles DELETE /api/v1/orgs/:id/members/:user_id.
func (h *Handler) RemoveMember(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization id"})
		return
	}

	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}

	if err := h.repo.RemoveMember(c.Request.Context(), orgID, userID); err != nil {
		if strings.Contains(err.Error(), "database not connected") {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove member"})
		return
	}

	c.Status(http.StatusNoContent)
}

// HealthCheck returns service health status.
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "org-service",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// ReadinessCheck returns readiness status.
func (h *Handler) ReadinessCheck(c *gin.Context) {
	ready := true
	if h.repo == nil {
		ready = false
	} else if err := h.repo.Ping(c.Request.Context()); err != nil {
		ready = false
	}
	status := http.StatusOK
	if !ready {
		status = http.StatusServiceUnavailable
	}
	c.JSON(status, gin.H{
		"ready":     ready,
		"service":   "org-service",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func ptr(t time.Time) *time.Time {
	return &t
}
