package search

// Implements DESIGN-002 FilterProcessor repository integration verification.

import (
	"context"
	"testing"

	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

func TestApplyFiltersDietaryPresetExcludesRepositoryAllergenKeys(t *testing.T) {
	db := openAutocompleteTestDB(t)
	ctx := context.Background()
	foodRepo := repository.NewPostgresFoodItemRepository(db)

	allowedID, err := foodRepo.Create(ctx, repository.FoodItemEntity{
		Name:                            "Oat Milk",
		PhysicalState:                   repository.PhysicalStateLiquid,
		DensityGramsPerMilliliter:       1.03,
		DensitySourceKind:               "estimated",
		AverageServingVolumeMilliliters: 240,
		MacrosPer100:                    repository.MacroValues{Protein: 1, Carbohydrates: 7, Fat: 1.5},
	})
	if err != nil {
		t.Fatalf("create allowed food: %v", err)
	}
	excludedID, err := foodRepo.Create(ctx, repository.FoodItemEntity{
		Name:                            "Cow Milk",
		PhysicalState:                   repository.PhysicalStateLiquid,
		DensityGramsPerMilliliter:       1.03,
		DensitySourceKind:               "estimated",
		AverageServingVolumeMilliliters: 240,
		MacrosPer100:                    repository.MacroValues{Protein: 3.4, Carbohydrates: 5, Fat: 1},
	})
	if err != nil {
		t.Fatalf("create excluded food: %v", err)
	}
	if _, err := db.Exec(ctx, "INSERT INTO food_item_allergens (food_item_id, allergen_key) VALUES ($1, $2)", excludedID, "dairy"); err != nil {
		t.Fatalf("attach dairy allergen: %v", err)
	}

	processed, rejection := ApplyFilters(ParsedQuery{Limit: 10}, []SearchFilter{
		{FilterID: string(DietaryPresetDairyFree), Kind: SearchFilterKindDietaryPreset, Include: false},
	})
	if rejection != nil {
		t.Fatalf("ApplyFilters() rejection = %+v", rejection)
	}

	items, total, err := foodRepo.Search(ctx, processed.RepositoryQuery)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].ID != allowedID {
		t.Fatalf("Search() total=%d items=%#v, want only allergen-free item %s", total, items, allowedID)
	}

	classificationRepo := repository.NewPostgresClassificationRepository(db)
	foodCategories, err := classificationRepo.List(ctx, repository.ClassificationKindFoodCategory)
	if err != nil {
		t.Fatalf("list food categories: %v", err)
	}
	culinaryRoles, err := classificationRepo.List(ctx, repository.ClassificationKindCulinaryRole)
	if err != nil {
		t.Fatalf("list culinary roles: %v", err)
	}
	for _, classification := range append(foodCategories, culinaryRoles...) {
		if classification.Name == string(DietaryPresetDairyFree) {
			t.Fatalf("dietary preset created misleading classification row: %#v", classification)
		}
	}
}
