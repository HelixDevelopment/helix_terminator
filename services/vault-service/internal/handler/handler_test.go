package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helixdevelopment/vault-service/internal/handler"
	"github.com/helixdevelopment/vault-service/internal/model"
)

func setupRouter(h *handler.Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/healthz", h.HealthCheck)
	r.GET("/healthz/ready", h.ReadinessCheck)
	r.POST("/api/v1/vault/secrets", h.CreateSecret)
	r.GET("/api/v1/vault/secrets/:id", h.GetSecret)
	r.GET("/api/v1/vault/secrets", h.ListSecrets)
	r.PUT("/api/v1/vault/secrets/:id", h.UpdateSecret)
	r.DELETE("/api/v1/vault/secrets/:id", h.DeleteSecret)
	r.GET("/api/v1/vault/secrets/:id/versions", h.GetSecretVersions)
	r.POST("/api/v1/vault/secrets/:id/rotate", h.RotateSecret)
	return r
}

type mockRepo struct {
	secrets         map[uuid.UUID]*model.Secret
	versions        map[uuid.UUID][]*model.SecretVersion
	createSecretErr error
	getSecretErr    error
	listSecretsErr  error
	updateSecretErr error
	deleteSecretErr error
	createVersionErr error
	getVersionsErr  error
	countErr        error
	pingErr         error
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		secrets:  make(map[uuid.UUID]*model.Secret),
		versions: make(map[uuid.UUID][]*model.SecretVersion),
	}
}

func (m *mockRepo) CreateSecret(ctx interface{}, secret *model.Secret) error {
	if m.createSecretErr != nil {
		return m.createSecretErr
	}
	m.secrets[secret.ID] = secret
	return nil
}

func (m *mockRepo) GetSecretByID(ctx interface{}, id uuid.UUID) (*model.Secret, error) {
	if m.getSecretErr != nil {
		return nil, m.getSecretErr
	}
	s, ok := m.secrets[id]
	if !ok {
		return nil, assert.AnError
	}
	return s, nil
}

func (m *mockRepo) ListSecrets(ctx interface{}, userID, orgID uuid.UUID, secretType model.SecretType, tags []string, limit, offset int) ([]*model.Secret, error) {
	if m.listSecretsErr != nil {
		return nil, m.listSecretsErr
	}
	var result []*model.Secret
	for _, s := range m.secrets {
		if s.DeletedAt != nil {
			continue
		}
		if userID != uuid.Nil && s.UserID != userID {
			continue
		}
		if orgID != uuid.Nil && s.OrgID != orgID {
			continue
		}
		if secretType != "" && s.Type != secretType {
			continue
		}
		result = append(result, s)
	}
	return result, nil
}

func (m *mockRepo) UpdateSecret(ctx interface{}, secret *model.Secret) error {
	if m.updateSecretErr != nil {
		return m.updateSecretErr
	}
	m.secrets[secret.ID] = secret
	return nil
}

func (m *mockRepo) DeleteSecret(ctx interface{}, id uuid.UUID) error {
	if m.deleteSecretErr != nil {
		return m.deleteSecretErr
	}
	if s, ok := m.secrets[id]; ok {
		now := interface{}(nil)
		_ = now
		s.DeletedAt = &[]interface{}{}[0].(interface{})
	}
	return nil
}

func (m *mockRepo) CreateSecretVersion(ctx interface{}, version *model.SecretVersion) error {
	if m.createVersionErr != nil {
		return m.createVersionErr
	}
	m.versions[version.SecretID] = append(m.versions[version.SecretID], version)
	return nil
}

func (m *mockRepo) GetSecretVersions(ctx interface{}, secretID uuid.UUID, limit int) ([]*model.SecretVersion, error) {
	if m.getVersionsErr != nil {
		return nil, m.getVersionsErr
	}
	return m.versions[secretID], nil
}

func (m *mockRepo) CountSecrets(ctx interface{}, userID, orgID uuid.UUID) (int, error) {
	if m.countErr != nil {
		return 0, m.countErr
	}
	count := 0
	for _, s := range m.secrets {
		if s.DeletedAt != nil {
			continue
		}
		if userID != uuid.Nil && s.UserID != userID {
			continue
		}
		if orgID != uuid.Nil && s.OrgID != orgID {
			continue
		}
		count++
	}
	return count, nil
}

func (m *mockRepo) Ping(ctx interface{}) error {
	return m.pingErr
}

func TestHealthCheck(t *testing.T) {
	repo := newMockRepo()
	h := handler.New(repo)
	r := setupRouter(h)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/healthz", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "healthy", body["status"])
	assert.Equal(t, "vault-service", body["service"])
}

func TestReadinessCheck_Ready(t *testing.T) {
	repo := newMockRepo()
	h := handler.New(repo)
	r := setupRouter(h)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/healthz/ready", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, true, body["ready"])
}

func TestReadinessCheck_NotReady(t *testing.T) {
	repo := newMockRepo()
	repo.pingErr = assert.AnError
	h := handler.New(repo)
	r := setupRouter(h)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/healthz/ready", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestCreateSecret_Validation(t *testing.T) {
	repo := newMockRepo()
	h := handler.New(repo)
	r := setupRouter(h)

	// Missing required fields
	reqBody := map[string]interface{}{
		"name": "test-secret",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/vault/secrets", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateSecret_Success(t *testing.T) {
	repo := newMockRepo()
	h := handler.New(repo)
	r := setupRouter(h)

	userID := uuid.New()
	reqBody := model.CreateSecretRequest{
		UserID:         userID,
		Name:           "my-api-key",
		Type:           "api_token",
		EncryptedValue: "ciphertext",
		IV:             "iv123",
		Salt:           "salt456",
		Tags:           []string{"prod", "api"},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/vault/secrets", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp model.SecretResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "my-api-key", resp.Name)
	assert.Equal(t, "api_token", resp.Type)
	assert.Equal(t, userID, resp.UserID)
}

func TestGetSecret_NotFound(t *testing.T) {
	repo := newMockRepo()
	h := handler.New(repo)
	r := setupRouter(h)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/vault/secrets/"+uuid.New().String(), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetSecret_InvalidID(t *testing.T) {
	repo := newMockRepo()
	h := handler.New(repo)
	r := setupRouter(h)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/vault/secrets/not-a-uuid", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteSecret_InvalidID(t *testing.T) {
	repo := newMockRepo()
	h := handler.New(repo)
	r := setupRouter(h)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/vault/secrets/bad-id", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRotateSecret_InvalidID(t *testing.T) {
	repo := newMockRepo()
	h := handler.New(repo)
	r := setupRouter(h)

	reqBody := map[string]interface{}{
		"encrypted_value": "newcipher",
		"iv":              "newiv",
		"salt":            "newsalt",
		"created_by":      uuid.New().String(),
	}
	bodyBytes, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/vault/secrets/bad-id/rotate", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListSecrets_Validation(t *testing.T) {
	repo := newMockRepo()
	h := handler.New(repo)
	r := setupRouter(h)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/vault/secrets?limit=0", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
