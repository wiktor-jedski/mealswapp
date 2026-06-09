package userdata

// Implements DESIGN-008 DataExporter verification.

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/security"
)

type memoryExportRepository struct {
	user    repository.EncryptedAuthUser
	profile repository.EncryptedUserProfile
	saved   []repository.SavedItem
	history []repository.EncryptedSearchHistoryEntry
	consent []repository.ConsentRecord
}

func (r *memoryExportRepository) GetEncryptedUserByID(_ context.Context, userID uuid.UUID) (repository.EncryptedAuthUser, error) {
	r.user.ID = userID
	return r.user, nil
}

func (r *memoryExportRepository) GetOrCreateEncryptedProfile(context.Context, uuid.UUID) (repository.EncryptedUserProfile, error) {
	return r.profile, nil
}

func (r *memoryExportRepository) UpdateEncryptedProfile(context.Context, repository.EncryptedUserProfile) (repository.EncryptedUserProfile, error) {
	return repository.EncryptedUserProfile{}, nil
}

func (r *memoryExportRepository) SaveItem(context.Context, uuid.UUID, uuid.UUID, repository.SavedItemKind) (uuid.UUID, error) {
	return uuid.Nil, nil
}

func (r *memoryExportRepository) RemoveItem(context.Context, uuid.UUID, uuid.UUID, repository.SavedItemKind) error {
	return nil
}

func (r *memoryExportRepository) ListItems(context.Context, uuid.UUID, *repository.SavedItemKind) ([]repository.SavedItem, error) {
	return r.saved, nil
}

func (r *memoryExportRepository) AddEncryptedHistory(context.Context, repository.EncryptedSearchHistoryEntry) (uuid.UUID, error) {
	return uuid.Nil, nil
}

func (r *memoryExportRepository) ListEncryptedHistory(context.Context, uuid.UUID, int) ([]repository.EncryptedSearchHistoryEntry, error) {
	return r.history, nil
}

func (r *memoryExportRepository) RecordConsent(context.Context, repository.ConsentRecord) (uuid.UUID, error) {
	return uuid.Nil, nil
}

func (r *memoryExportRepository) HasRequiredConsent(context.Context, uuid.UUID, string, string) (bool, error) {
	return true, nil
}

func (r *memoryExportRepository) ListConsent(context.Context, uuid.UUID) ([]repository.ConsentRecord, error) {
	return r.consent, nil
}

// TestExportServiceBuildsJSONAndCSV verifies DESIGN-008 DataExporter serialization.
func TestExportServiceBuildsJSONAndCSV(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	encryption := security.NewEncryptionService(keyLoader{active: "pii-v1", entries: map[string][]byte{"pii-v1": []byte("11111111111111111111111111111111")}})
	encrypt := func(value string) repository.EncryptedField {
		field, err := encryption.EncryptPII(ctx, []byte(value))
		if err != nil {
			t.Fatal(err)
		}
		return repository.EncryptedField{KeyVersion: field.KeyVersion, Nonce: field.Nonce, Ciphertext: field.Ciphertext}
	}
	display := encrypt("Ada")
	repo := &memoryExportRepository{
		user:    repository.EncryptedAuthUser{Email: encrypt("ada@example.test"), Role: repository.UserRoleUser},
		profile: repository.EncryptedUserProfile{UserID: userID, DisplayName: &display, UnitSystem: repository.UnitSystemMetric, ThemePreference: "dark"},
		saved:   []repository.SavedItem{{ID: uuid.New(), UserID: userID, ItemID: uuid.New(), Kind: repository.SavedItemKindFavorite}},
		history: []repository.EncryptedSearchHistoryEntry{{ID: uuid.New(), UserID: userID, Query: encrypt("tomato"), Mode: "search", FiltersHash: "hash"}},
		consent: []repository.ConsentRecord{{UserID: userID, PrivacyPolicyVersion: "privacy-v1", TermsVersion: "terms-v1"}},
	}
	service := NewExportService(repo, repo, repo, repo, repo, encryption)
	payload, err := service.BuildExport(ctx, userID, "json")
	if err != nil {
		t.Fatalf("BuildExport(json) error = %v", err)
	}
	var bundle ExportBundle
	if err := json.Unmarshal(payload.Body, &bundle); err != nil {
		t.Fatalf("decode export json: %v", err)
	}
	if bundle.User.Email != "ada@example.test" || bundle.User.DisplayName != "Ada" || len(bundle.SavedItems) != 1 || len(bundle.History) != 1 || len(bundle.CustomItems) != 0 {
		t.Fatalf("json bundle = %#v", bundle)
	}
	csvPayload, err := service.BuildExport(ctx, userID, "csv")
	if err != nil {
		t.Fatalf("BuildExport(csv) error = %v", err)
	}
	if !strings.Contains(string(csvPayload.Body), "history,search,tomato") || !strings.Contains(string(csvPayload.Body), "customItems,count,0") {
		t.Fatalf("csv body = %s", csvPayload.Body)
	}
	if _, err := service.BuildExport(ctx, userID, "xml"); err == nil {
		t.Fatal("BuildExport() accepted unsupported format")
	}
}
