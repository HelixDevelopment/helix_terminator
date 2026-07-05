package model

import (
	"time"

	"github.com/google/uuid"
)

// TerminalStatus represents the state of a terminal session.
type TerminalStatus string

const (
	TerminalStatusPending TerminalStatus = "pending"
	TerminalStatusActive  TerminalStatus = "active"
	TerminalStatusPaused  TerminalStatus = "paused"
	TerminalStatusClosed  TerminalStatus = "closed"
	TerminalStatusError   TerminalStatus = "error"
)

// OutputType represents the type of terminal output.
type OutputType string

const (
	OutputTypeStdout  OutputType = "stdout"
	OutputTypeStderr  OutputType = "stderr"
	OutputTypeCommand OutputType = "command"
)

// RecordingFormat represents the format of a terminal recording.
type RecordingFormat string

const (
	RecordingFormatAsciinema RecordingFormat = "asciinema"
	RecordingFormatRaw       RecordingFormat = "raw"
	RecordingFormatHTML      RecordingFormat = "html"
)

// TerminalSession represents a terminal session managed by the service.
type TerminalSession struct {
	ID           uuid.UUID      `json:"id" db:"id"`
	UserID       uuid.UUID      `json:"user_id" db:"user_id"`
	HostID       uuid.UUID      `json:"host_id" db:"host_id"`
	SSHSessionID *uuid.UUID     `json:"ssh_session_id,omitempty" db:"ssh_session_id"`
	Status       TerminalStatus `json:"status" db:"status"`
	StartedAt    *time.Time     `json:"started_at,omitempty" db:"started_at"`
	EndedAt      *time.Time     `json:"ended_at,omitempty" db:"ended_at"`
	DurationMs   int            `json:"duration_ms" db:"duration_ms"`
	Cols         int            `json:"cols" db:"cols"`
	Rows         int            `json:"rows" db:"rows"`
	ShellType    string         `json:"shell_type" db:"shell_type"`
	CreatedAt    time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at" db:"updated_at"`
}

// TerminalOutput represents a chunk of terminal output.
type TerminalOutput struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	SessionID    uuid.UUID  `json:"session_id" db:"session_id"`
	OutputType   OutputType `json:"output_type" db:"output_type"`
	Data         []byte     `json:"data" db:"data"`
	Timestamp    time.Time  `json:"timestamp" db:"timestamp"`
	SequenceNum  int        `json:"sequence_num" db:"sequence_num"`
}

// TerminalRecording represents a persisted recording of a terminal session.
type TerminalRecording struct {
	ID         uuid.UUID       `json:"id" db:"id"`
	SessionID  uuid.UUID       `json:"session_id" db:"session_id"`
	Format     RecordingFormat `json:"format" db:"format"`
	FilePath   string          `json:"file_path" db:"file_path"`
	FileSize   int64           `json:"file_size" db:"file_size"`
	DurationMs int             `json:"duration_ms" db:"duration_ms"`
	CreatedAt  time.Time       `json:"created_at" db:"created_at"`
}

// CreateTerminalSessionRequest represents a request to create a new terminal session.
type CreateTerminalSessionRequest struct {
	UserID    string `json:"user_id" binding:"required,uuid"`
	HostID    string `json:"host_id" binding:"required,uuid"`
	Cols      int    `json:"cols" binding:"min=1,max=999"`
	Rows      int    `json:"rows" binding:"min=1,max=999"`
	ShellType string `json:"shell_type,omitempty" binding:"omitempty,max=50"`
}

// UpdateTerminalSessionRequest represents a request to update a terminal session.
type UpdateTerminalSessionRequest struct {
	Status   string `json:"status,omitempty" binding:"omitempty,oneof=pending active paused closed error"`
	Cols     int    `json:"cols,omitempty" binding:"omitempty,min=1,max=999"`
	Rows     int    `json:"rows,omitempty" binding:"omitempty,min=1,max=999"`
	ShellType string `json:"shell_type,omitempty" binding:"omitempty,max=50"`
}

// ListTerminalSessionsRequest represents query parameters for listing sessions.
type ListTerminalSessionsRequest struct {
	UserID string `form:"user_id" binding:"omitempty,uuid"`
	HostID string `form:"host_id" binding:"omitempty,uuid"`
	Status string `form:"status" binding:"omitempty,oneof=pending active paused closed error"`
	Limit  int    `form:"limit,default=20" binding:"min=1,max=100"`
	Offset int    `form:"offset,default=0" binding:"min=0"`
}

// TerminalSessionResponse wraps a TerminalSession with additional metadata.
type TerminalSessionResponse struct {
	Session TerminalSession `json:"session"`
}

// PlaybackRequest represents query parameters for playback.
type PlaybackRequest struct {
	Format string `form:"format,default=asciinema" binding:"omitempty,oneof=asciinema raw"`
	After  int    `form:"after,default=0" binding:"min=0"`
	Limit  int    `form:"limit,default=1000" binding:"min=1,max=10000"`
}

// WriteOutputRequest represents a batch of output chunks to write.
type WriteOutputRequest struct {
	Outputs []OutputChunk `json:"outputs" binding:"required,dive"`
}

// OutputChunk represents a single output chunk in a batch write.
type OutputChunk struct {
	OutputType OutputType `json:"output_type" binding:"required,oneof=stdout stderr command"`
	Data       string     `json:"data" binding:"required"`
}

// StartRecordingRequest represents a request to start recording a session.
type StartRecordingRequest struct {
	Format string `json:"format" binding:"required,oneof=asciinema raw html"`
}

// HealthResponse represents the health check response.
type HealthResponse struct {
	Status    string    `json:"status"`
	Service   string    `json:"service"`
	Timestamp time.Time `json:"timestamp"`
}

// ReadyResponse represents the readiness check response.
type ReadyResponse struct {
	Ready     bool      `json:"ready"`
	Service   string    `json:"service"`
	Timestamp time.Time `json:"timestamp"`
}
