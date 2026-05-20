package search

import (
	"testing"

	"mealswapp/backend/internal/http/apperrors"

	"github.com/google/uuid"
)

func TestApplyFiltersBuildsIncludeExcludeTagQuery(t *testing.T) {
	includeID := uuid.MustParse("10000000-0000-0000-0000-000000000001")
	excludeID := uuid.MustParse("10000000-0000-0000-0000-000000000002")

	query, err := ApplyFilters(FilterInput{
		NormalizedSearch: "tofu",
		Limit:            10,
		Offset:           20,
		TagFilters: []TagFilter{
			{TagID: includeID.String(), Kind: TagFilterKindFunctionality, Include: true},
			{TagID: excludeID.String(), Kind: TagFilterKindCuration, Include: false},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if query.Text != "tofu" || query.Limit != 10 || query.Offset != 20 {
		t.Fatalf("unexpected base query fields: %#v", query.FoodItemQuery)
	}
	assertUUIDs(t, query.IncludeTagIDs, []uuid.UUID{includeID})
	assertUUIDs(t, query.ExcludeTagIDs, []uuid.UUID{excludeID})
}

func TestApplyFiltersExpandsDietaryAndAllergenFilters(t *testing.T) {
	dietID := uuid.MustParse("20000000-0000-0000-0000-000000000001")
	allergenID := uuid.MustParse("20000000-0000-0000-0000-000000000002")

	query, err := ApplyFilters(FilterInput{
		DietaryTagIDs:  []string{dietID.String()},
		AllergenTagIDs: []string{allergenID.String()},
	})
	if err != nil {
		t.Fatal(err)
	}

	assertUUIDs(t, query.IncludeTagIDs, []uuid.UUID{dietID})
	assertUUIDs(t, query.ExcludeTagIDs, []uuid.UUID{allergenID})
}

func TestApplyFiltersDeduplicatesCombinedFilters(t *testing.T) {
	tagID := uuid.MustParse("30000000-0000-0000-0000-000000000001")

	query, err := ApplyFilters(FilterInput{
		TagFilters: []TagFilter{
			{TagID: tagID.String(), Kind: TagFilterKindDiet, Include: true},
		},
		DietaryTagIDs: []string{tagID.String()},
	})
	if err != nil {
		t.Fatal(err)
	}

	assertUUIDs(t, query.IncludeTagIDs, []uuid.UUID{tagID})
}

func TestApplyFiltersValidatesMacros(t *testing.T) {
	_, err := ApplyFilters(FilterInput{
		EnabledMacros: map[string]bool{"protein": false, "carbs": false, "fat": false},
	})
	if !hasValidationField(t, err, "enabledMacros", "at_least_one_required") {
		t.Fatalf("expected all-disabled macro validation, got %v", err)
	}

	_, err = ApplyFilters(FilterInput{
		EnabledMacros: map[string]bool{"protein": true, "carbs": true, "fat": true, "calories": true},
	})
	if !hasValidationField(t, err, "enabledMacros.calories", "unsupported") {
		t.Fatalf("expected unsupported macro validation, got %v", err)
	}
}

func TestApplyFiltersNormalizesSourceProviders(t *testing.T) {
	query, err := ApplyFilters(FilterInput{
		SourceProviders: []string{" USDA ", "openfoodfacts", "usda"},
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{"usda", "openfoodfacts"}
	if len(query.SourceProviders) != len(expected) {
		t.Fatalf("expected providers %v, got %v", expected, query.SourceProviders)
	}
	for i := range expected {
		if query.SourceProviders[i] != expected[i] {
			t.Fatalf("expected providers %v, got %v", expected, query.SourceProviders)
		}
	}
}

func TestApplyFiltersRejectsInvalidTagShapes(t *testing.T) {
	_, err := ApplyFilters(FilterInput{
		TagFilters: []TagFilter{{TagID: "not-a-uuid", Kind: "category", Include: true}},
	})

	if !hasValidationField(t, err, "filters.0.tagId", "invalid") {
		t.Fatalf("expected invalid tag id validation, got %v", err)
	}
	if !hasValidationField(t, err, "filters.0.kind", "unsupported") {
		t.Fatalf("expected invalid tag kind validation, got %v", err)
	}
}

func TestApplyFiltersCombinesAllFilterTypes(t *testing.T) {
	includeID := uuid.MustParse("40000000-0000-0000-0000-000000000001")
	dietID := uuid.MustParse("40000000-0000-0000-0000-000000000002")
	allergenID := uuid.MustParse("40000000-0000-0000-0000-000000000003")

	query, err := ApplyFilters(FilterInput{
		NormalizedSearch: "beans",
		TagFilters:       []TagFilter{{TagID: includeID.String(), Kind: TagFilterKindFunctionality, Include: true}},
		DietaryTagIDs:    []string{dietID.String()},
		AllergenTagIDs:   []string{allergenID.String()},
		EnabledMacros:    map[string]bool{"protein": true, "carbs": true, "fat": false},
		SourceProviders:  []string{"USDA"},
		Limit:            10,
	})
	if err != nil {
		t.Fatal(err)
	}

	assertUUIDs(t, query.IncludeTagIDs, []uuid.UUID{includeID, dietID})
	assertUUIDs(t, query.ExcludeTagIDs, []uuid.UUID{allergenID})
	if query.EnabledMacros["fat"] || !query.EnabledMacros["protein"] || !query.EnabledMacros["carbs"] {
		t.Fatalf("unexpected macro toggles: %#v", query.EnabledMacros)
	}
	if len(query.SourceProviders) != 1 || query.SourceProviders[0] != "usda" {
		t.Fatalf("unexpected source providers: %#v", query.SourceProviders)
	}
}

func assertUUIDs(t *testing.T, actual []uuid.UUID, expected []uuid.UUID) {
	t.Helper()
	if len(actual) != len(expected) {
		t.Fatalf("expected UUIDs %v, got %v", expected, actual)
	}
	for i := range expected {
		if actual[i] != expected[i] {
			t.Fatalf("expected UUIDs %v, got %v", expected, actual)
		}
	}
}

func hasValidationField(t *testing.T, err error, field string, code string) bool {
	t.Helper()
	appErr, ok := apperrors.As(err)
	if !ok || appErr.Code != "validation_error" {
		return false
	}
	details, ok := appErr.Fields.([]map[string]string)
	if !ok {
		return false
	}
	for _, detailMap := range details {
		if detailMap["field"] == field && detailMap["code"] == code {
			return true
		}
	}
	return false
}
