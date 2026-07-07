package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/helixdevelopment/user-service/internal/model"
	"github.com/helixdevelopment/user-service/internal/repository"
)

// Handler holds user service handlers
type Handler struct {
	repo *repository.Repository
}

// New creates a new Handler
func New(repo *repository.Repository) *Handler {
	return &Handler{repo: repo}
}

// CreateUser handles user creation
func (h *Handler) CreateUser(c *gin.Context) {
	var req model.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	exists, err := h.repo.EmailExists(c.Request.Context(), req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check email availability"})
		return
	}
	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		return
	}

	user := &model.User{
		ID:            uuid.New().String(),
		Email:         req.Email,
		DisplayName:   req.DisplayName,
		Role:          req.Role,
		Permissions:   req.Permissions,
		OrgID:         req.OrgID,
		EmailVerified: false,
	}

	if err := h.repo.CreateUser(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, toUserResponse(user))
}

// GetUser handles retrieving a user by ID
func (h *Handler) GetUser(c *gin.Context) {
	id := c.Param("id")
	user, err := h.repo.GetUserByID(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user"})
		return
	}
	c.JSON(http.StatusOK, toUserResponse(user))
}

// GetUserByEmail handles retrieving a user by email
func (h *Handler) GetUserByEmail(c *gin.Context) {
	email := c.Query("email")
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email query parameter required"})
		return
	}
	user, err := h.repo.GetUserByEmail(c.Request.Context(), email)
	if err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user"})
		return
	}
	c.JSON(http.StatusOK, toUserResponse(user))
}

// ListUsers handles listing users with filtering
func (h *Handler) ListUsers(c *gin.Context) {
	var req model.ListUsersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	users, total, err := h.repo.ListUsers(c.Request.Context(), req.OrgID, req.Role, req.Search, req.Limit, req.Offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list users"})
		return
	}

	resp := &model.ListUsersResponse{
		Users:  make([]*model.UserResponse, len(users)),
		Total:  total,
		Limit:  req.Limit,
		Offset: req.Offset,
	}
	for i, u := range users {
		resp.Users[i] = toUserResponse(u)
	}
	c.JSON(http.StatusOK, resp)
}

// UpdateUser handles updating a user
func (h *Handler) UpdateUser(c *gin.Context) {
	id := c.Param("id")
	var req model.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := make(map[string]interface{})
	if req.DisplayName != nil {
		updates["display_name"] = *req.DisplayName
	}
	if req.AvatarURL != nil {
		updates["avatar_url"] = *req.AvatarURL
	}
	if req.Role != nil {
		updates["role"] = *req.Role
	}
	if req.Permissions != nil {
		updates["permissions"] = req.Permissions
	}
	if req.OrgID != nil {
		updates["org_id"] = *req.OrgID
	}
	if req.Bio != nil {
		updates["bio"] = *req.Bio
	}
	if req.Timezone != nil {
		updates["timezone"] = *req.Timezone
	}
	if req.Locale != nil {
		updates["locale"] = *req.Locale
	}

	if err := h.repo.UpdateUser(c.Request.Context(), id, updates); err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user"})
		return
	}

	user, err := h.repo.GetUserByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get updated user"})
		return
	}
	c.JSON(http.StatusOK, toUserResponse(user))
}

// DeleteUser handles soft-deleting a user
func (h *Handler) DeleteUser(c *gin.Context) {
	id := c.Param("id")
	if err := h.repo.DeleteUser(c.Request.Context(), id); err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete user"})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

// GetProfile handles retrieving a user profile
func (h *Handler) GetProfile(c *gin.Context) {
	id := c.Param("id")
	profile, err := h.repo.GetProfile(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get profile"})
		return
	}
	c.JSON(http.StatusOK, toUserProfileResponse(profile))
}

// UpdateProfile handles updating a user profile
func (h *Handler) UpdateProfile(c *gin.Context) {
	id := c.Param("id")
	var req model.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := make(map[string]interface{})
	if req.DisplayName != nil {
		updates["display_name"] = *req.DisplayName
	}
	if req.AvatarURL != nil {
		updates["avatar_url"] = *req.AvatarURL
	}

	profile := make(map[string]interface{})
	if req.Bio != nil {
		profile["bio"] = *req.Bio
	}
	if req.Timezone != nil {
		profile["timezone"] = *req.Timezone
	}
	if req.Locale != nil {
		profile["locale"] = *req.Locale
	}
	if req.Preferences != nil {
		profile["preferences"] = req.Preferences
	}
	if req.SSHPublicKey != nil {
		profile["ssh_public_key"] = *req.SSHPublicKey
	}
	if req.AvatarURL != nil {
		profile["avatar_url"] = *req.AvatarURL
	}

	if len(updates) > 0 {
		if err := h.repo.UpdateUser(c.Request.Context(), id, updates); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user"})
			return
		}
	}
	if len(profile) > 0 {
		if err := h.repo.CreateOrUpdateProfile(c.Request.Context(), id, profile); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update profile"})
			return
		}
	}

	profileData, err := h.repo.GetProfile(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get updated profile"})
		return
	}
	c.JSON(http.StatusOK, toUserProfileResponse(profileData))
}

// HealthCheck returns service health status
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "user-service", "timestamp": time.Now().UTC()})
}

// ReadinessCheck returns service readiness status. Unlike HealthCheck
// (liveness - "is the process up"), readiness reports whether the
// service can genuinely serve traffic, which for user-service means a
// reachable database. Reports 503 + status:not_ready the moment the
// database is unreachable, closing the T8-6 bluff where this handler
// previously returned an unconditional "status":"ready" without ever
// checking the DB (a crashed-DB service still reported ready,
// defeating orchestrator/k8s health gating on this security-critical
// service).
func (h *Handler) ReadinessCheck(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "not_ready",
			"service": "user-service",
			"reason":  "database repository not configured",
		})
		return
	}

	if err := h.repo.Ping(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "not_ready",
			"service": "user-service",
			"reason":  "database unreachable: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ready", "service": "user-service"})
}

func toUserResponse(u *model.User) *model.UserResponse {
	return &model.UserResponse{
		ID:            u.ID,
		Email:         u.Email,
		DisplayName:   u.DisplayName,
		AvatarURL:     u.AvatarURL,
		Role:          u.Role,
		Permissions:   u.Permissions,
		OrgID:         u.OrgID,
		EmailVerified: u.EmailVerified,
		LastLoginAt:   u.LastLoginAt,
		CreatedAt:     u.CreatedAt,
	}
}

func toUserProfileResponse(p *model.UserProfile) *model.UserProfileResponse {
	return &model.UserProfileResponse{
		UserResponse: *toUserResponse(&p.User),
		Bio:          p.Bio,
		Timezone:     p.Timezone,
		Locale:       p.Locale,
		Preferences:  p.Preferences,
		SSHPublicKey: p.SSHPublicKey,
		GitHubID:     p.GitHubID,
		GitLabID:     p.GitLabID,
	}
}
