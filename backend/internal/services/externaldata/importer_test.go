package externaldata

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"mealswapp/backend/internal/cache"
	"mealswapp/backend/internal/domain/food"
	"mealswapp/backend/internal/repositories"

	"github.com/google/uuid"
)

func TestDataImporterImportsDraftPersistsWarningsAndInvalidatesCache(t *testing.T) {
	foods := &fakeFoodItemStore{createdID: uuid.New()}
	imports := &fakeImportRecordStore{createdID: uuid.New()}
	invalidator := &fakeCacheInvalidator{result: cache.InvalidationResult{TagsDeleted: []string{"namespace:search", "namespace:similarity"}, KeysDeleted: 2}}
	importer := NewDataImporter(foods, imports, invalidator)
	importer.now = func() time.Time { return time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC) }
	draft := validImportDraft()
	draft.Warnings = []ExternalDataWarning{{Provider: ProviderUSDA, ExternalID: draft.ExternalID, Code: "missing_image", Message: "Missing image"}}

	result, err := importer.Import(context.Background(), draft)
	if err != nil {
		t.Fatalf("unexpected import error: %v", err)
	}

	if result.FoodItemID != foods.createdID || result.ImportRecordID != imports.createdID {
		t.Fatalf("unexpected result IDs: %#v", result)
	}
	if foods.created.Source.Provider != "usda" || foods.created.Source.ExternalID != "1101" || foods.created.Source.CurationState != "approved" {
		t.Fatalf("unexpected created food source: %#v", foods.created.Source)
	}
	if foods.created.Source.ImportedAt == nil || !foods.created.Source.ImportedAt.Equal(time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("expected imported timestamp, got %#v", foods.created.Source.ImportedAt)
	}
	if imports.created.Status != "imported" || imports.created.Provider != "usda" || imports.created.ExternalID != "1101" {
		t.Fatalf("unexpected import record: %#v", imports.created)
	}
	var payload map[string]any
	if err := json.Unmarshal(imports.created.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if warnings, ok := payload["warnings"].([]any); !ok || len(warnings) != 1 {
		t.Fatalf("expected persisted warnings payload, got %#v", payload)
	}
	if len(invalidator.events) != 1 || invalidator.events[0].Reason != cache.ReasonImportChanged || invalidator.events[0].ItemIDs[0] != foods.createdID {
		t.Fatalf("expected import cache invalidation, got %#v", invalidator.events)
	}
}

func TestDataImporterDetectsDuplicateSourceAndName(t *testing.T) {
	draft := validImportDraft()
	cases := []struct {
		name  string
		items []food.FoodItemEntity
	}{
		{
			name: "source",
			items: []food.FoodItemEntity{{
				Name: "Other Cheese",
				Source: food.SourceMetadata{
					Provider:   "usda",
					ExternalID: "1101",
				},
			}},
		},
		{
			name:  "name",
			items: []food.FoodItemEntity{{Name: " cheddar cheese "}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			importer := NewDataImporter(&fakeFoodItemStore{searchItems: tc.items}, &fakeImportRecordStore{}, nil)
			_, err := importer.Import(context.Background(), draft)
			if !errors.Is(err, ErrImportConflict) {
				t.Fatalf("expected import conflict, got %v", err)
			}
		})
	}
}

func TestDataImporterRejectsInvalidDraft(t *testing.T) {
	importer := NewDataImporter(&fakeFoodItemStore{}, &fakeImportRecordStore{}, nil)

	_, err := importer.Import(context.Background(), CuratedImportDraft{Name: "No provider"})

	if !errors.Is(err, ErrImportDraftInvalid) {
		t.Fatalf("expected invalid draft, got %v", err)
	}
}

func TestNewCuratedImportDraftDefaultsEditableFields(t *testing.T) {
	candidate := NormalizedFoodCandidate{
		Provider:       ProviderOpenFoodFacts,
		ExternalID:     "737628064502",
		Name:           "Tofu",
		PhysicalState:  food.PhysicalStateLiquid,
		MacrosPer100:   food.MacroValues{ProteinGrams: 12, CarbsGrams: 2, FatGrams: 6},
		CaloriesPer100: 110,
		ServingUnit:    "milliliter",
	}

	draft := NewCuratedImportDraft(candidate)

	if draft.ServingSize != 100 || draft.ServingUnit != food.ServingUnitMilliliter || draft.PhysicalState != food.PhysicalStateLiquid {
		t.Fatalf("unexpected draft defaults: %#v", draft)
	}
}

func validImportDraft() CuratedImportDraft {
	return CuratedImportDraft{
		Provider:       ProviderUSDA,
		ExternalID:     "1101",
		Name:           "Cheddar Cheese",
		PhysicalState:  food.PhysicalStateSolid,
		ServingUnit:    food.ServingUnitGram,
		ServingSize:    100,
		CaloriesPer100: 403,
		MacrosPer100:   food.MacroValues{ProteinGrams: 24.9, CarbsGrams: 1.3, FatGrams: 33.1},
		Micros:         map[string]float64{"Calcium": 710},
		ImageURL:       "https://example.test/cheddar.jpg",
	}
}

type fakeFoodItemStore struct {
	searchItems []food.FoodItemEntity
	createdID   uuid.UUID
	created     food.FoodItemEntity
}

func (store *fakeFoodItemStore) Create(ctx context.Context, item food.FoodItemEntity) (uuid.UUID, error) {
	store.created = item
	if store.createdID == uuid.Nil {
		store.createdID = uuid.New()
	}
	return store.createdID, nil
}

func (store *fakeFoodItemStore) Search(ctx context.Context, query repositories.FoodItemQuery) ([]food.FoodItemEntity, int, error) {
	return store.searchItems, len(store.searchItems), nil
}

type fakeImportRecordStore struct {
	createdID uuid.UUID
	created   repositories.ImportEntity
}

func (store *fakeImportRecordStore) Create(ctx context.Context, importRecord repositories.ImportEntity) (uuid.UUID, error) {
	store.created = importRecord
	if store.createdID == uuid.Nil {
		store.createdID = uuid.New()
	}
	return store.createdID, nil
}

type fakeCacheInvalidator struct {
	events []cache.InvalidationEvent
	result cache.InvalidationResult
}

func (invalidator *fakeCacheInvalidator) Invalidate(ctx context.Context, event cache.InvalidationEvent) (cache.InvalidationResult, error) {
	invalidator.events = append(invalidator.events, event)
	return invalidator.result, nil
}
