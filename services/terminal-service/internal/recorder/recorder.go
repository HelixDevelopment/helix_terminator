package recorder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/helixdevelopment/terminal-service/internal/model"
)

// RecordingState holds the in-memory state for an active recording.
type RecordingState struct {
	SessionID   uuid.UUID
	Format      model.RecordingFormat
	StartedAt   time.Time
	Buffer      bytes.Buffer
	SequenceNum int
	Cols        int
	Rows        int
	mu          sync.Mutex
}

// Recorder manages terminal output buffering and recording.
type Recorder struct {
	mu         sync.RWMutex
	recordings map[uuid.UUID]*RecordingState
	outputDir  string
	repo       OutputRepository
}

// OutputRepository defines the minimal DB interface the recorder needs.
type OutputRepository interface {
	CreateOutput(ctx context.Context, output *model.TerminalOutput) error
}

// NewRecorder creates a new Recorder.
func NewRecorder(outputDir string, repo OutputRepository) *Recorder {
	if outputDir == "" {
		outputDir = "/tmp/terminal-recordings"
	}
	_ = os.MkdirAll(outputDir, 0755)
	return &Recorder{
		recordings: make(map[uuid.UUID]*RecordingState),
		outputDir:  outputDir,
		repo:       repo,
	}
}

// WriteOutput buffers terminal output and optionally persists it.
func (rec *Recorder) WriteOutput(ctx context.Context, sessionID uuid.UUID, outputType model.OutputType, data []byte) error {
	rec.mu.RLock()
	state, ok := rec.recordings[sessionID]
	rec.mu.RUnlock()

	if ok && state != nil {
		state.mu.Lock()
		state.SequenceNum++
		seq := state.SequenceNum

		if state.Format == model.RecordingFormatAsciinema {
			// ASCIInema v2 line: [time, "o", "data"]
			elapsed := time.Since(state.StartedAt).Seconds()
			line, _ := json.Marshal([]interface{}{elapsed, "o", string(data)})
			state.Buffer.Write(line)
			state.Buffer.WriteByte('\n')
		} else if state.Format == model.RecordingFormatRaw {
			state.Buffer.Write(data)
		}
		state.mu.Unlock()

		// Persist to DB
		if rec.repo != nil {
			output := &model.TerminalOutput{
				ID:          uuid.New(),
				SessionID:   sessionID,
				OutputType:  outputType,
				Data:        data,
				Timestamp:   time.Now().UTC(),
				SequenceNum: seq,
			}
			_ = rec.repo.CreateOutput(ctx, output)
		}
		return nil
	}

	// Not recording; just persist to DB
	if rec.repo == nil {
		return nil
	}
	output := &model.TerminalOutput{
		ID:          uuid.New(),
		SessionID:   sessionID,
		OutputType:  outputType,
		Data:        data,
		Timestamp:   time.Now().UTC(),
		SequenceNum: 0,
	}
	return rec.repo.CreateOutput(ctx, output)
}

// StartRecording begins recording a session in the specified format.
func (rec *Recorder) StartRecording(sessionID uuid.UUID, format model.RecordingFormat, cols, rows int) error {
	rec.mu.Lock()
	defer rec.mu.Unlock()

	if _, exists := rec.recordings[sessionID]; exists {
		return fmt.Errorf("recording already started for session %s", sessionID)
	}

	state := &RecordingState{
		SessionID:   sessionID,
		Format:      format,
		StartedAt:   time.Now().UTC(),
		SequenceNum: 0,
		Cols:        cols,
		Rows:        rows,
	}

	if format == model.RecordingFormatAsciinema {
		// Write ASCIInema v2 header
		header := map[string]interface{}{
			"version":   2,
			"width":     cols,
			"height":    rows,
			"timestamp": state.StartedAt.Unix(),
			"env": map[string]string{
				"SHELL": "/bin/bash",
				"TERM":  "xterm-256color",
			},
		}
		headerBytes, _ := json.Marshal(header)
		state.Buffer.Write(headerBytes)
		state.Buffer.WriteByte('\n')
	}

	rec.recordings[sessionID] = state
	return nil
}

// StopRecording finalizes a recording and saves it to disk.
func (rec *Recorder) StopRecording(sessionID uuid.UUID) (*model.TerminalRecording, error) {
	rec.mu.Lock()
	state, ok := rec.recordings[sessionID]
	if !ok {
		rec.mu.Unlock()
		return nil, fmt.Errorf("no active recording for session %s", sessionID)
	}
	delete(rec.recordings, sessionID)
	rec.mu.Unlock()

	state.mu.Lock()
	defer state.mu.Unlock()

	elapsedMs := int(time.Since(state.StartedAt).Milliseconds())
	fileName := fmt.Sprintf("%s.%s", sessionID, rec.extForFormat(state.Format))
	filePath := filepath.Join(rec.outputDir, fileName)

	data := state.Buffer.Bytes()
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to write recording file: %w", err)
	}

	recording := &model.TerminalRecording{
		ID:         uuid.New(),
		SessionID:  sessionID,
		Format:     state.Format,
		FilePath:   filePath,
		FileSize:   int64(len(data)),
		DurationMs: elapsedMs,
		CreatedAt:  time.Now().UTC(),
	}
	return recording, nil
}

// GetPlaybackData returns output chunks for playback.
func (rec *Recorder) GetPlaybackData(sessionID uuid.UUID, fromSequence int) ([]model.OutputChunk, error) {
	rec.mu.RLock()
	state, ok := rec.recordings[sessionID]
	rec.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("no active recording for session %s", sessionID)
	}

	state.mu.Lock()
	defer state.mu.Unlock()

	if state.Format != model.RecordingFormatAsciinema {
		return nil, fmt.Errorf("playback only supported for asciinema format")
	}

	var chunks []model.OutputChunk
	lines := bytes.Split(state.Buffer.Bytes(), []byte("\n"))
	seq := 0
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		// Skip header
		if line[0] == '{' {
			continue
		}
		seq++
		if seq <= fromSequence {
			continue
		}
		var entry []interface{}
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}
		if len(entry) >= 3 {
			dataStr, _ := entry[2].(string)
			chunks = append(chunks, model.OutputChunk{
				OutputType: model.OutputTypeStdout,
				Data:       dataStr,
			})
		}
	}

	return chunks, nil
}

// GetAsciinemaData returns the raw ASCIInema v2 formatted recording data.
func (rec *Recorder) GetAsciinemaData(sessionID uuid.UUID) ([]byte, error) {
	rec.mu.RLock()
	state, ok := rec.recordings[sessionID]
	rec.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("no active recording for session %s", sessionID)
	}

	state.mu.Lock()
	defer state.mu.Unlock()

	data := make([]byte, state.Buffer.Len())
	copy(data, state.Buffer.Bytes())
	return data, nil
}

// IsRecording returns true if the session is actively being recorded.
func (rec *Recorder) IsRecording(sessionID uuid.UUID) bool {
	rec.mu.RLock()
	_, ok := rec.recordings[sessionID]
	rec.mu.RUnlock()
	return ok
}

func (rec *Recorder) extForFormat(format model.RecordingFormat) string {
	switch format {
	case model.RecordingFormatAsciinema:
		return "cast"
	case model.RecordingFormatRaw:
		return "raw"
	case model.RecordingFormatHTML:
		return "html"
	default:
		return "txt"
	}
}
