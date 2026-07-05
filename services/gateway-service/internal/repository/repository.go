package repository

import (
	"context"
	"errors"
)

// Repository defines the persistence interface.
// TODO: add methods for domain entities.
type Repository interface {
	Ping(ctx context.Context) error
}

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	// TODO: inject *sql.DB or *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgresRepository.
func NewPostgresRepository() *PostgresRepository {
	return &PostgresRepository{}
}

// Ping verifies connectivity.
func (r *PostgresRepository) Ping(ctx context.Context) error {
	// TODO: implement real ping
	return errors.New("not implemented")
}
