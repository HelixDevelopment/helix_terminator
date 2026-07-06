package model

import (
	"time"

	"github.com/google/uuid"
)

// RecordingStatus constants
const (
	RecordingStatusRecording = "recording"
	RecordingStatusPaused    = "paused"
	RecordingStatusCompleted = "completed"
	RecordingStatusFailed    = "failed"
)

// RecordingFormat constants
const (
	RecordingFormatAsciinema = "asciinema"
	RecordingFormatRaw       = "raw"
)

// Recording represents a terminal session recording
type Recording struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	SessionID    uuid.UUID  `json:"sessionId" db:"session_id"`
	HostID       uuid.UUID  `json:"hostId" db:"host_id"`
	UserID       uuid.UUID  `json:"userId" db:"user_id"`
	OrgID        *uuid.UUID `json:"orgId,omitempty" db:"org_id"`
	FilePath     string     `json:"filePath" db:"file_path"`
	Format       string     `json:"format" db:"format"`
	Status       string     `json:"status" db:"status"`
	DurationSec  int        `json:"durationSec" db:"duration_sec"`
	FileSizeBytes int64     `json:"fileSizeBytes" db:"file_size_bytes"`
	CreatedAt    time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt    time.Time  `json:"updatedAt" db:"updated_at"`
}

// CreateRecordingRequest represents a request to create a recording
type CreateRecordingRequest struct {
	SessionID     string `json:"sessionId" binding:"required,uuid"`
	HostID        string `json:"hostId" binding:"required,uuid"`
	FilePath      string `json:"filePath" binding:"required,max=1024"`
	Format        string `json:"format" binding:"required,oneof=asciinema raw"`
}

// UpdateRecordingRequest represents a request to update a recording
type UpdateRecordingRequest struct {
	Status        string `json:"status" binding:"oneof=recording paused completed failed"`
	DurationSec   int    `json:"durationSec" binding:"min=0"`
	FileSizeBytes int64  `json:"fileSizeBytes" binding:"min=0"`
}

// RecordingResponse is the API response
type RecordingResponse struct {
	ID            uuid.UUID  `json:"id"`
	SessionID     uuid.UUID  `json:"sessionId"`
	HostID        uuid.UUID  `json:"hostId"`
	UserID        uuid.UUID  `json:"userId"`
	OrgID         *uuid.UUID `json:"orgId,omitempty"`
	FilePath      string     `json:"filePath"`
	Format        string     `json:"format"`
	Status        string     `json:"status"`
	DurationSec   int        `json:"durationSec"`
	FileSizeBytes int64      `json:"fileSizeBytes"`
	CreatedAt     time.Time  `json:"createdAt"`
}

// ListRecordingsResponse is the API response for listing
type ListRecordingsResponse struct {
	Items  []*RecordingResponse `json:"items"`
	Total  int                  `json:"total"`
	Limit  int                  `json:"limit"`
	Offset int                  `json:"offset"`
}
