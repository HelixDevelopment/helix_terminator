package model

import (
	"time"

	"github.com/google/uuid"
)

// SFTPSessionStatus constants
const (
	SFTPSessionStatusPending   = "pending"
	SFTPSessionStatusActive    = "active"
	SFTPSessionStatusCompleted = "completed"
	SFTPSessionStatusFailed    = "failed"
	SFTPSessionStatusCancelled = "cancelled"
)

// SFTPDirection constants
const (
	SFTPDirectionUpload   = "upload"
	SFTPDirectionDownload = "download"
)

// SFTPSession represents an SFTP file transfer session
type SFTPSession struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	HostID           uuid.UUID  `json:"hostId" db:"host_id"`
	UserID           uuid.UUID  `json:"userId" db:"user_id"`
	RemotePath       string     `json:"remotePath" db:"remote_path"`
	LocalPath        string     `json:"localPath" db:"local_path"`
	Direction        string     `json:"direction" db:"direction"`
	Status           string     `json:"status" db:"status"`
	BytesTransferred int64      `json:"bytesTransferred" db:"bytes_transferred"`
	CreatedAt        time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt        time.Time  `json:"updatedAt" db:"updated_at"`
	CompletedAt      *time.Time `json:"completedAt,omitempty" db:"completed_at"`
}

// CreateSFTPSessionRequest represents a request to create a session
type CreateSFTPSessionRequest struct {
	HostID     string `json:"hostId" binding:"required,uuid"`
	RemotePath string `json:"remotePath" binding:"required,max=1024"`
	LocalPath  string `json:"localPath" binding:"required,max=1024"`
	Direction  string `json:"direction" binding:"required,oneof=upload download"`
}

// UpdateSFTPSessionRequest represents a request to update a session
type UpdateSFTPSessionRequest struct {
	Status           string `json:"status" binding:"oneof=pending active completed failed cancelled"`
	BytesTransferred int64  `json:"bytesTransferred" binding:"min=0"`
}

// SFTPSessionResponse is the API response
type SFTPSessionResponse struct {
	ID               uuid.UUID  `json:"id"`
	HostID           uuid.UUID  `json:"hostId"`
	UserID           uuid.UUID  `json:"userId"`
	RemotePath       string     `json:"remotePath"`
	LocalPath        string     `json:"localPath"`
	Direction        string     `json:"direction"`
	Status           string     `json:"status"`
	BytesTransferred int64      `json:"bytesTransferred"`
	CreatedAt        time.Time  `json:"createdAt"`
	CompletedAt      *time.Time `json:"completedAt,omitempty"`
}

// ListSFTPSessionsResponse is the API response for listing
type ListSFTPSessionsResponse struct {
	Items  []*SFTPSessionResponse `json:"items"`
	Total  int                    `json:"total"`
	Limit  int                    `json:"limit"`
	Offset int                    `json:"offset"`
}
