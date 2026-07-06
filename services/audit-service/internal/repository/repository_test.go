package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/helixdevelopment/audit-service/internal/model"
	"github.com/helixdevelopment/audit-service/internal/repository"
)

func TestRepositoryCheckPool(t *testing.T) {
	repo := repository.New(nil)
	assert.NotNil(t, repo)

	ctx := context.Background()
	log := &model.AuditLog{
		ID:        uuid.New(),
		Action:    model.ActionCreate,
		Timestamp: time.Now().UTC(),
		Severity:  model.SeverityInfo,
	}

	err := repo.CreateAuditLog(ctx, log)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database not connected")

	_, err = repo.GetAuditLogByID(ctx, uuid.New())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database not connected")

	_, _, err = repo.ListAuditLogs(ctx, nil, nil, "", "", "", nil, nil, 20, 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database not connected")

	_, err = repo.CountByAction(ctx, nil, nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database not connected")

	_, err = repo.CountByResourceType(ctx, nil, nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database not connected")

	err = repo.Ping(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database not connected")
}
