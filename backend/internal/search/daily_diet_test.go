package search

import (
	"context"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// Implements DESIGN-002 QueryParser daily diet alternative boundary verification.

func TestPrepareSearchRequestRejectsDailyDietModeWithoutDailyDietID(t *testing.T) {
	_, err := PrepareSearchRequest(SearchRequest{
		Query: "lentil",
		Mode:  SearchModeDailyDietAlternative,
		Page:  1,
	}, DailyDietDataUnavailable)
	if err == nil {
		t.Fatal("daily diet alternative search without dailyDietId accepted")
	}
}

func TestPrepareSearchRequestReturnsDeterministicPhase07Rejection(t *testing.T) {
	dailyDietID := uuid.MustParse("61e0cae4-0f45-4854-8ac5-b228214cdd1d")
	prepared, err := PrepareSearchRequest(SearchRequest{
		Query:       " Lentil   soup ",
		Mode:        SearchModeDailyDietAlternative,
		Page:        2,
		DailyDietID: &dailyDietID,
	}, DailyDietDataUnavailable)
	if err != nil {
		t.Fatal(err)
	}
	if prepared.ParsedQuery.Strategy != SearchStrategyDailyDietAlternative || prepared.ParsedQuery.Offset != 10 {
		t.Fatalf("parsed query = %+v", prepared.ParsedQuery)
	}
	if prepared.Rejection == nil || prepared.Rejection.Code != "phase_07_saved_diet_unavailable" || prepared.Rejection.Field != "dailyDietId" {
		t.Fatalf("rejection = %+v", prepared.Rejection)
	}
}

func TestPrepareSearchRequestHonorsFiltersPaginationAndSimilarityEligibleAlternatives(t *testing.T) {
	dailyDietID := uuid.MustParse("61e0cae4-0f45-4854-8ac5-b228214cdd1d")
	categoryID := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	prepared, err := PrepareSearchRequest(SearchRequest{
		Query:       "  Chickpea  ",
		Mode:        SearchModeDailyDietAlternative,
		Page:        3,
		DailyDietID: &dailyDietID,
		Filters: []SearchFilter{
			{FilterID: categoryID.String(), Kind: SearchFilterKindFoodCategory, Include: true},
			{FilterID: string(repository.PhysicalStateLiquid), Kind: SearchFilterKindPhysicalState, Include: false},
		},
	}, DailyDietDataAvailable)
	if err != nil {
		t.Fatal(err)
	}
	if prepared.Rejection != nil {
		t.Fatalf("unexpected rejection = %+v", prepared.Rejection)
	}
	if prepared.ParsedQuery.NormalizedText != "chickpea" || prepared.ParsedQuery.Limit != 10 || prepared.ParsedQuery.Offset != 20 {
		t.Fatalf("parsed query = %+v", prepared.ParsedQuery)
	}
	query := prepared.Filters.RepositoryQuery
	if query.Name != "chickpea" || query.Limit != 10 || query.Offset != 20 {
		t.Fatalf("repository query pagination/text = %+v", query)
	}
	if !reflect.DeepEqual(query.FoodCategoryIDs, []uuid.UUID{categoryID}) {
		t.Fatalf("food category filters = %#v", query.FoodCategoryIDs)
	}
	if !reflect.DeepEqual(query.ExcludedFoodObjectTypes, []repository.PhysicalState{repository.PhysicalStateLiquid}) {
		t.Fatalf("excluded object types = %#v", query.ExcludedFoodObjectTypes)
	}

	results, diagnostics, err := CompareMacros(context.Background(), ComparisonRequest{
		SourceMacros: repository.MacroValues{Protein: 20, Carbohydrates: 30, Fat: 10},
		Targets: []TargetMacroVector{
			{ItemID: uuid.MustParse("22222222-2222-4222-8222-222222222222"), Macros: repository.MacroValues{Protein: 20, Carbohydrates: 30, Fat: 10}},
			{ItemID: uuid.MustParse("33333333-3333-4333-8333-333333333333"), Macros: repository.MacroValues{Fat: 10}},
		},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || !nearlyEqual(results[0].Score, 1) {
		t.Fatalf("similarity results = %+v diagnostics = %+v", results, diagnostics)
	}
	if len(diagnostics) != 1 || diagnostics[0].Code != "below_threshold" {
		t.Fatalf("similarity diagnostics = %+v", diagnostics)
	}
}

func TestPrepareSearchRequestUsesDailyDietModeEvenWithSubstitutionInputs(t *testing.T) {
	dailyDietID := uuid.MustParse("61e0cae4-0f45-4854-8ac5-b228214cdd1d")
	prepared, err := PrepareSearchRequest(SearchRequest{
		Query: "tofu",
		Mode:  SearchModeDailyDietAlternative,
		Page:  1,
		SubstitutionInputs: []SubstitutionInput{{
			FoodObjectID: uuid.MustParse("2d4a5f20-c55f-4ba7-9751-779e682f7063"),
			Quantity:     100,
			Unit:         "g",
		}},
		DailyDietID: &dailyDietID,
	}, DailyDietDataUnavailable)
	if err != nil {
		t.Fatal(err)
	}
	if prepared.ParsedQuery.Strategy != SearchStrategyDailyDietAlternative || prepared.Rejection == nil {
		t.Fatalf("prepared daily diet = %+v", prepared)
	}
}

func TestDailyDietValidationErrorAndParserFailure(t *testing.T) {
	if got := ErrDailyDietIDRequired.Error(); got == "" {
		t.Fatal("ErrDailyDietIDRequired.Error() returned an empty message")
	}
	if _, err := PrepareSearchRequest(SearchRequest{Mode: SearchModeCatalog, Page: 1}, DailyDietDataUnavailable); err == nil {
		t.Fatal("PrepareSearchRequest() accepted an empty query")
	}
}
