package model_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/helixdevelopment/audit-service/internal/model"
)

func TestAuditLogCreation(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	resourceID := uuid.New()
	now := time.Now().UTC()

	log := model.AuditLog{
		ID:           uuid.New(),
		OrgID:        &orgID,
		UserID:       &userID,
		Action:       model.ActionCreate,
		ResourceType: model.ResourceTypeUser,
		ResourceID:   &resourceID,
		Details:      []byte(`{"key":"value"}`),
		IPAddress:    "192.168.1.1",
		UserAgent:    "Mozilla/5.0",
		Timestamp:    now,
		Severity:     model.SeverityInfo,
	}

	assert.Equal(t, model.ActionCreate, log.Action)
	assert.Equal(t, model.ResourceTypeUser, log.ResourceType)
	assert.Equal(t, model.SeverityInfo, log.Severity)
	assert.Equal(t, "192.168.1.1", log.IPAddress)
	assert.Equal(t, "Mozilla/5.0", log.UserAgent)
	assert.Equal(t, now, log.Timestamp)
	assert.Equal(t, &orgID, log.OrgID)
	assert.Equal(t, &userID, log.UserID)
	assert.Equal(t, &resourceID, log.ResourceID)
}

func TestCreateAuditLogRequestValidation(t *testing.T) {
	req := model.CreateAuditLogRequest{
		Action:       model.ActionUpdate,
		ResourceType: model.ResourceTypeWorkspace,
		Severity:     model.SeverityWarning,
		Details: map[string]interface{}{
			"field": "value",
		},
	}

	assert.Equal(t, model.ActionUpdate, req.Action)
	assert.Equal(t, model.ResourceTypeWorkspace, req.ResourceType)
	assert.Equal(t, model.SeverityWarning, req.Severity)
	assert.NotNil(t, req.Details)
}

func TestListAuditLogsRequestDefaults(t *testing.T) {
	req := model.ListAuditLogsRequest{
		Limit:  20,
		Offset: 0,
	}

	assert.Equal(t, 20, req.Limit)
	assert.Equal(t, 0, req.Offset)
}

func TestAuditLogResponse(t *testing.T) {
	logID := uuid.New()
	resp := model.AuditLogResponse{
		ID:           logID,
		Action:       model.ActionDelete,
		ResourceType: model.ResourceTypeVault,
		Severity:     model.SeverityCritical,
		IPAddress:    "10.0.0.1",
		Timestamp:    time.Now().UTC(),
	}

	assert.Equal(t, logID, resp.ID)
	assert.Equal(t, model.ActionDelete, resp.Action)
	assert.Equal(t, model.ResourceTypeVault, resp.ResourceType)
	assert.Equal(t, model.SeverityCritical, resp.Severity)
}

func TestListAuditLogsResponse(t *testing.T) {
	resp := model.ListAuditLogsResponse{
		Logs:   []*model.AuditLogResponse{},
		Total:  0,
		Limit:  20,
		Offset: 0,
	}

	assert.Empty(t, resp.Logs)
	assert.Equal(t, 0, resp.Total)
	assert.Equal(t, 20, resp.Limit)
	assert.Equal(t, 0, resp.Offset)
}

func TestCountResponse(t *testing.T) {
	resp := model.CountResponse{
		Counts: map[string]int{
			"create": 5,
			"read":   10,
		},
	}

	assert.Equal(t, 5, resp.Counts["create"])
	assert.Equal(t, 10, resp.Counts["read"])
}
