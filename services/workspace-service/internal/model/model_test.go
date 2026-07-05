package model_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/helixdevelopment/workspace-service/internal/model"
)

func TestWorkspaceModel(t *testing.T) {
	ws := model.Workspace{
		ID:     uuid.New(),
		OrgID:  uuid.New(),
		UserID: uuid.New(),
		Name:   "test-workspace",
		Tags:   []string{"prod", "eu-west"},
	}
	assert.NotEqual(t, uuid.Nil, ws.ID)
	assert.Equal(t, "test-workspace", ws.Name)
	assert.Len(t, ws.Tags, 2)
}

func TestCreateWorkspaceRequest(t *testing.T) {
	req := model.CreateWorkspaceRequest{
		Name:        "dev",
		Description: "dev environment",
		Color:       "#ff0000",
		Icon:        "server",
		Tags:        []string{"dev"},
	}
	assert.Equal(t, "dev", req.Name)
	assert.Equal(t, "#ff0000", req.Color)
}

func TestWorkspaceHostModel(t *testing.T) {
	wh := model.WorkspaceHost{
		WorkspaceID: uuid.New(),
		HostID:      uuid.New(),
		AddedBy:     uuid.New(),
	}
	assert.NotEqual(t, uuid.Nil, wh.WorkspaceID)
	assert.NotEqual(t, uuid.Nil, wh.HostID)
}
