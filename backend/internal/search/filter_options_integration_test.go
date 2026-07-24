package search

import (
	"context"
	"testing"

	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// TestFilterOptionServiceReloadsActivePersistedVocabularyAfterAdministration
// verifies IT-ARCH-009-005, ARCH-009, DESIGN-009 TagManager, and SW-REQ-057.
// Implements DESIGN-009 TagManager active classification and administration invalidation verification.
func TestFilterOptionServiceReloadsActivePersistedVocabularyAfterAdministration(t *testing.T) {
	db := openAutocompleteTestDB(t)
	ctx := context.Background()
	classifications := repository.NewPostgresClassificationRepository(db)
	service := NewFilterOptionService(classifications, repository.NewPostgresAllergenVocabularyRepository(db))

	inactiveID, err := classifications.Upsert(ctx, repository.ClassificationEntity{Name: "Inactive category", Kind: repository.ClassificationKindFoodCategory})
	if err != nil {
		t.Fatalf("create inactive fixture: %v", err)
	}
	if err := classifications.SoftDelete(ctx, inactiveID); err != nil {
		t.Fatalf("deactivate fixture: %v", err)
	}
	before, err := service.Options(ctx, SearchModeSubstitution)
	if err != nil {
		t.Fatalf("Options() before administration error = %v", err)
	}
	if hasFilterOption(before.Options, SearchFilterKindFoodCategory, inactiveID.String()) {
		t.Fatal("Options() exposed inactive classification")
	}

	activeID, err := classifications.Upsert(ctx, repository.ClassificationEntity{Name: "Fresh category", Kind: repository.ClassificationKindFoodCategory})
	if err != nil {
		t.Fatalf("admin upsert fixture: %v", err)
	}
	stale, err := service.Options(ctx, SearchModeSubstitution)
	if err != nil {
		t.Fatalf("Options() cached error = %v", err)
	}
	if hasFilterOption(stale.Options, SearchFilterKindFoodCategory, activeID.String()) {
		t.Fatal("cached Options() changed without invalidation")
	}
	service.Invalidate()
	refreshed, err := service.Options(ctx, SearchModeSubstitution)
	if err != nil {
		t.Fatalf("Options() after invalidation error = %v", err)
	}
	if !hasFilterOption(refreshed.Options, SearchFilterKindFoodCategory, activeID.String()) || hasFilterOption(refreshed.Options, SearchFilterKindFoodCategory, inactiveID.String()) {
		t.Fatalf("Options() after invalidation = %#v", refreshed.Options)
	}
}

func hasFilterOption(options []FilterOption, kind SearchFilterKind, id string) bool {
	for _, option := range options {
		if option.Kind == kind && option.FilterID == id {
			return true
		}
	}
	return false
}
