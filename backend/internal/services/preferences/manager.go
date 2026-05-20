package preferences

import (
	"context"
	"slices"

	"mealswapp/backend/internal/http/apperrors"
	"mealswapp/backend/internal/repositories"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var macroKeys = []string{"protein", "carbs", "fat"}

type Repository interface {
	Upsert(ctx context.Context, preference repositories.PreferenceEntity) error
	GetByUserID(ctx context.Context, userID uuid.UUID) (repositories.PreferenceEntity, error)
}

type Manager struct {
	repository Repository
}

type Update struct {
	Theme             *string
	DefaultSearchMode *string
	EnabledMacros     map[string]bool
	ExcludedTagIDs    []uuid.UUID
	DietaryFilterIDs  []uuid.UUID
}

func NewManager(repository Repository) Manager {
	return Manager{repository: repository}
}

func (manager Manager) Defaults(userID uuid.UUID) repositories.PreferenceEntity {
	return repositories.PreferenceEntity{
		UserID:            userID,
		Theme:             "system",
		DefaultSearchMode: "single",
		EnabledMacros:     map[string]bool{"protein": true, "carbs": true, "fat": true},
		ExcludedTagIDs:    []uuid.UUID{},
		DietaryFilterIDs:  []uuid.UUID{},
	}
}

func (manager Manager) Get(ctx context.Context, userID uuid.UUID) (repositories.PreferenceEntity, error) {
	preference, err := manager.repository.GetByUserID(ctx, userID)
	if err == nil {
		return preference, nil
	}
	if err == pgx.ErrNoRows {
		return manager.Defaults(userID), nil
	}
	return repositories.PreferenceEntity{}, err
}

func (manager Manager) Update(ctx context.Context, userID uuid.UUID, update Update) (repositories.PreferenceEntity, error) {
	preference, err := manager.Get(ctx, userID)
	if err != nil {
		return repositories.PreferenceEntity{}, err
	}

	if update.Theme != nil {
		preference.Theme = *update.Theme
	}
	if update.DefaultSearchMode != nil {
		preference.DefaultSearchMode = *update.DefaultSearchMode
	}
	if update.EnabledMacros != nil {
		preference.EnabledMacros = update.EnabledMacros
	}
	if update.ExcludedTagIDs != nil {
		preference.ExcludedTagIDs = update.ExcludedTagIDs
	}
	if update.DietaryFilterIDs != nil {
		preference.DietaryFilterIDs = update.DietaryFilterIDs
	}

	if err := validate(preference); err != nil {
		return repositories.PreferenceEntity{}, err
	}
	if err := manager.repository.Upsert(ctx, preference); err != nil {
		return repositories.PreferenceEntity{}, err
	}
	return preference, nil
}

func validate(preference repositories.PreferenceEntity) error {
	var fields []map[string]string
	if !slices.Contains([]string{"system", "light", "dark"}, preference.Theme) {
		fields = append(fields, map[string]string{"field": "theme", "code": "unsupported"})
	}
	if !slices.Contains([]string{"single", "replacement", "diet"}, preference.DefaultSearchMode) {
		fields = append(fields, map[string]string{"field": "defaultSearchMode", "code": "unsupported"})
	}
	for _, key := range macroKeys {
		if _, ok := preference.EnabledMacros[key]; !ok {
			fields = append(fields, map[string]string{"field": "enabledMacros." + key, "code": "required"})
		}
	}
	for key := range preference.EnabledMacros {
		if !slices.Contains(macroKeys, key) {
			fields = append(fields, map[string]string{"field": "enabledMacros." + key, "code": "unsupported"})
		}
	}
	if len(fields) > 0 {
		return apperrors.Validation("Preference validation failed", fields)
	}
	return nil
}
