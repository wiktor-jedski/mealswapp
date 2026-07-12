package userdata

// Implements DESIGN-008 DataExporter verification.

import (
	"context"
	"encoding/json"
	"errors"
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
	errAt   string
}

type memoryExportDiets struct {
	diets []repository.SavedDiet
}

func (r memoryExportDiets) Create(context.Context, uuid.UUID, repository.SavedDiet) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (r memoryExportDiets) Get(context.Context, uuid.UUID, uuid.UUID) (repository.SavedDiet, error) {
	return repository.SavedDiet{}, nil
}
func (r memoryExportDiets) List(context.Context, uuid.UUID) ([]repository.SavedDiet, error) {
	return r.diets, nil
}
func (r memoryExportDiets) Replace(context.Context, uuid.UUID, repository.SavedDiet) error {
	return nil
}
func (r memoryExportDiets) Delete(context.Context, uuid.UUID, uuid.UUID) error { return nil }

func (r *memoryExportRepository) GetEncryptedUserByID(_ context.Context, userID uuid.UUID) (repository.EncryptedAuthUser, error) {
	if r.errAt == "identity" {
		return repository.EncryptedAuthUser{}, errors.New("identity failed")
	}
	r.user.ID = userID
	return r.user, nil
}

func (r *memoryExportRepository) GetOrCreateEncryptedProfile(context.Context, uuid.UUID) (repository.EncryptedUserProfile, error) {
	if r.errAt == "profile" {
		return repository.EncryptedUserProfile{}, errors.New("profile failed")
	}
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
	if r.errAt == "saved" {
		return nil, errors.New("saved failed")
	}
	return r.saved, nil
}

func (r *memoryExportRepository) AddEncryptedHistory(context.Context, repository.EncryptedSearchHistoryEntry) (uuid.UUID, error) {
	return uuid.Nil, nil
}

func (r *memoryExportRepository) ListEncryptedHistory(context.Context, uuid.UUID, int) ([]repository.EncryptedSearchHistoryEntry, error) {
	if r.errAt == "history" {
		return nil, errors.New("history failed")
	}
	return r.history, nil
}

func (r *memoryExportRepository) RecordConsent(context.Context, repository.ConsentRecord) (uuid.UUID, error) {
	return uuid.Nil, nil
}

func (r *memoryExportRepository) HasRequiredConsent(context.Context, uuid.UUID, string, string) (bool, error) {
	return true, nil
}

func (r *memoryExportRepository) ListConsent(context.Context, uuid.UUID) ([]repository.ConsentRecord, error) {
	if r.errAt == "consent" {
		return nil, errors.New("consent failed")
	}
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
	dietID := uuid.New()
	diets := memoryExportDiets{diets: []repository.SavedDiet{{ID: dietID, UserID: userID, Name: "Training Day", Entries: []repository.SavedDietMealEntry{{MealID: uuid.New(), Position: 0}}}}}
	service := NewExportService(repo, repo, repo, repo, repo, encryption, diets)
	payload, err := service.BuildExport(ctx, userID, "json")
	if err != nil {
		t.Fatalf("BuildExport(json) error = %v", err)
	}
	var bundle ExportBundle
	if err := json.Unmarshal(payload.Body, &bundle); err != nil {
		t.Fatalf("decode export json: %v", err)
	}
	if bundle.User.Email != "ada@example.test" || bundle.User.DisplayName != "Ada" || len(bundle.SavedItems) != 1 || len(bundle.SavedDiets) != 1 || len(bundle.History) != 1 || len(bundle.CustomItems) != 0 {
		t.Fatalf("json bundle = %#v", bundle)
	}
	var rawBundle map[string]any
	if err := json.Unmarshal(payload.Body, &rawBundle); err != nil {
		t.Fatalf("decode raw export json: %v", err)
	}
	if _, ok := rawBundle["format"]; ok {
		t.Fatalf("json bundle leaked transport format: %s", payload.Body)
	}
	csvPayload, err := service.BuildExport(ctx, userID, "csv")
	if err != nil {
		t.Fatalf("BuildExport(csv) error = %v", err)
	}
	if !strings.Contains(string(csvPayload.Body), "history,search,tomato") || !strings.Contains(string(csvPayload.Body), "savedDiets,Training Day,"+dietID.String()) || !strings.Contains(string(csvPayload.Body), "customItems,count,0") {
		t.Fatalf("csv body = %s", csvPayload.Body)
	}
	if _, err := service.BuildExport(ctx, userID, "xml"); err == nil {
		t.Fatal("BuildExport() accepted unsupported format")
	}
}

func TestExportServicePropagatesValidationDependencyAndDecryptionErrors(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	key := []byte("11111111111111111111111111111111")
	encryption := security.NewEncryptionService(keyLoader{active: "pii-v1", entries: map[string][]byte{"pii-v1": key}})
	encrypt := func(value string) repository.EncryptedField {
		field, err := encryption.EncryptPII(ctx, []byte(value))
		if err != nil {
			t.Fatal(err)
		}
		return repository.EncryptedField{KeyVersion: field.KeyVersion, Nonce: field.Nonce, Ciphertext: field.Ciphertext}
	}
	valid := func() *memoryExportRepository {
		return &memoryExportRepository{
			user:    repository.EncryptedAuthUser{Email: encrypt("ada@example.test")},
			profile: repository.EncryptedUserProfile{UserID: userID, UnitSystem: repository.UnitSystemMetric, ThemePreference: "system"},
		}
	}
	if _, err := NewExportService(valid(), valid(), valid(), valid(), valid(), encryption).BuildExport(ctx, userID, ""); err == nil {
		t.Fatal("empty export format accepted")
	}
	for _, stage := range []string{"identity", "profile", "saved", "history", "consent"} {
		repo := valid()
		repo.errAt = stage
		if _, err := NewExportService(repo, repo, repo, repo, repo, encryption).BuildExport(ctx, userID, "json"); err == nil {
			t.Fatalf("%s failure ignored", stage)
		}
	}
	badEmail := valid()
	badEmail.user.Email.KeyVersion = "missing"
	if _, err := NewExportService(badEmail, badEmail, badEmail, badEmail, badEmail, encryption).BuildExport(ctx, userID, "json"); err == nil {
		t.Fatal("email decryption failure ignored")
	}
	badDisplay := valid()
	display := encrypt("Ada")
	display.KeyVersion = "missing"
	badDisplay.profile.DisplayName = &display
	if _, err := NewExportService(badDisplay, badDisplay, badDisplay, badDisplay, badDisplay, encryption).BuildExport(ctx, userID, "json"); err == nil {
		t.Fatal("display-name decryption failure ignored")
	}
	badHistory := valid()
	query := encrypt("apple")
	query.KeyVersion = "missing"
	badHistory.history = []repository.EncryptedSearchHistoryEntry{{Query: query}}
	if _, err := NewExportService(badHistory, badHistory, badHistory, badHistory, badHistory, encryption).BuildExport(ctx, userID, "json"); err == nil {
		t.Fatal("history decryption failure ignored")
	}
	defaultRole := valid()
	payload, err := NewExportService(defaultRole, defaultRole, defaultRole, defaultRole, defaultRole, encryption).BuildExport(ctx, userID, "json")
	if err != nil {
		t.Fatal(err)
	}
	var bundle ExportBundle
	if err := json.Unmarshal(payload.Body, &bundle); err != nil || bundle.User.Role != repository.UserRoleUser {
		t.Fatalf("default role bundle=%+v err=%v", bundle, err)
	}
}
