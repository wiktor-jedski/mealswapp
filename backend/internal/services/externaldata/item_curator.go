package externaldata

import (
	"context"
	"errors"
	"strings"
	"time"

	"mealswapp/backend/internal/cache"
	"mealswapp/backend/internal/domain/food"
	"mealswapp/backend/internal/repositories"

	"github.com/google/uuid"
)

var (
	ErrItemCuratorInvalidTransition = errors.New("invalid curation transition")
)

type FoodItemCuratorStore interface {
	Create(ctx context.Context, item food.FoodItemEntity) (uuid.UUID, error)
	GetByID(ctx context.Context, id uuid.UUID, rc repositories.RepositoryContext) (food.FoodItemEntity, error)
	Search(ctx context.Context, query repositories.FoodItemQuery) ([]food.FoodItemEntity, int, error)
	Update(ctx context.Context, item food.FoodItemEntity) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type ItemCurator struct {
	foods       FoodItemCuratorStore
	invalidator CacheInvalidator
	now         func() time.Time
}

type ItemListResult struct {
	Items []food.FoodItemEntity `json:"items"`
	Total int                   `json:"total"`
	Page  int                   `json:"page"`
	Limit int                   `json:"limit"`
}

type CurationTransition string

const (
	TransitionApprove    CurationTransition = "approve"
	TransitionReject     CurationTransition = "reject"
	TransitionDeactivate CurationTransition = "deactivate"
)

func NewItemCurator(foods FoodItemCuratorStore, invalidator CacheInvalidator) ItemCurator {
	return ItemCurator{foods: foods, invalidator: invalidator, now: time.Now}
}

func (curator ItemCurator) List(ctx context.Context, query string, page int, limit int) (ItemListResult, error) {
	if limit <= 0 {
		limit = 10
	}
	if page <= 0 {
		page = 1
	}
	items, total, err := curator.foods.Search(ctx, repositories.FoodItemQuery{Text: query, Limit: limit, Offset: (page - 1) * limit})
	if err != nil {
		return ItemListResult{}, err
	}
	return ItemListResult{Items: items, Total: total, Page: page, Limit: limit}, nil
}

func (curator ItemCurator) Get(ctx context.Context, id uuid.UUID) (food.FoodItemEntity, error) {
	return curator.foods.GetByID(ctx, id, repositories.RepositoryContext{})
}

func (curator ItemCurator) Create(ctx context.Context, item food.FoodItemEntity) (food.FoodItemEntity, error) {
	if strings.TrimSpace(item.Source.CurationState) == "" {
		item.Source.CurationState = "draft"
	}
	id, err := curator.foods.Create(ctx, item)
	if err != nil {
		return food.FoodItemEntity{}, err
	}
	item.ID = id
	if err := curator.invalidate(ctx, id); err != nil {
		return food.FoodItemEntity{}, err
	}
	return item, nil
}

func (curator ItemCurator) Update(ctx context.Context, id uuid.UUID, patch food.FoodItemEntity) (food.FoodItemEntity, error) {
	current, err := curator.foods.GetByID(ctx, id, repositories.RepositoryContext{})
	if err != nil {
		return food.FoodItemEntity{}, err
	}
	current.Name = patch.Name
	current.PhysicalState = patch.PhysicalState
	current.ServingUnit = patch.ServingUnit
	current.ServingSize = patch.ServingSize
	current.CaloriesPer100 = patch.CaloriesPer100
	current.MacrosPer100 = patch.MacrosPer100
	current.Micros = patch.Micros
	current.ImageURL = patch.ImageURL
	current.PrepTimeMinutes = patch.PrepTimeMinutes
	current.AverageUnitWeightGrams = patch.AverageUnitWeightGrams
	current.Source.Provider = patch.Source.Provider
	current.Source.ExternalID = patch.Source.ExternalID
	current.Source.ProviderURL = patch.Source.ProviderURL
	if patch.Source.ImportedAt != nil {
		current.Source.ImportedAt = patch.Source.ImportedAt
	}
	if strings.TrimSpace(patch.Source.CurationState) != "" {
		current.Source.CurationState = patch.Source.CurationState
	}
	if err := curator.foods.Update(ctx, current); err != nil {
		return food.FoodItemEntity{}, err
	}
	if err := curator.invalidate(ctx, id); err != nil {
		return food.FoodItemEntity{}, err
	}
	return current, nil
}

func (curator ItemCurator) Transition(ctx context.Context, id uuid.UUID, transition CurationTransition) (food.FoodItemEntity, error) {
	current, err := curator.foods.GetByID(ctx, id, repositories.RepositoryContext{})
	if err != nil {
		return food.FoodItemEntity{}, err
	}
	switch transition {
	case TransitionApprove:
		current.Source.CurationState = "approved"
	case TransitionReject:
		current.Source.CurationState = "rejected"
	case TransitionDeactivate:
		current.Source.CurationState = "inactive"
	default:
		return food.FoodItemEntity{}, ErrItemCuratorInvalidTransition
	}
	if err := curator.foods.Update(ctx, current); err != nil {
		return food.FoodItemEntity{}, err
	}
	if err := curator.invalidate(ctx, id); err != nil {
		return food.FoodItemEntity{}, err
	}
	return current, nil
}

func (curator ItemCurator) Delete(ctx context.Context, id uuid.UUID) error {
	if err := curator.foods.Delete(ctx, id); err != nil {
		return err
	}
	return curator.invalidate(ctx, id)
}

func (curator ItemCurator) invalidate(ctx context.Context, id uuid.UUID) error {
	if curator.invalidator == nil {
		return nil
	}
	_, err := curator.invalidator.Invalidate(ctx, cache.InvalidationEvent{ItemIDs: []uuid.UUID{id}, Reason: cache.ReasonFoodUpdated, CreatedAt: curator.now().UTC()})
	return err
}
