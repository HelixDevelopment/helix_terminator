package repository

import (
	"context"
	"errors"
)

// Repository defines the persistence interface.
type Repository interface {
	Ping(ctx context.Context) error
}

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct{}

// NewPostgresRepository creates a new PostgresRepository.
func NewPostgresRepository() *PostgresRepository {
	return &PostgresRepository{}
}

// Ping verifies connectivity.
func (r *PostgresRepository) Ping(ctx context.Context) error {
	return errors.New("not implemented")
}
