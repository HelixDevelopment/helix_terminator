package model

import (
	"time"

	"github.com/google/uuid"
)

// SecretType represents the type of a secret.
type SecretType string

const (
	SecretTypeSSHKey      SecretType = "ssh_key"
	SecretTypeAPIToken    SecretType = "api_token"
	SecretTypePassword    SecretType = "password"
	SecretTypeCertificate SecretType = "certificate"
	SecretTypeEnvVar      SecretType = "env_var"
)

// ValidSecretTypes returns all valid secret types.
func ValidSecretTypes() []string {
	return []string{
		string(SecretTypeSSHKey),
		string(SecretTypeAPIToken),
		string(SecretTypePassword),
		string(SecretTypeCertificate),
		string(SecretTypeEnvVar),
	}
}

// Secret represents an encrypted secret stored in the vault.
type Secret struct {
	ID             uuid.UUID      `json:"id" db:"id"`
	UserID         uuid.UUID      `json:"user_id" db:"user_id"`
	OrgID          uuid.UUID      `json:"org_id,omitempty" db:"org_id"`
	Name           string         `json:"name" db:"name"`
	Type           SecretType     `json:"type" db:"type"`
	EncryptedValue string         `json:"-" db:"encrypted_value"`
	IV             string         `json:"-" db:"iv"`
	Salt           string         `json:"-" db:"salt"`
	Metadata       map[string]any `json:"metadata,omitempty" db:"metadata"`
	Tags           []string       `json:"tags,omitempty" db:"tags"`
	CreatedAt      time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at" db:"updated_at"`
	DeletedAt      *time.Time     `json:"-" db:"deleted_at"`
}

// SecretVersion represents a historical version of a secret.
type SecretVersion struct {
	ID             uuid.UUID `json:"id" db:"id"`
	SecretID       uuid.UUID `json:"secret_id" db:"secret_id"`
	EncryptedValue string    `json:"-" db:"encrypted_value"`
	IV             string    `json:"-" db:"iv"`
	Salt           string    `json:"-" db:"salt"`
	CreatedBy      uuid.UUID `json:"created_by" db:"created_by"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// CreateSecretRequest represents a request to create a new secret.
type CreateSecretRequest struct {
	UserID         uuid.UUID      `json:"user_id" binding:"required"`
	OrgID          uuid.UUID      `json:"org_id,omitempty"`
	Name           string         `json:"name" binding:"required,max=255"`
	Type           string         `json:"type" binding:"required,oneof=ssh_key api_token password certificate env_var"`
	EncryptedValue string         `json:"encrypted_value" binding:"required"`
	IV             string         `json:"iv" binding:"required"`
	Salt           string         `json:"salt" binding:"required"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	Tags           []string       `json:"tags,omitempty"`
}

// UpdateSecretRequest represents a request to update an existing secret.
type UpdateSecretRequest struct {
	Name           string         `json:"name,omitempty" binding:"omitempty,max=255"`
	EncryptedValue string         `json:"encrypted_value,omitempty"`
	IV             string         `json:"iv,omitempty"`
	Salt           string         `json:"salt,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	Tags           []string       `json:"tags,omitempty"`
}

// ListSecretsRequest represents query parameters for listing secrets.
type ListSecretsRequest struct {
	UserID string `form:"user_id" binding:"omitempty,uuid"`
	OrgID  string `form:"org_id" binding:"omitempty,uuid"`
	Type   string `form:"type" binding:"omitempty,oneof=ssh_key api_token password certificate env_var"`
	Tags   string `form:"tags"`
	Limit  int    `form:"limit,default=20" binding:"min=1,max=100"`
	Offset int    `form:"offset,default=0" binding:"min=0"`
}

// SecretResponse represents the response payload for a single secret.
type SecretResponse struct {
	ID        uuid.UUID      `json:"id"`
	UserID    uuid.UUID      `json:"user_id"`
	OrgID     uuid.UUID      `json:"org_id,omitempty"`
	Name      string         `json:"name"`
	Type      string         `json:"type"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	Tags      []string       `json:"tags,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// ListSecretsResponse represents the response payload for listing secrets.
type ListSecretsResponse struct {
	Secrets []*SecretResponse `json:"secrets"`
	Total   int               `json:"total"`
	Limit   int               `json:"limit"`
	Offset  int               `json:"offset"`
}

// ListSecretVersionsResponse represents the response payload for listing secret versions.
type ListSecretVersionsResponse struct {
	Versions []*SecretVersion `json:"versions"`
}

// RotateSecretRequest represents the request payload for rotating a secret.
type RotateSecretRequest struct {
	EncryptedValue string    `json:"encrypted_value" binding:"required"`
	IV             string    `json:"iv" binding:"required"`
	Salt           string    `json:"salt" binding:"required"`
	CreatedBy      uuid.UUID `json:"created_by" binding:"required"`
}

// ToSecretResponse converts a Secret to a SecretResponse (omits encrypted fields).
func ToSecretResponse(s *Secret) *SecretResponse {
	return &SecretResponse{
		ID:        s.ID,
		UserID:    s.UserID,
		OrgID:     s.OrgID,
		Name:      s.Name,
		Type:      string(s.Type),
		Metadata:  s.Metadata,
		Tags:      s.Tags,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}
