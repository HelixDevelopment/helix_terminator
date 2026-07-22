package model_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/helixdevelopment/gateway-service/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NOTE (§11.4.124 dead/unwired-code investigation, captured before this
// test was written): internal/model is genuinely UNWIRED scaffolding.
// `git log --follow` shows this package was added in a single MVP
// scaffold commit and has never been imported by any other file in
// gateway-service (grep across services/gateway-service for
// "gateway-service/internal/model" matches only this test). main.go
// wires internal/server directly; gateway-service is a routing/proxy
// service that does not (yet) own domain entities of its own. Per
// §11.4.124 this is surfaced, not silently deleted - removal requires
// operator confirmation (§11.4.122) since the package is exported. This
// test exercises the REAL, CURRENT behaviour of the code that exists
// (JSON tag correctness) rather than inventing behaviour that isn't
// there.
func TestBaseModel_JSONRoundTrip(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	original := model.BaseModel{
		ID:        "base-1",
		CreatedAt: now,
		UpdatedAt: now,
	}

	raw, err := json.Marshal(original)
	require.NoError(t, err)
	assert.JSONEq(t, `{"id":"base-1","created_at":"2026-01-02T03:04:05Z","updated_at":"2026-01-02T03:04:05Z"}`, string(raw))

	var decoded model.BaseModel
	require.NoError(t, json.Unmarshal(raw, &decoded))
	assert.Equal(t, original.ID, decoded.ID)
	assert.True(t, original.CreatedAt.Equal(decoded.CreatedAt))
	assert.True(t, original.UpdatedAt.Equal(decoded.UpdatedAt))
}

// TestBaseModel_ZeroValueMarshalsExplicitTimestamps proves BaseModel has
// no omitempty tags - a zero-valued instance still emits every field
// (including the zero time.Time) rather than silently dropping it. This
// is the actual current contract; a future author adding omitempty
// would change the wire shape and this test would catch it.
func TestBaseModel_ZeroValueMarshalsExplicitTimestamps(t *testing.T) {
	var zero model.BaseModel
	raw, err := json.Marshal(zero)
	require.NoError(t, err)

	var asMap map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &asMap))

	for _, key := range []string{"id", "created_at", "updated_at"} {
		_, present := asMap[key]
		assert.True(t, present, "expected %q to always be present (no omitempty on BaseModel), got: %s", key, raw)
	}
}
