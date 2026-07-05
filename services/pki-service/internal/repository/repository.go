package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/helixdevelopment/pki-service/internal/model"
)

// Repository defines the persistence interface for PKI service.
type Repository interface {
	CreateCA(ctx context.Context, ca *model.CertificateAuthority) error
	GetCAByID(ctx context.Context, id uuid.UUID) (*model.CertificateAuthority, error)
	ListCAs(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]*model.CertificateAuthority, error)
	UpdateCA(ctx context.Context, ca *model.CertificateAuthority) error
	DeleteCA(ctx context.Context, id uuid.UUID) error
	CreateCert(ctx context.Context, cert *model.Certificate) error
	GetCertByID(ctx context.Context, id uuid.UUID) (*model.Certificate, error)
	ListCerts(ctx context.Context, caID, orgID uuid.UUID, status string, limit, offset int) ([]*model.Certificate, int, error)
	RevokeCert(ctx context.Context, id uuid.UUID, reason string) error
	GetNextSerialNumber(ctx context.Context, caID uuid.UUID) (int64, error)
	Ping(ctx context.Context) error
}

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgresRepository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) checkPool() error {
	if r.pool == nil {
		return fmt.Errorf("database connection not available")
	}
	return nil
}

// Ping verifies connectivity.
func (r *PostgresRepository) Ping(ctx context.Context) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	return r.pool.Ping(ctx)
}

// CreateCA creates a new certificate authority.
func (r *PostgresRepository) CreateCA(ctx context.Context, ca *model.CertificateAuthority) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := `
		INSERT INTO certificate_authorities (id, org_id, name, description, ca_cert_pem, ca_key_pem, serial_number, validity_days, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.pool.Exec(ctx, query,
		ca.ID, ca.OrgID, ca.Name, ca.Description, ca.CACertPEM, ca.CAKeyPEM, ca.SerialNumber, ca.ValidityDays, ca.CreatedAt, ca.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create CA: %w", err)
	}
	return nil
}

// GetCAByID retrieves a CA by ID.
func (r *PostgresRepository) GetCAByID(ctx context.Context, id uuid.UUID) (*model.CertificateAuthority, error) {
	if err := r.checkPool(); err != nil {
		return nil, err
	}
	query := `
		SELECT id, org_id, name, description, ca_cert_pem, ca_key_pem, serial_number, validity_days, created_at, updated_at, deleted_at
		FROM certificate_authorities
		WHERE id = $1 AND deleted_at IS NULL
	`
	row := r.pool.QueryRow(ctx, query, id)
	ca := &model.CertificateAuthority{}
	err := row.Scan(
		&ca.ID, &ca.OrgID, &ca.Name, &ca.Description, &ca.CACertPEM, &ca.CAKeyPEM,
		&ca.SerialNumber, &ca.ValidityDays, &ca.CreatedAt, &ca.UpdatedAt, &ca.DeletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("CA not found")
		}
		return nil, fmt.Errorf("failed to get CA: %w", err)
	}
	return ca, nil
}

// ListCAs lists certificate authorities for an organization.
func (r *PostgresRepository) ListCAs(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]*model.CertificateAuthority, error) {
	if err := r.checkPool(); err != nil {
		return nil, err
	}
	query := `
		SELECT id, org_id, name, description, ca_cert_pem, ca_key_pem, serial_number, validity_days, created_at, updated_at, deleted_at
		FROM certificate_authorities
		WHERE org_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.pool.Query(ctx, query, orgID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list CAs: %w", err)
	}
	defer rows.Close()

	var cas []*model.CertificateAuthority
	for rows.Next() {
		ca := &model.CertificateAuthority{}
		err := rows.Scan(
			&ca.ID, &ca.OrgID, &ca.Name, &ca.Description, &ca.CACertPEM, &ca.CAKeyPEM,
			&ca.SerialNumber, &ca.ValidityDays, &ca.CreatedAt, &ca.UpdatedAt, &ca.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan CA: %w", err)
		}
		cas = append(cas, ca)
	}
	return cas, nil
}

// UpdateCA updates a certificate authority.
func (r *PostgresRepository) UpdateCA(ctx context.Context, ca *model.CertificateAuthority) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := `
		UPDATE certificate_authorities
		SET name = $2, description = $3, ca_cert_pem = $4, ca_key_pem = $5, serial_number = $6, validity_days = $7, updated_at = $8
		WHERE id = $1 AND deleted_at IS NULL
	`
	_, err := r.pool.Exec(ctx, query,
		ca.ID, ca.Name, ca.Description, ca.CACertPEM, ca.CAKeyPEM, ca.SerialNumber, ca.ValidityDays, time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("failed to update CA: %w", err)
	}
	return nil
}

// DeleteCA soft-deletes a certificate authority.
func (r *PostgresRepository) DeleteCA(ctx context.Context, id uuid.UUID) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := `
		UPDATE certificate_authorities
		SET deleted_at = $2
		WHERE id = $1 AND deleted_at IS NULL
	`
	_, err := r.pool.Exec(ctx, query, id, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to delete CA: %w", err)
	}
	return nil
}

// CreateCert creates a new certificate.
func (r *PostgresRepository) CreateCert(ctx context.Context, cert *model.Certificate) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := `
		INSERT INTO certificates (id, ca_id, org_id, name, cert_pem, key_pem, serial_number, subject, issuer, not_before, not_after, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`
	_, err := r.pool.Exec(ctx, query,
		cert.ID, cert.CAID, cert.OrgID, cert.Name, cert.CertPEM, cert.KeyPEM, cert.SerialNumber,
		cert.Subject, cert.Issuer, cert.NotBefore, cert.NotAfter, cert.Status, cert.CreatedAt, cert.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %w", err)
	}
	return nil
}

// GetCertByID retrieves a certificate by ID.
func (r *PostgresRepository) GetCertByID(ctx context.Context, id uuid.UUID) (*model.Certificate, error) {
	if err := r.checkPool(); err != nil {
		return nil, err
	}
	query := `
		SELECT id, ca_id, org_id, name, cert_pem, key_pem, serial_number, subject, issuer, not_before, not_after, revoked_at, revocation_reason, status, created_at, updated_at
		FROM certificates
		WHERE id = $1
	`
	row := r.pool.QueryRow(ctx, query, id)
	cert := &model.Certificate{}
	err := row.Scan(
		&cert.ID, &cert.CAID, &cert.OrgID, &cert.Name, &cert.CertPEM, &cert.KeyPEM,
		&cert.SerialNumber, &cert.Subject, &cert.Issuer, &cert.NotBefore, &cert.NotAfter,
		&cert.RevokedAt, &cert.RevocationReason, &cert.Status, &cert.CreatedAt, &cert.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("certificate not found")
		}
		return nil, fmt.Errorf("failed to get certificate: %w", err)
	}
	return cert, nil
}

// ListCerts lists certificates with optional filters.
func (r *PostgresRepository) ListCerts(ctx context.Context, caID, orgID uuid.UUID, status string, limit, offset int) ([]*model.Certificate, int, error) {
	if err := r.checkPool(); err != nil {
		return nil, 0, err
	}

	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if caID != uuid.Nil {
		whereClause += fmt.Sprintf(" AND ca_id = $%d", argIdx)
		args = append(args, caID)
		argIdx++
	}
	if orgID != uuid.Nil {
		whereClause += fmt.Sprintf(" AND org_id = $%d", argIdx)
		args = append(args, orgID)
		argIdx++
	}
	if status != "" {
		whereClause += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM certificates %s", whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count certificates: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT id, ca_id, org_id, name, cert_pem, key_pem, serial_number, subject, issuer, not_before, not_after, revoked_at, revocation_reason, status, created_at, updated_at
		FROM certificates
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list certificates: %w", err)
	}
	defer rows.Close()

	var certs []*model.Certificate
	for rows.Next() {
		cert := &model.Certificate{}
		err := rows.Scan(
			&cert.ID, &cert.CAID, &cert.OrgID, &cert.Name, &cert.CertPEM, &cert.KeyPEM,
			&cert.SerialNumber, &cert.Subject, &cert.Issuer, &cert.NotBefore, &cert.NotAfter,
			&cert.RevokedAt, &cert.RevocationReason, &cert.Status, &cert.CreatedAt, &cert.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan certificate: %w", err)
		}
		certs = append(certs, cert)
	}
	return certs, total, nil
}

// RevokeCert revokes a certificate.
func (r *PostgresRepository) RevokeCert(ctx context.Context, id uuid.UUID, reason string) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := `
		UPDATE certificates
		SET status = 'revoked', revoked_at = $2, revocation_reason = $3, updated_at = $4
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query, id, time.Now().UTC(), reason, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to revoke certificate: %w", err)
	}
	return nil
}

// GetNextSerialNumber increments and returns the next serial number for a CA.
func (r *PostgresRepository) GetNextSerialNumber(ctx context.Context, caID uuid.UUID) (int64, error) {
	if err := r.checkPool(); err != nil {
		return 0, err
	}
	query := `
		UPDATE certificate_authorities
		SET serial_number = serial_number + 1, updated_at = $2
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING serial_number
	`
	var serial int64
	err := r.pool.QueryRow(ctx, query, caID, time.Now().UTC()).Scan(&serial)
	if err != nil {
		return 0, fmt.Errorf("failed to get next serial number: %w", err)
	}
	return serial, nil
}
