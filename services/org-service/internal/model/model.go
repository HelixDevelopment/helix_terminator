package model

import (
	"time"

	"github.com/google/uuid"
)

// Plan represents an organization subscription plan.
type Plan string

const (
	PlanFree       Plan = "free"
	PlanPro        Plan = "pro"
	PlanEnterprise Plan = "enterprise"
)

// Role represents a membership role.
type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
)

// Organization represents a platform organization.
type Organization struct {
	ID          uuid.UUID              `json:"id" db:"id"`
	Name        string                 `json:"name" db:"name"`
	Slug        string                 `json:"slug" db:"slug"`
	Description string                 `json:"description,omitempty" db:"description"`
	LogoURL     string                 `json:"logoUrl,omitempty" db:"logo_url"`
	OwnerID     uuid.UUID              `json:"ownerId" db:"owner_id"`
	Plan        Plan                   `json:"plan" db:"plan"`
	Settings    map[string]interface{} `json:"settings,omitempty" db:"settings"`
	MemberCount int                    `json:"memberCount" db:"member_count"`
	CreatedAt   time.Time              `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time              `json:"updatedAt" db:"updated_at"`
	DeletedAt   *time.Time             `json:"-" db:"deleted_at"`
}

// Team represents a team within an organization.
type Team struct {
	ID          uuid.UUID `json:"id" db:"id"`
	OrgID       uuid.UUID `json:"orgId" db:"org_id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description,omitempty" db:"description"`
	MemberCount int       `json:"memberCount" db:"member_count"`
	CreatedAt   time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time `json:"updatedAt" db:"updated_at"`
}

// Membership represents a user's membership in an organization or team.
type Membership struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	OrgID     uuid.UUID  `json:"orgId" db:"org_id"`
	UserID    uuid.UUID  `json:"userId" db:"user_id"`
	TeamID    *uuid.UUID `json:"teamId,omitempty" db:"team_id"`
	Role      Role       `json:"role" db:"role"`
	InvitedBy *uuid.UUID `json:"invitedBy,omitempty" db:"invited_by"`
	InvitedAt *time.Time `json:"invitedAt,omitempty" db:"invited_at"`
	JoinedAt  *time.Time `json:"joinedAt,omitempty" db:"joined_at"`
	CreatedAt time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time  `json:"updatedAt" db:"updated_at"`
}

// CreateOrgRequest represents a request to create a new organization.
type CreateOrgRequest struct {
	Name        string                 `json:"name" binding:"required,max=255"`
	Slug        string                 `json:"slug" binding:"required,max=255"`
	Description string                 `json:"description,omitempty" binding:"omitempty,max=1000"`
	LogoURL     string                 `json:"logoUrl,omitempty" binding:"omitempty,max=2048"`
	Plan        Plan                   `json:"plan,omitempty" binding:"omitempty,oneof=free pro enterprise"`
	Settings    map[string]interface{} `json:"settings,omitempty"`
}

// UpdateOrgRequest represents a request to update an organization.
type UpdateOrgRequest struct {
	Name        string                 `json:"name,omitempty" binding:"omitempty,max=255"`
	Slug        string                 `json:"slug,omitempty" binding:"omitempty,max=255"`
	Description string                 `json:"description,omitempty" binding:"omitempty,max=1000"`
	LogoURL     string                 `json:"logoUrl,omitempty" binding:"omitempty,max=2048"`
	Plan        Plan                   `json:"plan,omitempty" binding:"omitempty,oneof=free pro enterprise"`
	Settings    map[string]interface{} `json:"settings,omitempty"`
}

// ListOrgsRequest represents query parameters for listing organizations.
type ListOrgsRequest struct {
	Search string `json:"search,omitempty" form:"search"`
	Limit  int    `json:"limit,omitempty" form:"limit" binding:"min=1,max=100"`
	Offset int    `json:"offset,omitempty" form:"offset" binding:"min=0"`
}

// CreateTeamRequest represents a request to create a new team.
type CreateTeamRequest struct {
	Name        string `json:"name" binding:"required,max=255"`
	Description string `json:"description,omitempty" binding:"omitempty,max=1000"`
}

// UpdateTeamRequest represents a request to update a team.
type UpdateTeamRequest struct {
	Name        string `json:"name,omitempty" binding:"omitempty,max=255"`
	Description string `json:"description,omitempty" binding:"omitempty,max=1000"`
}

// AddMemberRequest represents a request to add a member to an organization.
type AddMemberRequest struct {
	UserID    string `json:"userId" binding:"required,uuid"`
	TeamID    string `json:"teamId,omitempty" binding:"omitempty,uuid"`
	Role      Role   `json:"role" binding:"required,oneof=owner admin member"`
	InvitedBy string `json:"invitedBy,omitempty" binding:"omitempty,uuid"`
}

// UpdateMemberRequest represents a request to update a member's role or team.
type UpdateMemberRequest struct {
	Role   Role   `json:"role,omitempty" binding:"omitempty,oneof=owner admin member"`
	TeamID string `json:"teamId,omitempty" binding:"omitempty,uuid"`
}

// OrgResponse wraps an Organization for API responses.
type OrgResponse struct {
	Organization
}

// TeamResponse wraps a Team for API responses.
type TeamResponse struct {
	Team
}

// MemberResponse wraps a Membership for API responses.
type MemberResponse struct {
	Membership
}

// ListOrgsResponse represents the response for listing organizations.
type ListOrgsResponse struct {
	Data   []*Organization `json:"data"`
	Total  int             `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}
