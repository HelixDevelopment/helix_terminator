package recorder_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helixdevelopment/terminal-service/internal/model"
	"github.com/helixdevelopment/terminal-service/internal/recorder"
)

// mockRepo implements recorder.OutputRepository for testing.
type mockRepo struct {
	outputs []*model.TerminalOutput
}

func (m *mockRepo) CreateOutput(_ context.Context, output *model.TerminalOutput) error {
	m.outputs = append(m.outputs, output)
	return nil
}

func TestNewRecorder(t *testing.T) {
	rec := recorder.NewRecorder("", nil)
	assert.NotNil(t, rec)
}

func TestStartAndStopRecording(t *testing.T) {
	repo := &mockRepo{}
	rec := recorder.NewRecorder("", repo)

	sessionID := uuid.New()

	err := rec.StartRecording(sessionID, model.RecordingFormatAsciinema, 120, 40)
	require.NoError(t, err)
	assert.True(t, rec.IsRecording(sessionID))

	// Starting again should fail
	err = rec.StartRecording(sessionID, model.RecordingFormatAsciinema, 120, 40)
	assert.Error(t, err)

	recording, err := rec.StopRecording(sessionID)
	require.NoError(t, err)
	assert.NotNil(t, recording)
	assert.Equal(t, model.RecordingFormatAsciinema, recording.Format)
	assert.False(t, rec.IsRecording(sessionID))

	// Stopping again should fail
	_, err = rec.StopRecording(sessionID)
	assert.Error(t, err)
}

func TestWriteOutputDuringRecording(t *testing.T) {
	repo := &mockRepo{}
	rec := recorder.NewRecorder("", repo)

	sessionID := uuid.New()
	err := rec.StartRecording(sessionID, model.RecordingFormatAsciinema, 80, 24)
	require.NoError(t, err)

	ctx := context.Background()
	err = rec.WriteOutput(ctx, sessionID, model.OutputTypeStdout, []byte("hello world"))
	require.NoError(t, err)

	assert.Len(t, repo.outputs, 1)
	assert.Equal(t, "hello world", string(repo.outputs[0].Data))
	assert.Equal(t, 1, repo.outputs[0].SequenceNum)

	recording, err := rec.StopRecording(sessionID)
	require.NoError(t, err)
	assert.Greater(t, recording.FileSize, int64(0))
}

func TestWriteOutputWithoutRecording(t *testing.T) {
	repo := &mockRepo{}
	rec := recorder.NewRecorder("", repo)

	sessionID := uuid.New()
	ctx := context.Background()
	err := rec.WriteOutput(ctx, sessionID, model.OutputTypeStderr, []byte("error msg"))
	require.NoError(t, err)

	assert.Len(t, repo.outputs, 1)
	assert.Equal(t, "error msg", string(repo.outputs[0].Data))
}

func TestWriteOutputWithNilRepo(t *testing.T) {
	rec := recorder.NewRecorder("", nil)

	sessionID := uuid.New()
	ctx := context.Background()
	err := rec.WriteOutput(ctx, sessionID, model.OutputTypeStdout, []byte("data"))
	require.NoError(t, err)
}

func TestGetAsciinemaData(t *testing.T) {
	repo := &mockRepo{}
	rec := recorder.NewRecorder("", repo)

	sessionID := uuid.New()
	err := rec.StartRecording(sessionID, model.RecordingFormatAsciinema, 80, 24)
	require.NoError(t, err)

	ctx := context.Background()
	err = rec.WriteOutput(ctx, sessionID, model.OutputTypeStdout, []byte("hello"))
	require.NoError(t, err)

	data, err := rec.GetAsciinemaData(sessionID)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// First line should be valid JSON header
	lines := splitLines(data)
	require.True(t, len(lines) > 0)
	var header map[string]interface{}
	require.NoError(t, json.Unmarshal(lines[0], &header))
	assert.Equal(t, float64(2), header["version"])
	assert.Equal(t, float64(80), header["width"])
	assert.Equal(t, float64(24), header["height"])

	// Second line should be an output event
	require.True(t, len(lines) > 1)
	var event []interface{}
	require.NoError(t, json.Unmarshal(lines[1], &event))
	require.Len(t, event, 3)
	assert.Equal(t, "o", event[1])
	assert.Equal(t, "hello", event[2])
}

func TestGetPlaybackData(t *testing.T) {
	repo := &mockRepo{}
	rec := recorder.NewRecorder("", repo)

	sessionID := uuid.New()
	err := rec.StartRecording(sessionID, model.RecordingFormatAsciinema, 80, 24)
	require.NoError(t, err)

	ctx := context.Background()
	rec.WriteOutput(ctx, sessionID, model.OutputTypeStdout, []byte("first"))
	rec.WriteOutput(ctx, sessionID, model.OutputTypeStdout, []byte("second"))

	chunks, err := rec.GetPlaybackData(sessionID, 0)
	require.NoError(t, err)
	assert.Len(t, chunks, 2)
	assert.Equal(t, "first", chunks[0].Data)
	assert.Equal(t, "second", chunks[1].Data)

	chunks, err = rec.GetPlaybackData(sessionID, 1)
	require.NoError(t, err)
	assert.Len(t, chunks, 1)
	assert.Equal(t, "second", chunks[0].Data)
}

func TestGetPlaybackDataNotAsciinema(t *testing.T) {
	repo := &mockRepo{}
	rec := recorder.NewRecorder("", repo)

	sessionID := uuid.New()
	err := rec.StartRecording(sessionID, model.RecordingFormatRaw, 80, 24)
	require.NoError(t, err)

	_, err = rec.GetPlaybackData(sessionID, 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "playback only supported for asciinema format")
}

func TestRawRecording(t *testing.T) {
	repo := &mockRepo{}
	rec := recorder.NewRecorder("", repo)

	sessionID := uuid.New()
	err := rec.StartRecording(sessionID, model.RecordingFormatRaw, 80, 24)
	require.NoError(t, err)

	ctx := context.Background()
	rec.WriteOutput(ctx, sessionID, model.OutputTypeStdout, []byte("raw data"))

	recording, err := rec.StopRecording(sessionID)
	require.NoError(t, err)
	assert.Equal(t, model.RecordingFormatRaw, recording.Format)
	assert.Greater(t, recording.FileSize, int64(0))
}

func TestGetAsciinemaDataNoRecording(t *testing.T) {
	rec := recorder.NewRecorder("", nil)
	_, err := rec.GetAsciinemaData(uuid.New())
	assert.Error(t, err)
}

func TestGetPlaybackDataNoRecording(t *testing.T) {
	rec := recorder.NewRecorder("", nil)
	_, err := rec.GetPlaybackData(uuid.New(), 0)
	assert.Error(t, err)
}

func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i := 0; i < len(data); i++ {
		if data[i] == '\n' {
			lines = append(lines, data[start:i])
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}

func TestExtForFormat(t *testing.T) {
	rec := recorder.NewRecorder("", nil)

	// Use StopRecording to verify file extension indirectly via file path
	sessionID := uuid.New()
	rec.StartRecording(sessionID, model.RecordingFormatAsciinema, 80, 24)
	rec.StopRecording(sessionID)

	sessionID = uuid.New()
	rec.StartRecording(sessionID, model.RecordingFormatRaw, 80, 24)
	rec.StopRecording(sessionID)

	sessionID = uuid.New()
	rec.StartRecording(sessionID, model.RecordingFormatHTML, 80, 24)
	rec.StopRecording(sessionID)

	// Test unknown format defaults to txt
	sessionID = uuid.New()
	rec.StartRecording(sessionID, model.RecordingFormat("unknown"), 80, 24)
	recording, _ := rec.StopRecording(sessionID)
	assert.Contains(t, recording.FilePath, ".txt")
}

func TestRecordingDuration(t *testing.T) {
	repo := &mockRepo{}
	rec := recorder.NewRecorder("", repo)

	sessionID := uuid.New()
	err := rec.StartRecording(sessionID, model.RecordingFormatAsciinema, 80, 24)
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	recording, err := rec.StopRecording(sessionID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, recording.DurationMs, 50)
}
