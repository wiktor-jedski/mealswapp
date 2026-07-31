package repository

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

// TestPostgresMicronutrientMutationAuditCommitsAndRollsBack proves the real
// vocabulary transaction shares the compliance audit boundary.
// Implements DESIGN-005 MicronutrientVocabulary and DESIGN-009 AdminController.
func TestPostgresMicronutrientMutationAuditCommitsAndRollsBack(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	audit := NewPostgresAdminImportAuditRepository(db)
	key := "Audit" + uuid.NewString()[:8]
	adminID := createRepositoryUser(t, ctx, db, "micronutrient-audit-"+uuid.NewString()+"@example.test")
	entry := AdminAuditEntry{ActorKind: AdminAuditActorAdministrator, AdminUserID: &adminID, Action: "micronutrient.create", EntityType: "micronutrient_vocabulary", RequestID: uuid.NewString()}
	err := audit.WithMutationAudit(ctx, entry, func(tx AdminMutationExecutor) (AdminAuditChanges, error) {
		created, err := NewPostgresMicronutrientVocabularyRepository(tx).Create(ctx, MicronutrientVocabularyEntry{Key: key, DisplayName: "Audit test", Unit: "mg"})
		if err != nil {
			return AdminAuditChanges{}, err
		}
		after, err := json.Marshal(map[string]any{"active": created.Active, "keyDigest": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "unit": created.Unit})
		return AdminAuditChanges{After: after}, err
	})
	if err != nil {
		t.Fatalf("successful vocabulary audit mutation: %v", err)
	}
	if _, err := NewPostgresMicronutrientVocabularyRepository(db).Get(ctx, key); err != nil {
		t.Fatalf("committed vocabulary mutation missing: %v", err)
	}

	rollbackKey := "Audit" + uuid.NewString()[:8]
	err = audit.WithMutationAudit(ctx, entry, func(tx AdminMutationExecutor) (AdminAuditChanges, error) {
		if _, err := NewPostgresMicronutrientVocabularyRepository(tx).Create(ctx, MicronutrientVocabularyEntry{Key: rollbackKey, DisplayName: "Rollback test", Unit: "mg"}); err != nil {
			return AdminAuditChanges{}, err
		}
		return AdminAuditChanges{After: []byte(`{"forbidden":"field"}`)}, nil
	})
	if err == nil {
		t.Fatal("invalid audit snapshot unexpectedly committed")
	}
	if _, err := NewPostgresMicronutrientVocabularyRepository(db).Get(ctx, rollbackKey); !IsKind(err, ErrorKindNotFound) {
		t.Fatalf("audit failure left vocabulary mutation: %v", err)
	}
}
