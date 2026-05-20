package externaldata

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"mealswapp/backend/internal/cache"
	"mealswapp/backend/internal/domain/food"
	"mealswapp/backend/internal/repositories"

	"github.com/google/uuid"
)

var (
	ErrImportDraftInvalid = errors.New("curated import draft is invalid")
	ErrImportConflict     = errors.New("import conflicts with an existing food item")
)

type FoodItemStore interface {
	Create(ctx context.Context, item food.FoodItemEntity) (uuid.UUID, error)
	Search(ctx context.Context, query repositories.FoodItemQuery) ([]food.FoodItemEntity, int, error)
}

type ImportRecordStore interface {
	Create(ctx context.Context, importRecord repositories.ImportEntity) (uuid.UUID, error)
}

type CacheInvalidator interface {
	Invalidate(ctx context.Context, event cache.InvalidationEvent) (cache.InvalidationResult, error)
}

type CuratedImportDraft struct {
	Provider       Provider              `json:"provider"`
	ExternalID     string                `json:"externalId"`
	Name           string                `json:"name"`
	PhysicalState  food.PhysicalState    `json:"physicalState"`
	MacrosPer100   food.MacroValues      `json:"macrosPer100"`
	CaloriesPer100 float64               `json:"caloriesPer100"`
	Micros         map[string]float64    `json:"micros"`
	ServingSize    float64               `json:"servingSize"`
	ServingUnit    food.ServingUnit      `json:"servingUnit"`
	ImageURL       string                `json:"imageUrl"`
	Warnings       []ExternalDataWarning `json:"warnings,omitempty"`
}

type ImportResult struct {
	FoodItemID     uuid.UUID                `json:"foodItemId"`
	ImportRecordID uuid.UUID                `json:"importRecordId"`
	Warnings       []ExternalDataWarning    `json:"warnings,omitempty"`
	Invalidation   cache.InvalidationResult `json:"invalidation"`
}

type DataImporter struct {
	foods       FoodItemStore
	imports     ImportRecordStore
	invalidator CacheInvalidator
	now         func() time.Time
}

func NewDataImporter(foods FoodItemStore, imports ImportRecordStore, invalidator CacheInvalidator) DataImporter {
	return DataImporter{foods: foods, imports: imports, invalidator: invalidator, now: time.Now}
}

func NewCuratedImportDraft(candidate NormalizedFoodCandidate) CuratedImportDraft {
	servingSize := 100.0
	if candidate.ServingSize != nil && *candidate.ServingSize > 0 {
		servingSize = *candidate.ServingSize
	}
	servingUnit := food.ServingUnitGram
	switch candidate.ServingUnit {
	case string(food.ServingUnitMilliliter):
		servingUnit = food.ServingUnitMilliliter
	case string(food.ServingUnitPiece):
		servingUnit = food.ServingUnitPiece
	case string(food.ServingUnitServing):
		servingUnit = food.ServingUnitServing
	}
	physicalState := candidate.PhysicalState
	if !physicalState.Valid() {
		physicalState = food.PhysicalStateSolid
	}
	return CuratedImportDraft{
		Provider:       candidate.Provider,
		ExternalID:     candidate.ExternalID,
		Name:           candidate.Name,
		PhysicalState:  physicalState,
		MacrosPer100:   candidate.MacrosPer100,
		CaloriesPer100: candidate.CaloriesPer100,
		Micros:         candidate.Micros,
		ServingSize:    servingSize,
		ServingUnit:    servingUnit,
		ImageURL:       candidate.ImageURL,
		Warnings:       candidate.Warnings,
	}
}

func (importer DataImporter) Import(ctx context.Context, draft CuratedImportDraft) (ImportResult, error) {
	if importer.foods == nil || importer.imports == nil {
		return ImportResult{}, ErrImportDraftInvalid
	}
	if err := validateImportDraft(draft); err != nil {
		return ImportResult{}, err
	}
	if err := importer.ensureNoDuplicate(ctx, draft); err != nil {
		return ImportResult{}, err
	}

	now := importer.now().UTC()
	item := food.FoodItemEntity{
		Name:           strings.TrimSpace(draft.Name),
		PhysicalState:  draft.PhysicalState,
		ServingUnit:    draft.ServingUnit,
		ServingSize:    draft.ServingSize,
		CaloriesPer100: draft.CaloriesPer100,
		MacrosPer100:   draft.MacrosPer100,
		Micros:         draft.Micros,
		Source: food.SourceMetadata{
			Provider:      string(draft.Provider),
			ExternalID:    strings.TrimSpace(draft.ExternalID),
			ImportedAt:    &now,
			CurationState: "approved",
		},
		ImageURL: strings.TrimSpace(draft.ImageURL),
	}
	foodItemID, err := importer.foods.Create(ctx, item)
	if err != nil {
		return ImportResult{}, err
	}

	payload, err := json.Marshal(map[string]any{
		"foodItemId": foodItemID,
		"warnings":   draft.Warnings,
	})
	if err != nil {
		return ImportResult{}, err
	}
	importRecordID, err := importer.imports.Create(ctx, repositories.ImportEntity{
		Provider:   string(draft.Provider),
		ExternalID: strings.TrimSpace(draft.ExternalID),
		Status:     "imported",
		Payload:    payload,
	})
	if err != nil {
		return ImportResult{}, err
	}

	var invalidation cache.InvalidationResult
	if importer.invalidator != nil {
		invalidation, err = importer.invalidator.Invalidate(ctx, cache.InvalidationEvent{
			ItemIDs:   []uuid.UUID{foodItemID},
			Reason:    cache.ReasonImportChanged,
			CreatedAt: now,
		})
		if err != nil {
			return ImportResult{}, err
		}
	}

	return ImportResult{FoodItemID: foodItemID, ImportRecordID: importRecordID, Warnings: draft.Warnings, Invalidation: invalidation}, nil
}

func validateImportDraft(draft CuratedImportDraft) error {
	item := food.FoodItemEntity{
		Name:           strings.TrimSpace(draft.Name),
		PhysicalState:  draft.PhysicalState,
		ServingUnit:    draft.ServingUnit,
		ServingSize:    draft.ServingSize,
		CaloriesPer100: draft.CaloriesPer100,
		MacrosPer100:   draft.MacrosPer100,
		Micros:         draft.Micros,
	}
	if strings.TrimSpace(string(draft.Provider)) == "" || strings.TrimSpace(draft.ExternalID) == "" {
		return ErrImportDraftInvalid
	}
	if err := item.Validate(); err != nil {
		return err
	}
	return nil
}

func (importer DataImporter) ensureNoDuplicate(ctx context.Context, draft CuratedImportDraft) error {
	items, _, err := importer.foods.Search(ctx, repositories.FoodItemQuery{Text: strings.TrimSpace(draft.Name), Limit: 20})
	if err != nil {
		return err
	}
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.Source.Provider), strings.TrimSpace(string(draft.Provider))) &&
			strings.EqualFold(strings.TrimSpace(item.Source.ExternalID), strings.TrimSpace(draft.ExternalID)) {
			return ErrImportConflict
		}
		if strings.EqualFold(strings.TrimSpace(item.Name), strings.TrimSpace(draft.Name)) {
			return ErrImportConflict
		}
	}
	return nil
}
