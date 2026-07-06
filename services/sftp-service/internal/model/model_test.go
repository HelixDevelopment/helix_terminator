package model

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestSFTPSessionStatusConstants(t *testing.T) {
	assert.Equal(t, "pending", SFTPSessionStatusPending)
	assert.Equal(t, "active", SFTPSessionStatusActive)
	assert.Equal(t, "completed", SFTPSessionStatusCompleted)
	assert.Equal(t, "failed", SFTPSessionStatusFailed)
	assert.Equal(t, "cancelled", SFTPSessionStatusCancelled)
}

func TestSFTPDirectionConstants(t *testing.T) {
	assert.Equal(t, "upload", SFTPDirectionUpload)
	assert.Equal(t, "download", SFTPDirectionDownload)
}

func TestSFTPSession_Model(t *testing.T) {
	s := SFTPSession{
		ID:         uuid.New(),
		HostID:     uuid.New(),
		UserID:     uuid.New(),
		RemotePath: "/remote/file.txt",
		LocalPath:  "/local/file.txt",
		Direction:  SFTPDirectionDownload,
		Status:     SFTPSessionStatusPending,
	}
	assert.NotEqual(t, uuid.Nil, s.ID)
	assert.Equal(t, "/remote/file.txt", s.RemotePath)
	assert.Equal(t, SFTPDirectionDownload, s.Direction)
	assert.Equal(t, SFTPSessionStatusPending, s.Status)
}
