package preferences

import (
	"context"
	"errors"
	"testing"

	"mealswapp/backend/internal/http/apperrors"
	"mealswapp/backend/internal/repositories"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestManagerReturnsDefaultsWhenPreferencesMissing(t *testing.T) {
	userID := uuid.New()
	manager := NewManager(&fakePreferenceRepository{getErr: pgx.ErrNoRows})

	preference, err := manager.Get(context.Background(), userID)
	if err != nil {
		t.Fatal(err)
	}

	if preference.UserID != userID || preference.Theme != "system" || preference.DefaultSearchMode != "single" {
		t.Fatalf("unexpected defaults: %#v", preference)
	}
	if !preference.EnabledMacros["protein"] || !preference.EnabledMacros["carbs"] || !preference.EnabledMacros["fat"] {
		t.Fatalf("expected macro defaults enabled, got %#v", preference.EnabledMacros)
	}
}

func TestManagerUpdatesValidPreferences(t *testing.T) {
	userID := uuid.New()
	repo := &fakePreferenceRepository{stored: repositories.PreferenceEntity{
		UserID:            userID,
		Theme:             "system",
		DefaultSearchMode: "single",
		EnabledMacros:     map[string]bool{"protein": true, "carbs": true, "fat": true},
	}}
	manager := NewManager(repo)
	theme := "dark"
	mode := "diet"

	preference, err := manager.Update(context.Background(), userID, Update{
		Theme:             &theme,
		DefaultSearchMode: &mode,
		EnabledMacros:     map[string]bool{"protein": true, "carbs": false, "fat": true},
		ExcludedTagIDs:    []uuid.UUID{uuid.New()},
		DietaryFilterIDs:  []uuid.UUID{uuid.New()},
	})
	if err != nil {
		t.Fatal(err)
	}

	if preference.Theme != "dark" || preference.DefaultSearchMode != "diet" || preference.EnabledMacros["carbs"] {
		t.Fatalf("unexpected updated preference: %#v", preference)
	}
	if repo.upserted.UserID != userID || repo.upserted.Theme != "dark" {
		t.Fatalf("expected persisted preference, got %#v", repo.upserted)
	}
}

func TestManagerRejectsInvalidPreferences(t *testing.T) {
	userID := uuid.New()
	manager := NewManager(&fakePreferenceRepository{getErr: pgx.ErrNoRows})
	theme := "neon"

	_, err := manager.Update(context.Background(), userID, Update{Theme: &theme})
	appErr, ok := apperrors.As(err)
	if !ok {
		t.Fatalf("expected app error, got %v", err)
	}
	if appErr.Code != "validation_error" {
		t.Fatalf("expected validation error, got %#v", appErr)
	}
}

func TestManagerPropagatesRepositoryError(t *testing.T) {
	manager := NewManager(&fakePreferenceRepository{getErr: errors.New("database down")})

	_, err := manager.Get(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected repository error")
	}
}

type fakePreferenceRepository struct {
	stored   repositories.PreferenceEntity
	upserted repositories.PreferenceEntity
	getErr   error
}

func (repo *fakePreferenceRepository) Upsert(ctx context.Context, preference repositories.PreferenceEntity) error {
	repo.upserted = preference
	repo.stored = preference
	return nil
}

func (repo *fakePreferenceRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (repositories.PreferenceEntity, error) {
	if repo.getErr != nil {
		return repositories.PreferenceEntity{}, repo.getErr
	}
	return repo.stored, nil
}
