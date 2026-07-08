package model

import (
	"time"

	"github.com/google/uuid"
)

// KeychainItem represents a stored credential or key in the keychain
type KeychainItem struct {
	ID          uuid.UUID              `json:"id" db:"id"`
	UserID      uuid.UUID              `json:"userId" db:"user_id"`
	OrgID       *uuid.UUID             `json:"orgId,omitempty" db:"org_id"`
	Name        string                 `json:"name" db:"name"`
	Type        KeyType                `json:"type" db:"type"`
	Fingerprint string                 `json:"fingerprint" db:"fingerprint"`
	PublicKey   string                 `json:"publicKey,omitempty" db:"public_key"`
	PrivateKey  string                 `json:"-" db:"private_key"` // never serialized
	Passphrase  string                 `json:"-" db:"passphrase"`  // never serialized
	Metadata    map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	Tags        []string               `json:"tags,omitempty" db:"tags"`
	CreatedAt   time.Time              `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time              `json:"updatedAt" db:"updated_at"`
	DeletedAt   *time.Time             `json:"deletedAt,omitempty" db:"deleted_at"`
}

// KeyType represents the type of keychain item
type KeyType string

const (
	KeyTypeSSH      KeyType = "ssh"
	KeyTypeGPG      KeyType = "gpg"
	KeyTypeAPIKey   KeyType = "api_key"
	KeyTypePassword KeyType = "password"
	KeyTypeX509     KeyType = "x509"
)

// ValidKeyTypes returns all valid key types
func ValidKeyTypes() []string {
	return []string{string(KeyTypeSSH), string(KeyTypeGPG), string(KeyTypeAPIKey), string(KeyTypePassword), string(KeyTypeX509)}
}

// CreateKeychainItemRequest represents a request to create a keychain item
type CreateKeychainItemRequest struct {
	Name       string                 `json:"name" binding:"required,max=255"`
	Type       string                 `json:"type" binding:"required,oneof=ssh gpg api_key password x509"`
	PublicKey  string                 `json:"publicKey,omitempty"`
	PrivateKey string                 `json:"privateKey" binding:"required"`
	Passphrase string                 `json:"passphrase,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Tags       []string               `json:"tags,omitempty"`
}

// UpdateKeychainItemRequest represents a request to update a keychain item
type UpdateKeychainItemRequest struct {
	Name      *string                `json:"name,omitempty"`
	PublicKey *string                `json:"publicKey,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Tags      []string               `json:"tags,omitempty"`
}

// ListKeychainItemsRequest represents a request to list keychain items
type ListKeychainItemsRequest struct {
	UserID string `form:"userId"`
	OrgID  string `form:"orgId"`
	Type   string `form:"type"`
	Limit  int    `form:"limit,default=20"`
	Offset int    `form:"offset,default=0"`
}

// KeychainItemResponse is the API response for a keychain item
type KeychainItemResponse struct {
	ID          uuid.UUID              `json:"id"`
	UserID      uuid.UUID              `json:"userId"`
	OrgID       *uuid.UUID             `json:"orgId,omitempty"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Fingerprint string                 `json:"fingerprint"`
	PublicKey   string                 `json:"publicKey,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	CreatedAt   time.Time              `json:"createdAt"`
}

// ListKeychainItemsResponse is the API response for listing keychain items
type ListKeychainItemsResponse struct {
	Items  []*KeychainItemResponse `json:"items"`
	Total  int                     `json:"total"`
	Limit  int                     `json:"limit"`
	Offset int                     `json:"offset"`
}
