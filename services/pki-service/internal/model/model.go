package model

import (
	"time"

	"github.com/google/uuid"
)

// CertificateStatus represents the status of a certificate.
type CertificateStatus string

const (
	StatusActive  CertificateStatus = "active"
	StatusExpired CertificateStatus = "expired"
	StatusRevoked CertificateStatus = "revoked"
)

// CertificateAuthority represents a CA in the system.
type CertificateAuthority struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	OrgID         uuid.UUID  `json:"org_id" db:"org_id"`
	Name          string     `json:"name" db:"name"`
	Description   string     `json:"description,omitempty" db:"description"`
	CACertPEM     string     `json:"ca_cert_pem,omitempty" db:"ca_cert_pem"`
	CAKeyPEM      string     `json:"-" db:"ca_key_pem"`
	SerialNumber  int64      `json:"serial_number" db:"serial_number"`
	ValidityDays  int        `json:"validity_days" db:"validity_days"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt     *time.Time `json:"-" db:"deleted_at"`
}

// Certificate represents an issued certificate.
type Certificate struct {
	ID               uuid.UUID         `json:"id" db:"id"`
	CAID             uuid.UUID         `json:"ca_id" db:"ca_id"`
	OrgID            uuid.UUID         `json:"org_id" db:"org_id"`
	Name             string            `json:"name" db:"name"`
	CertPEM          string            `json:"cert_pem,omitempty" db:"cert_pem"`
	KeyPEM           string            `json:"-" db:"key_pem"`
	SerialNumber     int64             `json:"serial_number" db:"serial_number"`
	Subject          []byte            `json:"subject,omitempty" db:"subject"`
	Issuer           []byte            `json:"issuer,omitempty" db:"issuer"`
	NotBefore        time.Time         `json:"not_before" db:"not_before"`
	NotAfter         time.Time         `json:"not_after" db:"not_after"`
	RevokedAt        *time.Time        `json:"revoked_at,omitempty" db:"revoked_at"`
	RevocationReason string            `json:"revocation_reason,omitempty" db:"revocation_reason"`
	Status           CertificateStatus `json:"status" db:"status"`
	CreatedAt        time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at" db:"updated_at"`
}

// CreateCARequest represents a request to create a CA.
type CreateCARequest struct {
	OrgID        string `json:"org_id" binding:"required,uuid"`
	Name         string `json:"name" binding:"required,max=255"`
	Description  string `json:"description,omitempty" binding:"omitempty,max=1000"`
	ValidityDays int    `json:"validity_days" binding:"required,min=1,max=36500"`
}

// CreateCertRequest represents a request to create a certificate.
type CreateCertRequest struct {
	Name         string `json:"name" binding:"required,max=255"`
	Subject      string `json:"subject" binding:"required"`
	ValidityDays int    `json:"validity_days" binding:"required,min=1,max=36500"`
}

// ListCertsRequest represents query parameters for listing certificates.
type ListCertsRequest struct {
	CAID   string `form:"ca_id,omitempty" binding:"omitempty,uuid"`
	OrgID  string `form:"org_id,omitempty" binding:"omitempty,uuid"`
	Status string `form:"status,omitempty" binding:"omitempty,oneof=active expired revoked"`
	Limit  int    `form:"limit,default=20" binding:"min=1,max=100"`
	Offset int    `form:"offset,default=0" binding:"min=0"`
}

// CAResponse represents a CA response.
type CAResponse struct {
	ID           uuid.UUID `json:"id"`
	OrgID        uuid.UUID `json:"org_id"`
	Name         string    `json:"name"`
	Description  string    `json:"description,omitempty"`
	CACertPEM    string    `json:"ca_cert_pem,omitempty"`
	SerialNumber int64     `json:"serial_number"`
	ValidityDays int       `json:"validity_days"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// CertResponse represents a certificate response.
type CertResponse struct {
	ID               uuid.UUID         `json:"id"`
	CAID             uuid.UUID         `json:"ca_id"`
	OrgID            uuid.UUID         `json:"org_id"`
	Name             string            `json:"name"`
	CertPEM          string            `json:"cert_pem,omitempty"`
	SerialNumber     int64             `json:"serial_number"`
	Subject          []byte            `json:"subject,omitempty"`
	Issuer           []byte            `json:"issuer,omitempty"`
	NotBefore        time.Time         `json:"not_before"`
	NotAfter         time.Time         `json:"not_after"`
	RevokedAt        *time.Time        `json:"revoked_at,omitempty"`
	RevocationReason string            `json:"revocation_reason,omitempty"`
	Status           CertificateStatus `json:"status"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

// ListCertsResponse represents a list of certificates response.
type ListCertsResponse struct {
	Certificates []*CertResponse `json:"certificates"`
	Total        int             `json:"total"`
	Limit        int             `json:"limit"`
	Offset       int             `json:"offset"`
}
