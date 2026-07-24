package repository

// Implements DESIGN-009 AdminController audit snapshot privacy and rollback verification.

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestAdminAuditSnapshotsRejectUnsafeOrUnboundedData(t *testing.T) {
	base := AdminAuditEntry{AdminUserID: uuid.New(), Action: "fixture.update", EntityType: "fixture", RequestID: uuid.NewString()}
	tests := []struct {
		name     string
		snapshot []byte
	}{
		{name: "malformed", snapshot: []byte(`{`)},
		{name: "pii", snapshot: []byte(`{"email":"admin@example.test"}`)},
		{name: "secret", snapshot: []byte(`{"accessToken":"secret"}`)},
		{name: "provider payload", snapshot: []byte(`{"rawProviderPayload":{"name":"private"}}`)},
		{name: "unknown field", snapshot: []byte(`{"arbitrary":"value"}`)},
		{name: "name free text", snapshot: []byte(`{"name":"alice@example.test"}`)},
		{name: "status free text", snapshot: []byte(`{"status":"provider api key=secret"}`)},
		{name: "status wrong type", snapshot: []byte(`{"status":1}`)},
		{name: "state free text", snapshot: []byte(`{"state":"alice@example.test"}`)},
		{name: "kind free text", snapshot: []byte(`{"kind":"raw provider payload"}`)},
		{name: "physical state free text", snapshot: []byte(`{"physicalState":"access-token"}`)},
		{name: "reason free text", snapshot: []byte(`{"reason":"provider api key=secret"}`)},
		{name: "version free text", snapshot: []byte(`{"version":"alice@example.test"}`)},
		{name: "parent id free text", snapshot: []byte(`{"parentId":"secret"}`)},
		{name: "replacement id free text", snapshot: []byte(`{"replacementId":"secret"}`)},
		{name: "active wrong type", snapshot: []byte(`{"active":"secret"}`)},
		{name: "deleted wrong type", snapshot: []byte(`{"deleted":"secret"}`)},
		{name: "oversized", snapshot: []byte(`{"reason":"` + strings.Repeat("x", 5000) + `"}`)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			entry := base
			entry.After = test.snapshot
			if err := validateAdminAuditEntry(entry); !IsKind(err, ErrorKindValidation) {
				t.Fatalf("validateAdminAuditEntry() error = %v, want validation", err)
			}
		})
	}

	valid := base
	valid.Before = []byte(`{"status":"draft","active":true,"deleted":false}`)
	valid.After = []byte(`{"status":"published","active":true,"deleted":false}`)
	if err := validateAdminAuditEntry(valid); err != nil {
		t.Fatalf("safe metadata rejected: %v", err)
	}
	canonical, err := sanitizeAdminAuditSnapshot("fixture", "fixture.update", []byte(`{"status":"provider api key=secret","status":"published"}`))
	if err != nil || string(canonical) != `{"status":"published"}` || strings.Contains(string(canonical), "secret") {
		t.Fatalf("duplicate-field sanitization = %s, err=%v", canonical, err)
	}
	food := base
	food.Action = "update_food"
	food.EntityType = "food_item"
	food.After = []byte(`{"status":"imported"}`)
	if err := validateAdminAuditEntry(food); err != nil {
		t.Fatalf("safe food metadata rejected: %v", err)
	}
	wrongAction := base
	wrongAction.Action = "fixture.delete"
	wrongAction.After = []byte(`{"status":"published"}`)
	if err := validateAdminAuditEntry(wrongAction); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("unknown snapshot schema error = %v, want validation", err)
	}
	badBefore := base
	badBefore.Before = []byte(`{"status":"secret"}`)
	if err := validateAdminAuditEntry(badBefore); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("unsafe before snapshot error = %v, want validation", err)
	}
	classification := base
	classification.EntityType = "classification"
	classification.Action = "classification.update"
	classification.Before = []byte(`{"active":true,"deleted":false,"kind":"food_category","nameDigest":"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef","parentId":"` + uuid.NewString() + `"}`)
	classification.After = []byte(`{"active":true,"deleted":false,"kind":"food_category","nameDigest":"abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"}`)
	if err := validateAdminAuditEntry(classification); err != nil {
		t.Fatalf("safe classification metadata rejected: %v", err)
	}
	classification.After = []byte(`{"active":true,"deleted":false,"kind":"food_category","nameDigest":"not-a-digest","parentId":"not-a-uuid"}`)
	if err := validateAdminAuditEntry(classification); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("invalid classification metadata error = %v", err)
	}
}

func TestAdminAuditSnapshotValidationRollsBackTransaction(t *testing.T) {
	tx := &fakeTx{}
	repo := NewPostgresAdminImportAuditRepository(&fakeSQLExecutor{tx: tx})
	mutationCalled := false
	err := repo.WithMutationAudit(context.Background(), AdminAuditEntry{
		AdminUserID: uuid.New(), Action: "fixture.update", EntityType: "fixture", RequestID: uuid.NewString(),
	}, func(AdminMutationExecutor) (AdminAuditChanges, error) {
		mutationCalled = true
		return AdminAuditChanges{After: []byte(`{"password":"must-not-persist"}`)}, nil
	})
	if !mutationCalled || !IsKind(err, ErrorKindValidation) || !tx.rolledBack {
		t.Fatalf("unsafe snapshot error=%v mutation=%t rollback=%t", err, mutationCalled, tx.rolledBack)
	}
}

func TestAdminAuditPersistenceErrorPreservesCause(t *testing.T) {
	cause := errors.New("audit insert unavailable")
	tx := &fakeTx{fakeSQLExecutor: fakeSQLExecutor{row: fakeRow{err: cause}}}
	repo := NewPostgresAdminImportAuditRepository(&fakeSQLExecutor{tx: tx})
	err := repo.WithMutationAudit(context.Background(), AdminAuditEntry{
		AdminUserID: uuid.New(), Action: "fixture.update", EntityType: "fixture", RequestID: uuid.NewString(),
	}, func(AdminMutationExecutor) (AdminAuditChanges, error) {
		return AdminAuditChanges{After: []byte(`{"status":"published"}`)}, nil
	})
	if !errors.Is(err, ErrAdminAuditPersistence) || !errors.Is(err, cause) || !tx.rolledBack {
		t.Fatalf("audit error = %v, sentinel=%t cause=%t rollback=%t", err, errors.Is(err, ErrAdminAuditPersistence), errors.Is(err, cause), tx.rolledBack)
	}
}

func TestAdminMutationAuditSuccessfulCommitPath(t *testing.T) {
	tx := &fakeTx{fakeSQLExecutor: fakeSQLExecutor{row: fakeRow{values: []any{uuid.New()}}}}
	repo := NewPostgresAdminImportAuditRepository(&fakeSQLExecutor{tx: tx})
	err := repo.WithMutationAudit(context.Background(), AdminAuditEntry{
		AdminUserID: uuid.New(), Action: "fixture.update", EntityType: "fixture", RequestID: uuid.NewString(),
	}, func(AdminMutationExecutor) (AdminAuditChanges, error) {
		return AdminAuditChanges{After: []byte(`{"status":"published","active":true}`)}, nil
	})
	if err != nil {
		t.Fatalf("WithMutationAudit() success error = %v", err)
	}
}

func TestAdminMutationAuditReplayCommitsWithoutDuplicateAudit(t *testing.T) {
	tx := &fakeTx{}
	repo := NewPostgresAdminImportAuditRepository(&fakeSQLExecutor{tx: tx})
	entry := AdminAuditEntry{AdminUserID: uuid.New(), Action: "manual_create", EntityType: "food_item", RequestID: uuid.NewString()}
	if err := repo.WithMutationAudit(context.Background(), entry, func(AdminMutationExecutor) (AdminAuditChanges, error) {
		return AdminAuditChanges{Replayed: true}, nil
	}); err != nil || tx.rowN != 0 {
		t.Fatalf("replay error=%v audit inserts=%d", err, tx.rowN)
	}
	tx = &fakeTx{}
	repo = NewPostgresAdminImportAuditRepository(&fakeSQLExecutor{tx: tx})
	if err := repo.WithMutationAudit(context.Background(), entry, func(AdminMutationExecutor) (AdminAuditChanges, error) {
		id := uuid.New()
		return AdminAuditChanges{Replayed: true, EntityID: &id}, nil
	}); !IsKind(err, ErrorKindValidation) || !tx.rolledBack {
		t.Fatalf("unsafe replay error=%v rollback=%t", err, tx.rolledBack)
	}
}
