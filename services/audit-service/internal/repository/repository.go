package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/helixdevelopment/audit-service/internal/model"
)

// Repository handles database operations for audit service
type Repository struct {
	pool *pgxpool.Pool
}

// New creates a new repository
func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) checkPool() error {
	if r.pool == nil {
		return fmt.Errorf("database not connected")
	}
	return nil
}

// CreateAuditLog creates a new audit log entry
func (r *Repository) CreateAuditLog(ctx context.Context, log *model.AuditLog) error {
	if err := r.checkPool(); err != nil {
		return err
	}

	query := `
		INSERT INTO audit_logs (id, org_id, user_id, action, resource_type, resource_id, details, ip_address, user_agent, timestamp, severity)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := r.pool.Exec(ctx, query,
		log.ID, log.OrgID, log.UserID, log.Action, log.ResourceType, log.ResourceID,
		log.Details, log.IPAddress, log.UserAgent, log.Timestamp, log.Severity,
	)
	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}
	return nil
}

// GetAuditLogByID retrieves an audit log by ID
func (r *Repository) GetAuditLogByID(ctx context.Context, id uuid.UUID) (*model.AuditLog, error) {
	if err := r.checkPool(); err != nil {
		return nil, err
	}

	query := `
		SELECT id, org_id, user_id, action, resource_type, resource_id, details, ip_address, user_agent, timestamp, severity
		FROM audit_logs
		WHERE id = $1
	`
	row := r.pool.QueryRow(ctx, query, id)

	log := &model.AuditLog{}
	err := row.Scan(
		&log.ID, &log.OrgID, &log.UserID, &log.Action, &log.ResourceType, &log.ResourceID,
		&log.Details, &log.IPAddress, &log.UserAgent, &log.Timestamp, &log.Severity,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("audit log not found")
		}
		return nil, fmt.Errorf("failed to get audit log: %w", err)
	}
	return log, nil
}

// ListAuditLogs retrieves audit logs with optional filtering
func (r *Repository) ListAuditLogs(ctx context.Context, orgID, userID *uuid.UUID, action model.AuditAction, resourceType model.AuditResourceType, severity model.AuditSeverity, startTime, endTime *time.Time, limit, offset int) ([]*model.AuditLog, int, error) {
	if err := r.checkPool(); err != nil {
		return nil, 0, err
	}

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	// Build dynamic WHERE clause
	whereParts := []string{"1=1"}
	args := []interface{}{}
	argIdx := 1

	if orgID != nil {
		whereParts = append(whereParts, fmt.Sprintf("org_id = $%d", argIdx))
		args = append(args, orgID)
		argIdx++
	}
	if userID != nil {
		whereParts = append(whereParts, fmt.Sprintf("user_id = $%d", argIdx))
		args = append(args, userID)
		argIdx++
	}
	if action != "" {
		whereParts = append(whereParts, fmt.Sprintf("action = $%d", argIdx))
		args = append(args, action)
		argIdx++
	}
	if resourceType != "" {
		whereParts = append(whereParts, fmt.Sprintf("resource_type = $%d", argIdx))
		args = append(args, resourceType)
		argIdx++
	}
	if severity != "" {
		whereParts = append(whereParts, fmt.Sprintf("severity = $%d", argIdx))
		args = append(args, severity)
		argIdx++
	}
	if startTime != nil {
		whereParts = append(whereParts, fmt.Sprintf("timestamp >= $%d", argIdx))
		args = append(args, *startTime)
		argIdx++
	}
	if endTime != nil {
		whereParts = append(whereParts, fmt.Sprintf("timestamp <= $%d", argIdx))
		args = append(args, *endTime)
		argIdx++
	}

	whereClause := ""
	for i, part := range whereParts {
		if i == 0 {
			whereClause = "WHERE " + part
		} else {
			whereClause += " AND " + part
		}
	}

	// Count total
	countQuery := "SELECT COUNT(*) FROM audit_logs " + whereClause
	var total int
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	// Fetch logs
	query := fmt.Sprintf(`
		SELECT id, org_id, user_id, action, resource_type, resource_id, details, ip_address, user_agent, timestamp, severity
		FROM audit_logs
		%s
		ORDER BY timestamp DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list audit logs: %w", err)
	}
	defer rows.Close()

	var logs []*model.AuditLog
	for rows.Next() {
		log := &model.AuditLog{}
		err := rows.Scan(
			&log.ID, &log.OrgID, &log.UserID, &log.Action, &log.ResourceType, &log.ResourceID,
			&log.Details, &log.IPAddress, &log.UserAgent, &log.Timestamp, &log.Severity,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan audit log: %w", err)
		}
		logs = append(logs, log)
	}

	return logs, total, nil
}

// CountByAction returns counts of audit logs grouped by action
func (r *Repository) CountByAction(ctx context.Context, orgID *uuid.UUID, startTime, endTime *time.Time) (map[string]int, error) {
	if err := r.checkPool(); err != nil {
		return nil, err
	}

	whereParts := []string{"1=1"}
	args := []interface{}{}
	argIdx := 1

	if orgID != nil {
		whereParts = append(whereParts, fmt.Sprintf("org_id = $%d", argIdx))
		args = append(args, orgID)
		argIdx++
	}
	if startTime != nil {
		whereParts = append(whereParts, fmt.Sprintf("timestamp >= $%d", argIdx))
		args = append(args, *startTime)
		argIdx++
	}
	if endTime != nil {
		whereParts = append(whereParts, fmt.Sprintf("timestamp <= $%d", argIdx))
		args = append(args, *endTime)
		argIdx++
	}

	whereClause := ""
	for i, part := range whereParts {
		if i == 0 {
			whereClause = "WHERE " + part
		} else {
			whereClause += " AND " + part
		}
	}

	query := fmt.Sprintf("SELECT action, COUNT(*) FROM audit_logs %s GROUP BY action", whereClause)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to count by action: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var action string
		var count int
		if err := rows.Scan(&action, &count); err != nil {
			return nil, fmt.Errorf("failed to scan count: %w", err)
		}
		counts[action] = count
	}

	return counts, nil
}

// CountByResourceType returns counts of audit logs grouped by resource type
func (r *Repository) CountByResourceType(ctx context.Context, orgID *uuid.UUID, startTime, endTime *time.Time) (map[string]int, error) {
	if err := r.checkPool(); err != nil {
		return nil, err
	}

	whereParts := []string{"1=1"}
	args := []interface{}{}
	argIdx := 1

	if orgID != nil {
		whereParts = append(whereParts, fmt.Sprintf("org_id = $%d", argIdx))
		args = append(args, orgID)
		argIdx++
	}
	if startTime != nil {
		whereParts = append(whereParts, fmt.Sprintf("timestamp >= $%d", argIdx))
		args = append(args, *startTime)
		argIdx++
	}
	if endTime != nil {
		whereParts = append(whereParts, fmt.Sprintf("timestamp <= $%d", argIdx))
		args = append(args, *endTime)
		argIdx++
	}

	whereClause := ""
	for i, part := range whereParts {
		if i == 0 {
			whereClause = "WHERE " + part
		} else {
			whereClause += " AND " + part
		}
	}

	query := fmt.Sprintf("SELECT resource_type, COUNT(*) FROM audit_logs %s GROUP BY resource_type", whereClause)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to count by resource type: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var resourceType string
		var count int
		if err := rows.Scan(&resourceType, &count); err != nil {
			return nil, fmt.Errorf("failed to scan count: %w", err)
		}
		counts[resourceType] = count
	}

	return counts, nil
}

// Ping checks database connectivity
func (r *Repository) Ping(ctx context.Context) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	return r.pool.Ping(ctx)
}
