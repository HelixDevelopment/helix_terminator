package model_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/helixdevelopment/org-service/internal/model"
)

func TestPlanConstants(t *testing.T) {
	assert.Equal(t, model.Plan("free"), model.PlanFree)
	assert.Equal(t, model.Plan("pro"), model.PlanPro)
	assert.Equal(t, model.Plan("enterprise"), model.PlanEnterprise)
}

func TestRoleConstants(t *testing.T) {
	assert.Equal(t, model.Role("owner"), model.RoleOwner)
	assert.Equal(t, model.Role("admin"), model.RoleAdmin)
	assert.Equal(t, model.Role("member"), model.RoleMember)
}
