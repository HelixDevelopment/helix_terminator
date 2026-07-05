package model_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/helixdevelopment/pki-service/internal/model"
)

func TestCertificateStatusValues(t *testing.T) {
	assert.Equal(t, model.CertificateStatus("active"), model.StatusActive)
	assert.Equal(t, model.CertificateStatus("expired"), model.StatusExpired)
	assert.Equal(t, model.CertificateStatus("revoked"), model.StatusRevoked)
}

func TestCreateCARequestValidation(t *testing.T) {
	req := model.CreateCARequest{
		OrgID:        "550e8400-e29b-41d4-a716-446655440000",
		Name:         "Test CA",
		Description:  "A test CA",
		ValidityDays: 365,
	}
	assert.Equal(t, "Test CA", req.Name)
	assert.Equal(t, 365, req.ValidityDays)
}

func TestCreateCertRequestValidation(t *testing.T) {
	req := model.CreateCertRequest{
		Name:         "Test Cert",
		Subject:      "CN=Test Cert,O=Helix,C=US",
		ValidityDays: 30,
	}
	assert.Equal(t, "Test Cert", req.Name)
	assert.Equal(t, 30, req.ValidityDays)
}

func TestListCertsRequestDefaults(t *testing.T) {
	req := model.ListCertsRequest{
		Limit:  20,
		Offset: 0,
	}
	assert.Equal(t, 20, req.Limit)
	assert.Equal(t, 0, req.Offset)
}
