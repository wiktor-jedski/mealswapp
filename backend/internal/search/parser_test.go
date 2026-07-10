package search

import (
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// Implements DESIGN-002 QueryParser verification.

func TestSearchContractDTOsCarryDesignFields(t *testing.T) {
	foodObjectID := uuid.MustParse("2d4a5f20-c55f-4ba7-9751-779e682f7063")
	dailyDietID := uuid.MustParse("61e0cae4-0f45-4854-8ac5-b228214cdd1d")

	req := SearchRequest{
		Query: "tomato",
		Mode:  SearchModeCatalog,
		Filters: []SearchFilter{{
			FilterID: "vegetable",
			Kind:     SearchFilterKindFoodCategory,
			Include:  true,
		}},
		Page: 2,
		SubstitutionInputs: []SubstitutionInput{{
			FoodObjectID: foodObjectID,
			Quantity:     12.5,
			Unit:         "g",
		}},
		DailyDietID: &dailyDietID,
	}
	if req.Filters[0].FilterID != "vegetable" || req.Filters[0].Kind != SearchFilterKindFoodCategory || !req.Filters[0].Include {
		t.Fatalf("filter DTO = %+v", req.Filters[0])
	}
	if req.SubstitutionInputs[0].FoodObjectID != foodObjectID || req.SubstitutionInputs[0].Quantity != 12.5 || req.SubstitutionInputs[0].Unit != "g" {
		t.Fatalf("substitution input DTO = %+v", req.SubstitutionInputs[0])
	}
	if req.DailyDietID == nil || *req.DailyDietID != dailyDietID {
		t.Fatalf("daily diet id DTO = %v", req.DailyDietID)
	}

	response := SearchResponse{
		Items:            []repository.FoodItemEntity{{ID: foodObjectID, Name: "Tomato"}},
		TotalCount:       1,
		Page:             2,
		SimilarityScores: []float64{0.98},
		Warnings:         []string{"ranking_timeout"},
		Rejection: &SearchRejection{
			Code:    "rejected_search",
			Message: "conflicting filters",
			Field:   "filters",
		},
	}
	if response.Items[0].ID != foodObjectID || response.TotalCount != 1 || response.Page != 2 {
		t.Fatalf("search response core fields = %+v", response)
	}
	if !reflect.DeepEqual(response.SimilarityScores, []float64{0.98}) || !reflect.DeepEqual(response.Warnings, []string{"ranking_timeout"}) {
		t.Fatalf("search response scoring/warnings = %+v", response)
	}
	if response.Rejection == nil || response.Rejection.Code != "rejected_search" || response.Rejection.Field != "filters" {
		t.Fatalf("search rejection DTO = %+v", response.Rejection)
	}
}

func TestBuildParsedQueryNormalizesTokensAndPagination(t *testing.T) {
	parsed, err := BuildParsedQuery(SearchRequest{
		Query: "  Fresh   TOMATO  Soup ",
		Mode:  SearchModeCatalog,
		Page:  3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if parsed.NormalizedText != "fresh tomato soup" {
		t.Fatalf("normalized text = %q", parsed.NormalizedText)
	}
	if !reflect.DeepEqual(parsed.Tokens, []string{"fresh", "tomato", "soup"}) {
		t.Fatalf("tokens = %#v", parsed.Tokens)
	}
	if parsed.Strategy != SearchStrategyCatalog {
		t.Fatalf("strategy = %q", parsed.Strategy)
	}
	if parsed.Limit != PageSize || parsed.Offset != 20 {
		t.Fatalf("pagination = limit %d offset %d", parsed.Limit, parsed.Offset)
	}
}

func TestBuildParsedQueryAllowsEmptySubstitutionQuery(t *testing.T) {
	parsed, err := BuildParsedQuery(SearchRequest{
		Query: "",
		Mode:  SearchModeSubstitution,
		Page:  2,
		SubstitutionInputs: []SubstitutionInput{{
			FoodObjectID: uuid.MustParse("2d4a5f20-c55f-4ba7-9751-779e682f7063"),
			Quantity:     100,
			Unit:         "ml",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if parsed.NormalizedText != "" || len(parsed.Tokens) != 0 || parsed.Strategy != SearchStrategySubstitution || parsed.Offset != PageSize {
		t.Fatalf("parsed query = %+v", parsed)
	}
}

func TestPaginateClampsPageSizeToTenAndCalculatesOffset(t *testing.T) {
	limit, offset := Paginate(4, 50)
	if limit != 10 || offset != 30 {
		t.Fatalf("large page size pagination = limit %d offset %d", limit, offset)
	}
	limit, offset = Paginate(2, 5)
	if limit != 10 || offset != 10 {
		t.Fatalf("small page size pagination = limit %d offset %d", limit, offset)
	}
	limit, offset = Paginate(0, 0)
	if limit != 10 || offset != 0 {
		t.Fatalf("defensive pagination = limit %d offset %d", limit, offset)
	}
}

func TestSelectStrategyFromMode(t *testing.T) {
	dailyDietID := uuid.MustParse("61e0cae4-0f45-4854-8ac5-b228214cdd1d")
	foodObjectID := uuid.MustParse("2d4a5f20-c55f-4ba7-9751-779e682f7063")
	substitutionInputs := []SubstitutionInput{{
		FoodObjectID: foodObjectID,
		Quantity:     1,
		Unit:         "g",
	}}

	for name, req := range map[string]SearchRequest{
		"catalog mode": {
			Query: "tomato",
			Mode:  SearchModeCatalog,
			Page:  1,
		},
		"substitution mode": {
			Query: "tomato",
			Mode:  SearchModeSubstitution,
			Page:  1,
		},
		"daily diet mode": {
			Query: "tomato",
			Mode:  SearchModeDailyDiet,
			Page:  1,
		},
		"daily diet alternative mode": {
			Query: "tomato",
			Mode:  SearchModeDailyDietAlternative,
			Page:  1,
		},
		"catalog mode with daily diet id remains catalog": {
			Query:       "tomato",
			Mode:        SearchModeCatalog,
			Page:        1,
			DailyDietID: &dailyDietID,
		},
		"catalog mode with substitution input remains catalog": {
			Query:              "tomato",
			Mode:               SearchModeCatalog,
			Page:               1,
			SubstitutionInputs: substitutionInputs,
		},
		"daily diet mode with substitution inputs remains daily diet": {
			Query: "tomato",
			Mode:  SearchModeDailyDietAlternative,
			Page:  1,
			SubstitutionInputs: []SubstitutionInput{
				substitutionInputs[0],
				{FoodObjectID: uuid.MustParse("90d5ff43-3451-444d-88c9-0af96b6938f9"), Quantity: 2, Unit: "g"},
			},
			DailyDietID: &dailyDietID,
		},
	} {
		parsed, err := BuildParsedQuery(req)
		if err != nil {
			t.Fatalf("%s parse error = %v", name, err)
		}
		switch name {
		case "catalog mode", "catalog mode with daily diet id remains catalog", "catalog mode with substitution input remains catalog":
			if parsed.Strategy != SearchStrategyCatalog {
				t.Fatalf("%s strategy = %q", name, parsed.Strategy)
			}
		case "substitution mode":
			if parsed.Strategy != SearchStrategySubstitution {
				t.Fatalf("%s strategy = %q", name, parsed.Strategy)
			}
		case "daily diet mode":
			if parsed.Strategy != SearchStrategyDailyDiet {
				t.Fatalf("%s strategy = %q", name, parsed.Strategy)
			}
		case "daily diet alternative mode", "daily diet mode with substitution inputs remains daily diet":
			if parsed.Strategy != SearchStrategyDailyDietAlternative {
				t.Fatalf("%s strategy = %q", name, parsed.Strategy)
			}
		}
	}
}

func TestBuildParsedQueryRejectsInvalidRequestFields(t *testing.T) {
	for name, req := range map[string]SearchRequest{
		"empty query":  {Query: " ", Mode: SearchModeCatalog, Page: 1},
		"invalid page": {Query: "tomato", Mode: SearchModeCatalog, Page: 0},
		"invalid mode": {Query: "tomato", Mode: SearchMode("meal_plan"), Page: 1},
	} {
		if _, err := BuildParsedQuery(req); err == nil {
			t.Fatalf("%s accepted", name)
		}
	}
}
