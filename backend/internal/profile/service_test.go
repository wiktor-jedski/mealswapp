package profile

// Implements DESIGN-008 PreferenceManager verification.

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/security"
)

type keyLoader struct {
	active  string
	entries map[string][]byte
}

func (l keyLoader) ActiveKey(context.Context) (string, []byte, error) {
	return l.active, l.entries[l.active], nil
}

func (l keyLoader) Key(_ context.Context, version string) ([]byte, error) {
	key, ok := l.entries[version]
	if !ok {
		return nil, errors.New("missing key")
	}
	return key, nil
}

type memoryProfileRepository struct {
	profiles map[uuid.UUID]repository.EncryptedUserProfile
}

func (r *memoryProfileRepository) GetOrCreateEncryptedProfile(_ context.Context, userID uuid.UUID) (repository.EncryptedUserProfile, error) {
	if r.profiles == nil {
		r.profiles = map[uuid.UUID]repository.EncryptedUserProfile{}
	}
	profile, ok := r.profiles[userID]
	if !ok {
		profile = repository.EncryptedUserProfile{UserID: userID, UnitSystem: repository.UnitSystemMetric, ThemePreference: "system"}
		r.profiles[userID] = profile
	}
	return profile, nil
}

func (r *memoryProfileRepository) UpdateEncryptedProfile(_ context.Context, profile repository.EncryptedUserProfile) (repository.EncryptedUserProfile, error) {
	if r.profiles == nil {
		r.profiles = map[uuid.UUID]repository.EncryptedUserProfile{}
	}
	r.profiles[profile.UserID] = profile
	return profile, nil
}

// TestServiceProfilePreferences verifies DESIGN-008 PreferenceManager service behavior.
func TestServiceProfilePreferences(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	repo := &memoryProfileRepository{}
	service := NewService(repo, security.NewEncryptionService(keyLoader{active: "pii-v1", entries: map[string][]byte{"pii-v1": []byte("11111111111111111111111111111111")}}))

	profile, err := service.GetProfile(ctx, userID)
	if err != nil {
		t.Fatalf("GetProfile() error = %v", err)
	}
	if profile.UserID != userID || profile.DisplayName != "" || profile.UnitSystem != repository.UnitSystemMetric || profile.ThemePreference != "system" {
		t.Fatalf("default profile = %#v", profile)
	}
	name := "  Ada   Lovelace  "
	result, err := service.UpdatePreferences(ctx, userID, UpdateRequest{DisplayName: &name, UnitSystem: repository.UnitSystemImperial, ThemePreference: "dark"})
	if err != nil {
		t.Fatalf("UpdatePreferences() error = %v", err)
	}
	if !result.RequiresUnitRecalculation || result.Profile.DisplayName != "Ada Lovelace" || result.Profile.UnitSystem != repository.UnitSystemImperial || result.Profile.ThemePreference != "dark" {
		t.Fatalf("update result = %#v", result)
	}
	stored := repo.profiles[userID]
	if stored.DisplayName == nil || string(stored.DisplayName.Ciphertext) == "Ada Lovelace" {
		t.Fatalf("display name was not encrypted: %#v", stored.DisplayName)
	}
	next, err := service.UpdatePreferences(ctx, userID, UpdateRequest{UnitSystem: repository.UnitSystemImperial, ThemePreference: "light"})
	if err != nil {
		t.Fatalf("UpdatePreferences() same unit error = %v", err)
	}
	if next.RequiresUnitRecalculation || next.Profile.DisplayName != "Ada Lovelace" || next.Profile.ThemePreference != "light" {
		t.Fatalf("same-unit update = %#v", next)
	}
	if _, err := service.UpdatePreferences(ctx, userID, UpdateRequest{UnitSystem: "bad", ThemePreference: "system"}); err == nil {
		t.Fatal("UpdatePreferences() accepted invalid unit")
	}
	if _, err := service.UpdatePreferences(ctx, userID, UpdateRequest{UnitSystem: repository.UnitSystemMetric, ThemePreference: "bad"}); err == nil {
		t.Fatal("UpdatePreferences() accepted invalid theme")
	}
}
