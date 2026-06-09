package repository

// Implements DESIGN-013 AuditLogger repository verification.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/security"
)

func TestSecurityAuditRepository(t *testing.T) {
	userID := uuid.New()
	entry := security.AuditLogEntry{RequestID: "request", UserID: &userID, Action: "csrf.validate", Resource: "/fixture", Outcome: "failure", CreatedAt: time.Now()}
	if err := NewPostgresSecurityAuditRepository(&fakeSQLExecutor{row: fakeRow{values: []any{uuid.New()}}}).Audit(context.Background(), entry); err != nil {
		t.Fatal(err)
	}
	attempt := entry
	attempt.Outcome = "attempt"
	if err := NewPostgresSecurityAuditRepository(&fakeSQLExecutor{row: fakeRow{values: []any{uuid.New()}}}).Audit(context.Background(), attempt); err != nil {
		t.Fatalf("Audit(attempt) error = %v", err)
	}
	for _, invalid := range []security.AuditLogEntry{
		{},
		{RequestID: "r", Action: "a", Resource: "x", Outcome: "bad"},
		{RequestID: " r", Action: "a", Resource: "x", Outcome: "failure"},
		{RequestID: "r", Action: "a ", Resource: "x", Outcome: "failure"},
		{RequestID: "r", Action: "a", Resource: " x ", Outcome: "failure"},
	} {
		if err := NewPostgresSecurityAuditRepository(&fakeSQLExecutor{}).Audit(context.Background(), invalid); !IsKind(err, ErrorKindValidation) {
			t.Fatalf("Audit(%+v) = %v", invalid, err)
		}
	}
	if err := NewPostgresSecurityAuditRepository(&fakeSQLExecutor{row: fakeRow{err: errors.New("down")}}).Audit(context.Background(), entry); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("Audit() = %v", err)
	}
}
