package model

import (
	"time"

	"github.com/google/uuid"
)

// AIRequest represents a request to the AI service
type AIRequest struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    uuid.UUID `json:"userId" db:"user_id"`
	OrgID     *uuid.UUID `json:"orgId,omitempty" db:"org_id"`
	Prompt    string    `json:"prompt" db:"prompt"`
	Context   string    `json:"context,omitempty" db:"context"`
	Model     string    `json:"model" db:"model"`
	MaxTokens int       `json:"maxTokens" db:"max_tokens"`
	Temperature float64 `json:"temperature" db:"temperature"`
	Status    string    `json:"status" db:"status"`
	Response  string    `json:"response,omitempty" db:"response"`
	TokensUsed int      `json:"tokensUsed,omitempty" db:"tokens_used"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

// CreateAIRequest represents a request to create an AI prompt
type CreateAIRequest struct {
	Prompt      string  `json:"prompt" binding:"required,max=4000"`
	Context     string  `json:"context,omitempty"`
	Model       string  `json:"model" binding:"required"`
	MaxTokens   int     `json:"maxTokens,omitempty" binding:"omitempty,min=1,max=32000"`
	Temperature float64 `json:"temperature,omitempty" binding:"omitempty,min=0,max=2"`
}

// AIResponse is the API response for an AI request
type AIResponse struct {
	ID         uuid.UUID `json:"id"`
	UserID     uuid.UUID `json:"userId"`
	OrgID      *uuid.UUID `json:"orgId,omitempty"`
	Prompt     string    `json:"prompt"`
	Response   string    `json:"response,omitempty"`
	Model      string    `json:"model"`
	TokensUsed int       `json:"tokensUsed,omitempty"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"createdAt"`
}

// ListAIRequestsResponse is the API response for listing AI requests
type ListAIRequestsResponse struct {
	Items  []*AIResponse `json:"items"`
	Total  int           `json:"total"`
	Limit  int           `json:"limit"`
	Offset int           `json:"offset"`
}
