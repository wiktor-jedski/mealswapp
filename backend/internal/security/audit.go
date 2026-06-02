package security

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// AuditLogEntry describes request-correlated security metadata without PII.
// Implements DESIGN-013 AuditLogger.
type AuditLogEntry struct {
	RequestID string
	UserID    *uuid.UUID
	Action    string
	Resource  string
	Outcome   string
	IP        string
	UserAgent string
	CreatedAt time.Time
}

// AuditLogger persists security audit events.
// Implements DESIGN-013 AuditLogger.
type AuditLogger interface {
	Audit(context.Context, AuditLogEntry) error
}

// RecordAuditRequired records an event and fails closed when persistence is unavailable.
// Implements DESIGN-013 AuditLogger.
func RecordAuditRequired(ctx context.Context, logger AuditLogger, entry AuditLogEntry) error {
	if logger == nil {
		return errors.New("audit logger is required for security-sensitive mutations")
	}
	return logger.Audit(ctx, entry)
}

// RecordAuditBestEffort records an event without failing the request during audit outages.
// Implements DESIGN-013 AuditLogger.
func RecordAuditBestEffort(ctx context.Context, logger AuditLogger, entry AuditLogEntry) {
	if logger != nil {
		_ = logger.Audit(ctx, entry)
	}
}
