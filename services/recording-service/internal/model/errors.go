package model

import "errors"

var (
	ErrMissingSessionID = errors.New("session_id is required")
	ErrMissingHostID    = errors.New("host_id is required")
	ErrMissingUserID    = errors.New("user_id is required")
	ErrMissingOrgID     = errors.New("org_id is required")
	ErrMissingFilePath  = errors.New("file_path is required")
	ErrInvalidFormat    = errors.New("format must be 'asciinema' or 'raw'")
	ErrInvalidStatus    = errors.New("invalid status value")
	ErrNegativeDuration = errors.New("duration_sec cannot be negative")
	ErrNegativeFileSize = errors.New("file_size_bytes cannot be negative")
)
