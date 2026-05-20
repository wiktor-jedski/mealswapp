package externaldata

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"mealswapp/backend/internal/repositories"

	"github.com/google/uuid"
)

func TestAuditLoggerPersistsStructuredRedactedEntry(t *testing.T) {
	store := &fakeAuditStore{id: uuid.New()}
	logger := NewAuditLogger(store)
	logger.now = func() time.Time { return time.Date(2026, 5, 20, 12, 30, 0, 0, time.UTC) }
	actorID := uuid.New()

	view, err := logger.Log(context.Background(), AuditEvent{
		ActorID:   &actorID,
		Action:    "admin.import_item",
		Target:    "food_item:" + uuid.NewString(),
		RequestID: "req-123",
		Metadata: map[string]any{
			"provider":     "usda",
			"passwordHash": "should-not-persist",
			"nested": map[string]any{
				"refreshToken": "secret",
				"ok":           true,
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected audit error: %v", err)
	}

	if view.ID != store.id || view.ActorID == nil || *view.ActorID != actorID || view.Action != "admin.import_item" || view.RequestID != "req-123" {
		t.Fatalf("unexpected audit view: %#v", view)
	}
	if store.entry.Action != "admin.import_item" || store.entry.Target == "" {
		t.Fatalf("unexpected persisted entry: %#v", store.entry)
	}
	var metadata map[string]any
	if err := json.Unmarshal(store.entry.Metadata, &metadata); err != nil {
		t.Fatal(err)
	}
	if metadata["passwordHash"] != "[redacted]" || metadata["requestId"] != "req-123" {
		t.Fatalf("expected redacted metadata with request id, got %#v", metadata)
	}
	nested := metadata["nested"].(map[string]any)
	if nested["refreshToken"] != "[redacted]" || nested["ok"] != true {
		t.Fatalf("expected nested redaction, got %#v", nested)
	}
}

func TestAuditLoggerSupportsRequiredActionFamilies(t *testing.T) {
	store := &fakeAuditStore{id: uuid.New()}
	logger := NewAuditLogger(store)
	actions := []string{
		"admin.import_item",
		"admin.update_item",
		"admin.update_tag",
		"admin.disable_user",
		"auth.password_reset",
		"subscription.reconciliation",
		"account.deletion",
	}
	for _, action := range actions {
		if _, err := logger.Log(context.Background(), AuditEvent{Action: action, Target: "user:" + uuid.NewString(), RequestID: "req"}); err != nil {
			t.Fatalf("expected %s audit action to persist: %v", action, err)
		}
	}
	if len(store.entries) != len(actions) {
		t.Fatalf("expected %d audit entries, got %d", len(actions), len(store.entries))
	}
}

type fakeAuditStore struct {
	id      uuid.UUID
	entry   repositories.AuditLogEntity
	entries []repositories.AuditLogEntity
}

func (store *fakeAuditStore) Create(ctx context.Context, entry repositories.AuditLogEntity) (uuid.UUID, error) {
	if store.id == uuid.Nil {
		store.id = uuid.New()
	}
	store.entry = entry
	store.entries = append(store.entries, entry)
	return store.id, nil
}
