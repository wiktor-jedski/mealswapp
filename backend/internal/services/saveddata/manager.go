package saveddata

import (
	"context"
	"slices"

	"mealswapp/backend/internal/http/apperrors"
	"mealswapp/backend/internal/repositories"

	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, saved repositories.SavedDataEntity) (uuid.UUID, error)
	GetByID(ctx context.Context, id uuid.UUID) (repositories.SavedDataEntity, error)
	ListByUser(ctx context.Context, userID uuid.UUID, kind string) ([]repositories.SavedDataEntity, error)
	Update(ctx context.Context, saved repositories.SavedDataEntity) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type Manager struct {
	repository Repository
}

func NewManager(repository Repository) Manager {
	return Manager{repository: repository}
}

func (manager Manager) Create(ctx context.Context, saved repositories.SavedDataEntity) (uuid.UUID, error) {
	if err := validate(saved); err != nil {
		return uuid.Nil, err
	}

	if saved.Kind == "search_history" {
		existing, err := manager.repository.ListByUser(ctx, saved.UserID, saved.Kind)
		if err != nil {
			return uuid.Nil, err
		}
		for _, item := range existing {
			if item.Label == saved.Label {
				return item.ID, nil
			}
		}
	}

	return manager.repository.Create(ctx, saved)
}

func (manager Manager) List(ctx context.Context, userID uuid.UUID, kind string) ([]repositories.SavedDataEntity, error) {
	if kind != "" && !validKind(kind) {
		return nil, apperrors.Validation("Saved data validation failed", []map[string]string{{"field": "kind", "code": "unsupported"}})
	}
	return manager.repository.ListByUser(ctx, userID, kind)
}

func (manager Manager) Get(ctx context.Context, userID uuid.UUID, id uuid.UUID) (repositories.SavedDataEntity, error) {
	saved, err := manager.repository.GetByID(ctx, id)
	if err != nil {
		return repositories.SavedDataEntity{}, err
	}
	if saved.UserID != userID {
		return repositories.SavedDataEntity{}, apperrors.Forbidden("Forbidden")
	}
	return saved, nil
}

func (manager Manager) Update(ctx context.Context, userID uuid.UUID, saved repositories.SavedDataEntity) error {
	current, err := manager.Get(ctx, userID, saved.ID)
	if err != nil {
		return err
	}
	saved.UserID = current.UserID
	if saved.Kind == "" {
		saved.Kind = current.Kind
	}
	if err := validate(saved); err != nil {
		return err
	}
	return manager.repository.Update(ctx, saved)
}

func (manager Manager) Delete(ctx context.Context, userID uuid.UUID, id uuid.UUID) error {
	if _, err := manager.Get(ctx, userID, id); err != nil {
		return err
	}
	return manager.repository.Delete(ctx, id)
}

func validate(saved repositories.SavedDataEntity) error {
	var fields []map[string]string
	if saved.UserID == uuid.Nil {
		fields = append(fields, map[string]string{"field": "userId", "code": "required"})
	}
	if !validKind(saved.Kind) {
		fields = append(fields, map[string]string{"field": "kind", "code": "unsupported"})
	}
	if saved.Label == "" {
		fields = append(fields, map[string]string{"field": "label", "code": "required"})
	}
	if len(fields) > 0 {
		return apperrors.Validation("Saved data validation failed", fields)
	}
	return nil
}

func validKind(kind string) bool {
	return slices.Contains([]string{"favorite", "saved_search", "search_history"}, kind)
}
