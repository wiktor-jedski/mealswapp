package search

import (
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// Implements DESIGN-002 FilterProcessor verification.

func TestApplyFiltersTranslatesIncludeAndExcludeFilters(t *testing.T) {
	categoryID := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	excludedCategoryID := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	roleID := uuid.MustParse("33333333-3333-4333-8333-333333333333")
	allergenID := uuid.MustParse("44444444-4444-4444-8444-444444444444")

	processed, rejection := ApplyFilters(ParsedQuery{NormalizedText: "apple", Limit: 10, Offset: 20}, []SearchFilter{
		{FilterID: categoryID.String(), Kind: SearchFilterKindFoodCategory, Include: true},
		{FilterID: excludedCategoryID.String(), Kind: SearchFilterKindFoodCategory, Include: false},
		{FilterID: roleID.String(), Kind: SearchFilterKindCulinaryRole, Include: true},
		{FilterID: allergenID.String(), Kind: SearchFilterKindAllergen, Include: false},
		{FilterID: string(repository.PhysicalStateSolid), Kind: SearchFilterKindPhysicalState, Include: true},
		{FilterID: string(repository.PhysicalStateLiquid), Kind: SearchFilterKindPhysicalState, Include: false},
	})
	if rejection != nil {
		t.Fatalf("ApplyFilters() rejection = %+v", rejection)
	}
	query := processed.RepositoryQuery
	if query.Name != "apple" || query.Limit != 10 || query.Offset != 20 {
		t.Fatalf("query text/page = %+v", query)
	}
	if !reflect.DeepEqual(query.FoodCategoryIDs, []uuid.UUID{categoryID}) {
		t.Fatalf("FoodCategoryIDs = %#v", query.FoodCategoryIDs)
	}
	if !reflect.DeepEqual(query.ExcludedFoodCategoryIDs, []uuid.UUID{excludedCategoryID}) {
		t.Fatalf("ExcludedFoodCategoryIDs = %#v", query.ExcludedFoodCategoryIDs)
	}
	if !reflect.DeepEqual(query.CulinaryRoleIDs, []uuid.UUID{roleID}) {
		t.Fatalf("CulinaryRoleIDs = %#v", query.CulinaryRoleIDs)
	}
	if !reflect.DeepEqual(query.ExcludedAllergenIDs, []uuid.UUID{allergenID}) || len(query.AllergenIDs) != 0 {
		t.Fatalf("allergen filters = include %#v exclude %#v", query.AllergenIDs, query.ExcludedAllergenIDs)
	}
	if !reflect.DeepEqual(query.FoodObjectTypes, []repository.PhysicalState{repository.PhysicalStateSolid}) {
		t.Fatalf("FoodObjectTypes = %#v", query.FoodObjectTypes)
	}
	if !reflect.DeepEqual(query.ExcludedFoodObjectTypes, []repository.PhysicalState{repository.PhysicalStateLiquid}) {
		t.Fatalf("ExcludedFoodObjectTypes = %#v", query.ExcludedFoodObjectTypes)
	}
	if len(processed.ExclusionRules) != 3 {
		t.Fatalf("ExclusionRules = %#v", processed.ExclusionRules)
	}
}

func TestApplyFiltersExpandsDietaryPresetsToExclusionRules(t *testing.T) {
	processed, rejection := ApplyFilters(ParsedQuery{Limit: 10}, []SearchFilter{
		{FilterID: string(DietaryPresetVegan), Kind: SearchFilterKindDietaryPreset, Include: false},
		{FilterID: string(DietaryPresetGlutenFree), Kind: SearchFilterKindDietaryPreset, Include: false},
	})
	if rejection != nil {
		t.Fatalf("ApplyFilters() rejection = %+v", rejection)
	}
	got := make([]string, 0, len(processed.ExclusionRules))
	for _, rule := range processed.ExclusionRules {
		got = append(got, string(rule.Kind)+":"+rule.FilterID+":"+rule.Source)
	}
	want := []string{
		"allergen:animal_product:vegan",
		"allergen:dairy:vegan",
		"allergen:egg:vegan",
		"allergen:gluten:gluten_free",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expanded rules = %#v, want %#v", got, want)
	}
	if len(processed.RepositoryQuery.FoodCategoryIDs) != 0 || len(processed.RepositoryQuery.CulinaryRoleIDs) != 0 {
		t.Fatalf("dietary preset created classification filters: %+v", processed.RepositoryQuery)
	}
	if !reflect.DeepEqual(processed.RepositoryQuery.ExcludedAllergenKeys, []string{"animal_product", "dairy", "egg", "gluten"}) {
		t.Fatalf("ExcludedAllergenKeys = %#v", processed.RepositoryQuery.ExcludedAllergenKeys)
	}
}

func TestApplyFiltersRejectsContradictoryFilters(t *testing.T) {
	id := uuid.MustParse("11111111-1111-4111-8111-111111111111").String()
	_, rejection := ApplyFilters(ParsedQuery{Limit: 10}, []SearchFilter{
		{FilterID: id, Kind: SearchFilterKindFoodCategory, Include: true},
		{FilterID: id, Kind: SearchFilterKindFoodCategory, Include: false},
	})
	if rejection == nil || rejection.Code != "rejected_search" || rejection.Field != "filters" {
		t.Fatalf("rejection = %+v", rejection)
	}
}

func TestApplyFiltersRejectsExclusionRuleConflicts(t *testing.T) {
	_, rejection := ApplyFilters(ParsedQuery{Limit: 10}, []SearchFilter{
		{FilterID: "dairy", Kind: SearchFilterKindAllergen, Include: true},
		{FilterID: string(DietaryPresetDairyFree), Kind: SearchFilterKindDietaryPreset, Include: false},
	})
	if rejection == nil || rejection.Code != "rejected_search" || rejection.Field != "filters" {
		t.Fatalf("rejection = %+v", rejection)
	}
}

func TestApplyFiltersRejectsUnsupportedShapes(t *testing.T) {
	for name, filters := range map[string][]SearchFilter{
		"included preset":            {{FilterID: string(DietaryPresetVegan), Kind: SearchFilterKindDietaryPreset, Include: true}},
		"unsupported preset":         {{FilterID: "keto", Kind: SearchFilterKindDietaryPreset, Include: false}},
		"unsupported allergen":       {{FilterID: "sesame", Kind: SearchFilterKindAllergen, Include: false}},
		"unsupported physical state": {{FilterID: "meal", Kind: SearchFilterKindPhysicalState, Include: true}},
		"invalid uuid":               {{FilterID: "fruit", Kind: SearchFilterKindFoodCategory, Include: true}},
	} {
		if _, rejection := ApplyFilters(ParsedQuery{Limit: 10}, filters); rejection == nil {
			t.Fatalf("%s accepted", name)
		}
	}
}

func TestApplyFiltersDeduplicatesEveryRepositoryFilterType(t *testing.T) {
	categoryID := uuid.New().String()
	allergenID := uuid.New().String()
	filters := []SearchFilter{
		{FilterID: categoryID, Kind: SearchFilterKindFoodCategory, Include: true},
		{FilterID: categoryID, Kind: SearchFilterKindFoodCategory, Include: true},
		{FilterID: allergenID, Kind: SearchFilterKindAllergen, Include: true},
		{FilterID: allergenID, Kind: SearchFilterKindAllergen, Include: true},
		{FilterID: "dairy", Kind: SearchFilterKindAllergen, Include: true},
		{FilterID: "dairy", Kind: SearchFilterKindAllergen, Include: true},
		{FilterID: string(repository.PhysicalStateSolid), Kind: SearchFilterKindPhysicalState, Include: true},
		{FilterID: string(repository.PhysicalStateSolid), Kind: SearchFilterKindPhysicalState, Include: true},
	}
	processed, rejection := ApplyFilters(ParsedQuery{Limit: 10}, filters)
	if rejection != nil {
		t.Fatalf("ApplyFilters() rejection = %+v", rejection)
	}
	query := processed.RepositoryQuery
	if len(query.FoodCategoryIDs) != 1 || len(query.AllergenIDs) != 1 || len(query.AllergenKeys) != 1 || len(query.FoodObjectTypes) != 1 {
		t.Fatalf("deduplicated query = %+v", query)
	}

	processed, rejection = ApplyFilters(ParsedQuery{Limit: 10}, []SearchFilter{
		{FilterID: string(DietaryPresetDairyFree), Kind: SearchFilterKindDietaryPreset, Include: false},
		{FilterID: string(DietaryPresetDairyFree), Kind: SearchFilterKindDietaryPreset, Include: false},
	})
	if rejection != nil || len(processed.ExclusionRules) != 1 {
		t.Fatalf("duplicate exclusion rules = %+v rejection=%+v", processed.ExclusionRules, rejection)
	}
}

func TestApplyFiltersRejectsUnknownFilterKind(t *testing.T) {
	_, rejection := ApplyFilters(ParsedQuery{Limit: 10}, []SearchFilter{{FilterID: "value", Kind: SearchFilterKind("unknown")}})
	if rejection == nil || rejection.Field != "filters" {
		t.Fatalf("rejection = %+v", rejection)
	}
}

func TestApplyFiltersRejectsEmptyFilterID(t *testing.T) {
	_, rejection := ApplyFilters(ParsedQuery{Limit: 10}, []SearchFilter{{Kind: SearchFilterKindAllergen}})
	if rejection == nil || rejection.Field != "filters" {
		t.Fatalf("rejection = %+v", rejection)
	}
}
