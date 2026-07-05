package model_test

import (
	"testing"

	"github.com/helixdevelopment/terminal-service/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestTerminalStatusConstants(t *testing.T) {
	assert.Equal(t, model.TerminalStatusPending, model.TerminalStatus("pending"))
	assert.Equal(t, model.TerminalStatusActive, model.TerminalStatus("active"))
	assert.Equal(t, model.TerminalStatusClosed, model.TerminalStatus("closed"))
}

func TestOutputTypeConstants(t *testing.T) {
	assert.Equal(t, model.OutputTypeStdout, model.OutputType("stdout"))
	assert.Equal(t, model.OutputTypeStderr, model.OutputType("stderr"))
}

func TestRecordingFormatConstants(t *testing.T) {
	assert.Equal(t, model.RecordingFormatAsciinema, model.RecordingFormat("asciinema"))
	assert.Equal(t, model.RecordingFormatRaw, model.RecordingFormat("raw"))
}
