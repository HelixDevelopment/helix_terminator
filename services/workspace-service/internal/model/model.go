package model

import (
	"time"

	"github.com/google/uuid"
)

// Workspace represents a project container that groups hosts, terminals, and resources.
type Workspace struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	OrgID       uuid.UUID  `json:"org_id" db:"org_id"`
	UserID      uuid.UUID  `json:"user_id" db:"user_id"`
	Name        string     `json:"name" db:"name"`
	Description string     `json:"description,omitempty" db:"description"`
	Color       string     `json:"color,omitempty" db:"color"`
	Icon        string     `json:"icon,omitempty" db:"icon"`
	HostIDs     []uuid.UUID `json:"host_ids,omitempty" db:"host_ids"`
	Tags        []string   `json:"tags,omitempty" db:"tags"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt   *time.Time `json:"-" db:"deleted_at"`
}

// WorkspaceHost represents the many-to-many relationship between workspaces and hosts.
type WorkspaceHost struct {
	WorkspaceID uuid.UUID `json:"workspace_id" db:"workspace_id"`
	HostID      uuid.UUID `json:"host_id" db:"host_id"`
	AddedAt     time.Time `json:"added_at" db:"added_at"`
	AddedBy     uuid.UUID `json:"added_by" db:"added_by"`
}

// CreateWorkspaceRequest represents a request to create a new workspace.
type CreateWorkspaceRequest struct {
	Name        string      `json:"name" binding:"required,max=255"`
	Description string      `json:"description,omitempty" binding:"omitempty,max=1000"`
	Color       string      `json:"color,omitempty" binding:"omitempty,max=7"`
	Icon        string      `json:"icon,omitempty" binding:"omitempty,max=50"`
	HostIDs     []uuid.UUID `json:"host_ids,omitempty"`
	Tags        []string    `json:"tags,omitempty"`
}

// UpdateWorkspaceRequest represents a request to update an existing workspace.
type UpdateWorkspaceRequest struct {
	Name        string      `json:"name,omitempty" binding:"omitempty,max=255"`
	Description string      `json:"description,omitempty" binding:"omitempty,max=1000"`
	Color       string      `json:"color,omitempty" binding:"omitempty,max=7"`
	Icon        string      `json:"icon,omitempty" binding:"omitempty,max=50"`
	HostIDs     []uuid.UUID `json:"host_ids,omitempty"`
	Tags        []string    `json:"tags,omitempty"`
}

// ListWorkspacesRequest represents query parameters for listing workspaces.
type ListWorkspacesRequest struct {
	OrgID  string   `json:"org_id,omitempty" form:"org_id"`
	UserID string   `json:"user_id,omitempty" form:"user_id"`
	Tags   []string `json:"tags,omitempty" form:"tags"`
	Limit  int      `json:"limit,omitempty" form:"limit" binding:"min=1,max=100"`
	Offset int      `json:"offset,omitempty" form:"offset" binding:"min=0"`
}

// AddHostRequest represents a request to add a host to a workspace.
type AddHostRequest struct {
	HostID uuid.UUID `json:"host_id" binding:"required"`
}

// WorkspaceResponse wraps a Workspace for API responses.
type WorkspaceResponse struct {
	Workspace
}

// ListWorkspacesResponse wraps a list of workspaces with pagination.
type ListWorkspacesResponse struct {
	Data   []*Workspace `json:"data"`
	Total  int          `json:"total"`
	Limit  int          `json:"limit"`
	Offset int          `json:"offset"`
}
