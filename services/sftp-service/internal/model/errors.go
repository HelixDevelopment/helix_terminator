package model

import "errors"

var (
	ErrInvalidHostID     = errors.New("host_id is required")
	ErrInvalidUserID     = errors.New("user_id is required")
	ErrMissingRemotePath = errors.New("remote_path is required")
	ErrMissingLocalPath  = errors.New("local_path is required")
	ErrInvalidDirection  = errors.New("direction must be upload or download")
)
