package model

import (
	"time"

	"github.com/google/uuid"
)

// Snippet represents a reusable code/command snippet
type Snippet struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	OrgID       *uuid.UUID `json:"orgId,omitempty" db:"org_id"`
	CreatedBy   uuid.UUID  `json:"createdBy" db:"created_by"`
	Name        string     `json:"name" db:"name"`
	Content     string     `json:"content" db:"content"`
	Language    string     `json:"language" db:"language"`
	Tags        []string   `json:"tags" db:"tags"`
	Description string     `json:"description" db:"description"`
	IsPublic    bool       `json:"isPublic" db:"is_public"`
	UsageCount  int        `json:"usageCount" db:"usage_count"`
	CreatedAt   time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time  `json:"updatedAt" db:"updated_at"`
}

// CreateSnippetRequest represents a request to create a snippet
type CreateSnippetRequest struct {
	Name        string   `json:"name" binding:"required,max=255"`
	Content     string   `json:"content" binding:"required,max=10000"`
	Language    string   `json:"language" binding:"required,max=50"`
	Tags        []string `json:"tags"`
	Description string   `json:"description" binding:"max=1000"`
	IsPublic    bool     `json:"isPublic"`
}

// UpdateSnippetRequest represents a request to update a snippet
type UpdateSnippetRequest struct {
	Name        string   `json:"name" binding:"max=255"`
	Content     string   `json:"content" binding:"max=10000"`
	Language    string   `json:"language" binding:"max=50"`
	Tags        []string `json:"tags"`
	Description string   `json:"description" binding:"max=1000"`
	IsPublic    *bool    `json:"isPublic,omitempty"`
}

// SnippetResponse is the API response
type SnippetResponse struct {
	ID          uuid.UUID `json:"id"`
	OrgID       *uuid.UUID `json:"orgId,omitempty"`
	CreatedBy   uuid.UUID `json:"createdBy"`
	Name        string    `json:"name"`
	Content     string    `json:"content"`
	Language    string    `json:"language"`
	Tags        []string  `json:"tags"`
	Description string    `json:"description"`
	IsPublic    bool      `json:"isPublic"`
	UsageCount  int       `json:"usageCount"`
	CreatedAt   time.Time `json:"createdAt"`
}

// ListSnippetsResponse is the API response for listing
type ListSnippetsResponse struct {
	Items  []*SnippetResponse `json:"items"`
	Total  int                `json:"total"`
	Limit  int                `json:"limit"`
	Offset int                `json:"offset"`
}
