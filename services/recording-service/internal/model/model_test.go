package model

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestRecordingStatusConstants(t *testing.T) {
	assert.Equal(t, "recording", RecordingStatusRecording)
	assert.Equal(t, "paused", RecordingStatusPaused)
	assert.Equal(t, "completed", RecordingStatusCompleted)
	assert.Equal(t, "failed", RecordingStatusFailed)
}

func TestRecordingFormatConstants(t *testing.T) {
	assert.Equal(t, "asciinema", RecordingFormatAsciinema)
	assert.Equal(t, "raw", RecordingFormatRaw)
}

func TestRecording_Model(t *testing.T) {
	r := Recording{
		ID:            uuid.New(),
		SessionID:     uuid.New(),
		HostID:        uuid.New(),
		FilePath:      "/recordings/test.cast",
		Format:        RecordingFormatAsciinema,
		Status:        RecordingStatusRecording,
		DurationSec:   0,
		FileSizeBytes: 0,
	}
	assert.NotEqual(t, uuid.Nil, r.ID)
	assert.Equal(t, "/recordings/test.cast", r.FilePath)
	assert.Equal(t, RecordingFormatAsciinema, r.Format)
	assert.Equal(t, RecordingStatusRecording, r.Status)
}
