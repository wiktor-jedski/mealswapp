package repository

import (
	"context"
	_ "embed"
	"strings"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/security"
)

// Implements DESIGN-013 AuditLogger insert query.
//
//go:embed sql/security_audit_insert.sql
var securityAuditInsertSQL string

// PostgresSecurityAuditRepository persists security events.
// Implements DESIGN-013 AuditLogger.
type PostgresSecurityAuditRepository struct {
	db sqlExecutor
}

// Implements DESIGN-013 AuditLogger compile-time repository contract.
var _ security.AuditLogger = (*PostgresSecurityAuditRepository)(nil)

// NewPostgresSecurityAuditRepository creates a PostgreSQL-backed security audit logger.
// Implements DESIGN-013 AuditLogger.
func NewPostgresSecurityAuditRepository(db sqlExecutor) *PostgresSecurityAuditRepository {
	return &PostgresSecurityAuditRepository{db: db}
}

// Audit persists request-correlated security metadata.
// Implements DESIGN-013 AuditLogger.
func (r *PostgresSecurityAuditRepository) Audit(ctx context.Context, entry security.AuditLogEntry) error {
	if strings.TrimSpace(entry.RequestID) == "" || strings.TrimSpace(entry.Action) == "" ||
		strings.TrimSpace(entry.Resource) == "" || (entry.Outcome != "attempt" && entry.Outcome != "success" && entry.Outcome != "failure") {
		return validationError("security audit metadata is invalid")
	}
	if entry.RequestID != strings.TrimSpace(entry.RequestID) || entry.Action != strings.TrimSpace(entry.Action) ||
		entry.Resource != strings.TrimSpace(entry.Resource) {
		return validationError("security audit metadata must not contain surrounding whitespace")
	}
	var id uuid.UUID
	if err := r.db.QueryRow(ctx, securityAuditInsertSQL, entry.RequestID, entry.UserID, entry.Action, entry.Resource, entry.Outcome, entry.IP, entry.UserAgent, entry.CreatedAt).Scan(&id); err != nil {
		return mapPostgresError(err, "persist security audit")
	}
	return nil
}
