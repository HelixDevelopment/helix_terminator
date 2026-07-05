package model

import (
	"time"

	"github.com/google/uuid"
)

// Valid config scopes
type ConfigScope string

const (
	ScopeGlobal ConfigScope = "global"
	ScopeOrg    ConfigScope = "org"
	ScopeUser   ConfigScope = "user"
)

// Valid value types
const (
	ValueTypeString   = "string"
	ValueTypeInt      = "int"
	ValueTypeBool     = "bool"
	ValueTypeJSON     = "json"
	ValueTypeEncrypted = "encrypted"
)

// Config represents a configuration key-value pair.
type Config struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	Scope       string     `json:"scope" db:"scope"`
	ScopeID     *uuid.UUID `json:"scopeId,omitempty" db:"scope_id"`
	Key         string     `json:"key" db:"key"`
	Value       string     `json:"value" db:"value"`
	ValueType   string     `json:"valueType" db:"value_type"`
	Description string     `json:"description,omitempty" db:"description"`
	IsSecret    bool       `json:"isSecret" db:"is_secret"`
	CreatedAt   time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time  `json:"updatedAt" db:"updated_at"`
	DeletedAt   *time.Time `json:"-" db:"deleted_at"`
}

// CreateConfigRequest represents a request to create a new config.
type CreateConfigRequest struct {
	Scope       string  `json:"scope" binding:"required,oneof=global org user"`
	ScopeID     *string `json:"scopeId,omitempty"`
	Key         string  `json:"key" binding:"required,max=255"`
	Value       string  `json:"value" binding:"required"`
	ValueType   string  `json:"valueType" binding:"required,oneof=string int bool json encrypted"`
	Description string  `json:"description,omitempty" binding:"max=1000"`
	IsSecret    bool    `json:"isSecret"`
}

// UpdateConfigRequest represents a request to update an existing config.
type UpdateConfigRequest struct {
	Value       *string `json:"value,omitempty"`
	ValueType   *string `json:"valueType,omitempty" binding:"omitempty,oneof=string int bool json encrypted"`
	Description *string `json:"description,omitempty" binding:"omitempty,max=1000"`
	IsSecret    *bool   `json:"isSecret,omitempty"`
}

// ListConfigsRequest represents query parameters for listing configs.
type ListConfigsRequest struct {
	Scope   string `form:"scope" binding:"omitempty,oneof=global org user"`
	ScopeID string `form:"scope_id,omitempty"`
	Search  string `form:"search,omitempty"`
	Limit   int    `form:"limit,default=20" binding:"min=1,max=100"`
	Offset  int    `form:"offset,default=0" binding:"min=0"`
}

// ConfigResponse represents a single config in API responses.
type ConfigResponse struct {
	ID          uuid.UUID  `json:"id"`
	Scope       string     `json:"scope"`
	ScopeID     *uuid.UUID `json:"scopeId,omitempty"`
	Key         string     `json:"key"`
	Value       string     `json:"value"`
	ValueType   string     `json:"valueType"`
	Description string     `json:"description,omitempty"`
	IsSecret    bool       `json:"isSecret"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

// ListConfigsResponse represents a paginated list of configs.
type ListConfigsResponse struct {
	Configs []*ConfigResponse `json:"configs"`
	Total   int               `json:"total"`
	Limit   int               `json:"limit"`
	Offset  int               `json:"offset"`
}

// ToResponse converts a Config model to a ConfigResponse DTO.
func (c *Config) ToResponse() *ConfigResponse {
	return &ConfigResponse{
		ID:          c.ID,
		Scope:       c.Scope,
		ScopeID:     c.ScopeID,
		Key:         c.Key,
		Value:       c.Value,
		ValueType:   c.ValueType,
		Description: c.Description,
		IsSecret:    c.IsSecret,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
}
