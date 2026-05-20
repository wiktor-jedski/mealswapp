package exporter

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"mealswapp/backend/internal/repositories"

	"github.com/google/uuid"
)

func TestExporterBuildsUserDataExport(t *testing.T) {
	userID := uuid.New()
	source := fakeExportSource{
		user: repositories.UserEntity{
			ID:           userID,
			Email:        "user@example.com",
			DisplayName:  "User",
			PasswordHash: "secret-hash",
			Role:         "user",
		},
		preferences: repositories.PreferenceEntity{UserID: userID, Theme: "dark"},
		entitlement: repositories.EntitlementEntity{UserID: userID, Plan: "paid", Status: "active"},
		saved: []repositories.SavedDataEntity{
			{ID: uuid.New(), UserID: userID, Kind: "favorite", Label: "Tofu"},
			{ID: uuid.New(), UserID: userID, Kind: "search_history", Label: "lentils"},
		},
		consents: []repositories.ConsentRecordEntity{
			{UserID: userID, PrivacyPolicyVersion: "privacy-v1", TermsVersion: "terms-v1", NutritionDisclaimerVersion: "nutrition-v1"},
		},
	}
	exporter := New(source)
	exporter.now = func() time.Time {
		return time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC)
	}

	got, err := exporter.ExportUserData(context.Background(), userID)
	if err != nil {
		t.Fatal(err)
	}

	if got.Profile.ID != userID || got.Profile.Email != "user@example.com" || got.Preferences.Theme != "dark" || len(got.SavedData) != 2 || len(got.Consents) != 1 {
		t.Fatalf("unexpected export: %#v", got)
	}
}

func TestExporterRedactsSensitiveTokenAndHashData(t *testing.T) {
	userID := uuid.New()
	exporter := New(fakeExportSource{
		user:        repositories.UserEntity{ID: userID, Email: "user@example.com", PasswordHash: "secret-hash"},
		preferences: repositories.PreferenceEntity{UserID: userID},
		entitlement: repositories.EntitlementEntity{UserID: userID},
		saved:       []repositories.SavedDataEntity{{UserID: userID, Kind: "saved_search", Label: "Search", Payload: []byte(`{"token":"secret-token","query":"tofu"}`)}},
	})

	got, err := exporter.ExportUserData(context.Background(), userID)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	if strings.Contains(body, "secret-hash") {
		t.Fatalf("expected password hash redacted, got %s", body)
	}
	if strings.Contains(body, "secret-token") || strings.Contains(body, "secret-hash") {
		t.Fatalf("expected sensitive values redacted, got %s", body)
	}
}

func TestExporterRequestsOnlyOwnedUserData(t *testing.T) {
	userID := uuid.New()
	source := &recordingExportSource{fakeExportSource: fakeExportSource{
		user:        repositories.UserEntity{ID: userID, Email: "user@example.com"},
		preferences: repositories.PreferenceEntity{UserID: userID},
		entitlement: repositories.EntitlementEntity{UserID: userID},
	}}
	exporter := New(source)

	if _, err := exporter.ExportUserData(context.Background(), userID); err != nil {
		t.Fatal(err)
	}
	if source.savedUserID != userID || source.consentUserID != userID {
		t.Fatalf("expected owned user id passed to source, saved=%s consents=%s", source.savedUserID, source.consentUserID)
	}
}

type fakeExportSource struct {
	user        repositories.UserEntity
	preferences repositories.PreferenceEntity
	entitlement repositories.EntitlementEntity
	saved       []repositories.SavedDataEntity
	consents    []repositories.ConsentRecordEntity
}

func (source fakeExportSource) GetUser(ctx context.Context, userID uuid.UUID) (repositories.UserEntity, error) {
	return source.user, nil
}

func (source fakeExportSource) GetPreferences(ctx context.Context, userID uuid.UUID) (repositories.PreferenceEntity, error) {
	return source.preferences, nil
}

func (source fakeExportSource) GetEntitlement(ctx context.Context, userID uuid.UUID) (repositories.EntitlementEntity, error) {
	return source.entitlement, nil
}

func (source fakeExportSource) ListSavedData(ctx context.Context, userID uuid.UUID) ([]repositories.SavedDataEntity, error) {
	return source.saved, nil
}

func (source fakeExportSource) ListConsentRecords(ctx context.Context, userID uuid.UUID) ([]repositories.ConsentRecordEntity, error) {
	return source.consents, nil
}

type recordingExportSource struct {
	fakeExportSource
	savedUserID   uuid.UUID
	consentUserID uuid.UUID
}

func (source *recordingExportSource) ListSavedData(ctx context.Context, userID uuid.UUID) ([]repositories.SavedDataEntity, error) {
	source.savedUserID = userID
	return source.saved, nil
}

func (source *recordingExportSource) ListConsentRecords(ctx context.Context, userID uuid.UUID) ([]repositories.ConsentRecordEntity, error) {
	source.consentUserID = userID
	return source.consents, nil
}
