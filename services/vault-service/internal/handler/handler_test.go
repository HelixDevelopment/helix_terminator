package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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
	secrets          map[uuid.UUID]*model.Secret
	versions         map[uuid.UUID][]*model.SecretVersion
	createSecretErr  error
	getSecretErr     error
	listSecretsErr   error
	updateSecretErr  error
	deleteSecretErr  error
	createVersionErr error
	getVersionsErr   error
	countErr         error
	pingErr          error
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		secrets:  make(map[uuid.UUID]*model.Secret),
		versions: make(map[uuid.UUID][]*model.SecretVersion),
	}
}

func (m *mockRepo) CreateSecret(ctx context.Context, secret *model.Secret) error {
	if m.createSecretErr != nil {
		return m.createSecretErr
	}
	m.secrets[secret.ID] = secret
	return nil
}

func (m *mockRepo) GetSecretByID(ctx context.Context, id uuid.UUID) (*model.Secret, error) {
	if m.getSecretErr != nil {
		return nil, m.getSecretErr
	}
	s, ok := m.secrets[id]
	if !ok {
		return nil, assert.AnError
	}
	return s, nil
}

func (m *mockRepo) ListSecrets(ctx context.Context, userID, orgID uuid.UUID, secretType model.SecretType, tags []string, limit, offset int) ([]*model.Secret, error) {
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

func (m *mockRepo) UpdateSecret(ctx context.Context, secret *model.Secret) error {
	if m.updateSecretErr != nil {
		return m.updateSecretErr
	}
	m.secrets[secret.ID] = secret
	return nil
}

func (m *mockRepo) DeleteSecret(ctx context.Context, id uuid.UUID) error {
	if m.deleteSecretErr != nil {
		return m.deleteSecretErr
	}
	if s, ok := m.secrets[id]; ok {
		now := time.Now().UTC()
		s.DeletedAt = &now
	}
	return nil
}

func (m *mockRepo) CreateSecretVersion(ctx context.Context, version *model.SecretVersion) error {
	if m.createVersionErr != nil {
		return m.createVersionErr
	}
	m.versions[version.SecretID] = append(m.versions[version.SecretID], version)
	return nil
}

func (m *mockRepo) GetSecretVersions(ctx context.Context, secretID uuid.UUID, limit int) ([]*model.SecretVersion, error) {
	if m.getVersionsErr != nil {
		return nil, m.getVersionsErr
	}
	return m.versions[secretID], nil
}

func (m *mockRepo) CountSecrets(ctx context.Context, userID, orgID uuid.UUID) (int, error) {
	if m.countErr != nil {
		return 0, m.countErr
	}
	return len(m.secrets), nil
}

func (m *mockRepo) Ping(ctx context.Context) error {
	return m.pingErr
}

func TestHealthCheck(t *testing.T) {
	repo := newMockRepo()
	h := handler.New(repo)
	r := setupRouter(h)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/healthz", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "healthy", resp["status"])
}

func TestReadinessCheck(t *testing.T) {
	repo := newMockRepo()
	h := handler.New(repo)
	r := setupRouter(h)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/healthz/ready", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCreateSecret(t *testing.T) {
	repo := newMockRepo()
	h := handler.New(repo)
	r := setupRouter(h)

	reqBody := model.CreateSecretRequest{
		UserID:         uuid.MustParse("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12"),
		OrgID:          uuid.MustParse("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
		Name:           "test-secret",
		Type:           "api_token",
		EncryptedValue: "encrypted-data",
		IV:             "iv-data",
		Salt:           "salt-data",
		Metadata:       map[string]interface{}{"key": "value"},
		Tags:           []string{"prod"},
	}
	body, _ := json.Marshal(reqBody)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/vault/secrets", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	// T7: CreateSecret derives the secret's owner from the authenticated
	// caller (X-User-ID), which must match the body's user_id.
	req.Header.Set("X-User-ID", reqBody.UserID.String())
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp model.SecretResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, reqBody.Name, resp.Name)
	assert.Equal(t, reqBody.Type, resp.Type)
	assert.Equal(t, reqBody.UserID, resp.UserID)
}

func TestCreateSecretValidation(t *testing.T) {
	repo := newMockRepo()
	h := handler.New(repo)
	r := setupRouter(h)

	reqBody := model.CreateSecretRequest{
		Name: "",
	}
	body, _ := json.Marshal(reqBody)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/vault/secrets", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", uuid.New().String())
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestCreateSecret_RejectsMissingCallerIdentity proves CreateSecret rejects
// a request with no X-User-ID header at all (T7).
func TestCreateSecret_RejectsMissingCallerIdentity(t *testing.T) {
	repo := newMockRepo()
	h := handler.New(repo)
	r := setupRouter(h)

	reqBody := model.CreateSecretRequest{
		UserID:         uuid.New(),
		Name:           "test-secret",
		Type:           "api_token",
		EncryptedValue: "encrypted-data",
		IV:             "iv-data",
		Salt:           "salt-data",
	}
	body, _ := json.Marshal(reqBody)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/vault/secrets", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Empty(t, repo.secrets, "no secret must be persisted when caller identity is missing")
}

// TestCreateSecret_RejectsBodyUserIDMismatch is the core T7 IDOR proof at
// the handler level: a caller authenticated as tenant B cannot create a
// secret owned by tenant A merely by putting tenant A's user_id in the body.
func TestCreateSecret_RejectsBodyUserIDMismatch(t *testing.T) {
	repo := newMockRepo()
	h := handler.New(repo)
	r := setupRouter(h)

	tenantA := uuid.New()
	tenantB := uuid.New()

	reqBody := model.CreateSecretRequest{
		UserID:         tenantA, // attacker-supplied victim tenant
		Name:           "stolen-secret",
		Type:           "api_token",
		EncryptedValue: "encrypted-data",
		IV:             "iv-data",
		Salt:           "salt-data",
	}
	body, _ := json.Marshal(reqBody)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/vault/secrets", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", tenantB.String()) // real authenticated caller
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code,
		"tenant B was able to create a secret owned by tenant A — IDOR is present")
	assert.Empty(t, repo.secrets, "no secret must be persisted on a caller/body user_id mismatch")
}

func TestGetSecret(t *testing.T) {
	repo := newMockRepo()
	id := uuid.MustParse("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")
	repo.secrets[id] = &model.Secret{
		ID:   id,
		Name: "test-secret",
		Type: model.SecretTypeAPIToken,
	}
	h := handler.New(repo)
	r := setupRouter(h)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/vault/secrets/"+id.String(), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp model.SecretResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, id, resp.ID)
}

func TestGetSecretNotFound(t *testing.T) {
	repo := newMockRepo()
	h := handler.New(repo)
	r := setupRouter(h)

	id := uuid.MustParse("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a99")
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/vault/secrets/"+id.String(), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestListSecrets(t *testing.T) {
	repo := newMockRepo()
	caller := uuid.New()
	otherTenant := uuid.New()
	s1ID, s2ID := uuid.New(), uuid.New()
	repo.secrets[s1ID] = &model.Secret{ID: s1ID, UserID: caller, Name: "secret1", Type: model.SecretTypeAPIToken}
	repo.secrets[s2ID] = &model.Secret{ID: s2ID, UserID: caller, Name: "secret2", Type: model.SecretTypePassword}
	// A foreign tenant's secret MUST NOT appear in caller's list (T7 IDOR proof).
	foreignID := uuid.New()
	repo.secrets[foreignID] = &model.Secret{ID: foreignID, UserID: otherTenant, Name: "not-mine", Type: model.SecretTypeAPIToken}
	h := handler.New(repo)
	r := setupRouter(h)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/vault/secrets", nil)
	req.Header.Set("X-User-ID", caller.String())
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp model.ListSecretsResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Len(t, resp.Secrets, 2, "must return exactly the caller's own secrets, not the foreign tenant's")
	for _, s := range resp.Secrets {
		assert.Equal(t, caller, s.UserID)
		assert.NotEqual(t, foreignID, s.ID)
	}
}

// TestListSecrets_RejectsMissingCallerIdentity proves ListSecrets rejects a
// request with no X-User-ID header — previously an omitted user_id query
// param fell through to the repository's "no filter" behaviour, returning
// EVERY tenant's secrets (T7).
func TestListSecrets_RejectsMissingCallerIdentity(t *testing.T) {
	repo := newMockRepo()
	repo.secrets[uuid.New()] = &model.Secret{ID: uuid.New(), UserID: uuid.New(), Name: "someone-elses-secret"}
	h := handler.New(repo)
	r := setupRouter(h)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/vault/secrets", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestListSecrets_RejectsQueryUserIDMismatch is the core T7 IDOR proof: a
// caller authenticated as tenant B cannot list tenant A's secrets by
// passing tenant A's user_id as a query parameter.
func TestListSecrets_RejectsQueryUserIDMismatch(t *testing.T) {
	repo := newMockRepo()
	tenantA := uuid.New()
	tenantB := uuid.New()
	secretID := uuid.New()
	repo.secrets[secretID] = &model.Secret{ID: secretID, UserID: tenantA, Name: "tenant-a-secret"}
	h := handler.New(repo)
	r := setupRouter(h)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/vault/secrets?user_id="+tenantA.String(), nil)
	req.Header.Set("X-User-ID", tenantB.String()) // real authenticated caller
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code,
		"tenant B was able to list tenant A's secrets via the user_id query param — IDOR is present")
}

func TestUpdateSecret(t *testing.T) {
	repo := newMockRepo()
	id := uuid.MustParse("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")
	repo.secrets[id] = &model.Secret{ID: id, Name: "old-name", Type: model.SecretTypeAPIToken}
	h := handler.New(repo)
	r := setupRouter(h)

	reqBody := model.UpdateSecretRequest{
		Name: "new-name",
	}
	body, _ := json.Marshal(reqBody)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/vault/secrets/"+id.String(), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDeleteSecret(t *testing.T) {
	repo := newMockRepo()
	id := uuid.MustParse("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")
	repo.secrets[id] = &model.Secret{ID: id, Name: "test", Type: model.SecretTypeAPIToken}
	h := handler.New(repo)
	r := setupRouter(h)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/vault/secrets/"+id.String(), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.NotNil(t, repo.secrets[id].DeletedAt)
}

func TestRotateSecret(t *testing.T) {
	repo := newMockRepo()
	id := uuid.MustParse("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")
	repo.secrets[id] = &model.Secret{ID: id, Name: "test", Type: model.SecretTypeAPIToken, EncryptedValue: "old"}
	h := handler.New(repo)
	r := setupRouter(h)

	reqBody := map[string]interface{}{"encrypted_value": "new-value", "iv": "new-iv", "salt": "new-salt", "created_by": uuid.New().String()}
	body, _ := json.Marshal(reqBody)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/vault/secrets/"+id.String()+"/rotate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "new-value", repo.secrets[id].EncryptedValue)
	assert.Len(t, repo.versions[id], 1)
}

func TestGetSecretVersions(t *testing.T) {
	repo := newMockRepo()
	id := uuid.MustParse("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")
	repo.versions[id] = []*model.SecretVersion{
		{ID: uuid.New(), SecretID: id, EncryptedValue: "v1"},
		{ID: uuid.New(), SecretID: id, EncryptedValue: "v2"},
	}
	h := handler.New(repo)
	r := setupRouter(h)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/vault/secrets/"+id.String()+"/versions", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp model.ListSecretVersionsResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp.Versions, 2)
}
