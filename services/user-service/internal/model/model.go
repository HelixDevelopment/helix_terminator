package model

import (
	"time"
)

// TODO: define domain models for user-service

// BaseModel provides common fields.
type BaseModel struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
