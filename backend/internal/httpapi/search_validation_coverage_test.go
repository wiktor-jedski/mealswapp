package httpapi

// Implements DESIGN-010 RequestValidator defensive-path verification.

import (
	"math"
	"testing"

	"github.com/wiktor-jedski/mealswapp/backend/internal/search"
)

func TestValidateSearchRequestBodyRejectsMalformedFieldTypes(t *testing.T) {
	valid := func() map[string]any {
		return map[string]any{"query": "apple", "mode": "catalog", "page": float64(1)}
	}
	cases := map[string]func(map[string]any){
		"query type": func(body map[string]any) { body["query"] = 1 },
		"mode type":  func(body map[string]any) { body["mode"] = true },
		"daily type": func(body map[string]any) { body["dailyDietId"] = 1 },
	}
	for name, mutate := range cases {
		body := valid()
		mutate(body)
		if err := ValidateSearchRequestBody(body); err == nil {
			t.Fatalf("%s accepted", name)
		}
	}
}

func TestSearchRequestFromValidatedDTORejectsInvalidShapes(t *testing.T) {
	page := 1
	include := true
	quantity := 1.0
	input := validatedSubstitutionInputDTO{FoodObjectID: "2d4a5f20-c55f-4ba7-9751-779e682f7063", Quantity: &quantity, Unit: "g"}
	dailyID := "61e0cae4-0f45-4854-8ac5-b228214cdd1d"
	cases := map[string]validatedSearchRequestBodyDTO{
		"blank query":        {Mode: "catalog", Page: &page},
		"blank mode":         {Query: "apple", Page: &page},
		"missing page":       {Query: "apple", Mode: "catalog"},
		"catalog inputs":     {Query: "apple", Mode: "catalog", Page: &page, SubstitutionInputs: []validatedSubstitutionInputDTO{input}},
		"substitution empty": {Query: "apple", Mode: "substitution", Page: &page},
		"substitution diet":  {Query: "apple", Mode: "substitution", Page: &page, SubstitutionInputs: []validatedSubstitutionInputDTO{input}, DailyDietID: &dailyID},
		"diet inputs":        {Query: "apple", Mode: "daily_diet_alternative", Page: &page, SubstitutionInputs: []validatedSubstitutionInputDTO{input}, DailyDietID: &dailyID},
		"diet missing id":    {Query: "apple", Mode: "daily_diet_alternative", Page: &page},
	}
	badDailyID := "invalid"
	cases["invalid daily id"] = validatedSearchRequestBodyDTO{Query: "apple", Mode: "daily_diet_alternative", Page: &page, DailyDietID: &badDailyID}
	cases["bad filter"] = validatedSearchRequestBodyDTO{Query: "apple", Mode: "catalog", Page: &page, Filters: []validatedSearchFilterDTO{{FilterID: "x", Kind: "allergen", Include: nil}}}
	cases["bad input"] = validatedSearchRequestBodyDTO{Query: "apple", Mode: "substitution", Page: &page, SubstitutionInputs: []validatedSubstitutionInputDTO{{FoodObjectID: "bad", Quantity: &quantity, Unit: "g"}}}
	_ = include
	for name, dto := range cases {
		if _, err := searchRequestFromValidatedDTO(dto); err == nil {
			t.Fatalf("%s accepted: %+v", name, dto)
		}
	}
}

func TestDecodeValidatedSearchRequestBodyErrors(t *testing.T) {
	if _, err := decodeValidatedSearchRequestBody(map[string]any{"query": make(chan int)}); err == nil {
		t.Fatal("unsupported JSON value accepted")
	}
	if _, err := decodeValidatedSearchRequestBody(map[string]any{"page": map[string]any{"bad": true}}); err == nil {
		t.Fatal("mistyped DTO value accepted")
	}
}

func TestValidateSearchFiltersRejectsMalformedItems(t *testing.T) {
	cases := map[string]any{
		"not array":       "bad",
		"not object":      []any{"bad"},
		"missing id":      []any{map[string]any{"kind": "allergen", "include": true}},
		"missing kind":    []any{map[string]any{"filterId": "dairy", "include": true}},
		"invalid kind":    []any{map[string]any{"filterId": "dairy", "kind": "bad", "include": true}},
		"missing include": []any{map[string]any{"filterId": "dairy", "kind": "allergen"}},
	}
	for name, value := range cases {
		if err := validateSearchFilters(value); err == nil {
			t.Fatalf("%s accepted", name)
		}
	}
}

func TestValidateSubstitutionInputsRejectsMalformedItems(t *testing.T) {
	validID := "2d4a5f20-c55f-4ba7-9751-779e682f7063"
	cases := map[string]any{
		"not array":        "bad",
		"not object":       []any{"bad"},
		"id type":          []any{map[string]any{"foodObjectId": 1, "quantity": 1.0, "unit": "g"}},
		"invalid id":       []any{map[string]any{"foodObjectId": "bad", "quantity": 1.0, "unit": "g"}},
		"quantity type":    []any{map[string]any{"foodObjectId": validID, "quantity": "1", "unit": "g"}},
		"invalid quantity": []any{map[string]any{"foodObjectId": validID, "quantity": math.NaN(), "unit": "g"}},
		"unit type":        []any{map[string]any{"foodObjectId": validID, "quantity": 1.0, "unit": 1}},
		"unsupported unit": []any{map[string]any{"foodObjectId": validID, "quantity": 1.0, "unit": "serving"}},
	}
	for name, value := range cases {
		if err := validateSubstitutionInputs(value); err == nil {
			t.Fatalf("%s accepted", name)
		}
	}
}

func TestParseValidatedDTOCollectionsRejectMalformedValues(t *testing.T) {
	include := true
	for name, filters := range map[string][]validatedSearchFilterDTO{
		"missing id":      {{Kind: "allergen", Include: &include}},
		"missing kind":    {{FilterID: "dairy", Include: &include}},
		"missing include": {{FilterID: "dairy", Kind: "allergen"}},
	} {
		if _, err := parseValidatedSearchFilterDTOs(filters); err == nil {
			t.Fatalf("filter %s accepted", name)
		}
	}
	quantity := 1.0
	for name, inputs := range map[string][]validatedSubstitutionInputDTO{
		"invalid id":       {{FoodObjectID: "bad", Quantity: &quantity, Unit: "g"}},
		"missing quantity": {{FoodObjectID: "2d4a5f20-c55f-4ba7-9751-779e682f7063", Unit: "g"}},
		"missing unit":     {{FoodObjectID: "2d4a5f20-c55f-4ba7-9751-779e682f7063", Quantity: &quantity}},
		"invalid unit":     {{FoodObjectID: "2d4a5f20-c55f-4ba7-9751-779e682f7063", Quantity: &quantity, Unit: "serving"}},
	} {
		if _, err := parseValidatedSubstitutionInputDTOs(inputs); err == nil {
			t.Fatalf("input %s accepted", name)
		}
	}
}

func TestValidationHelpersAcceptAlternateRuntimeShapes(t *testing.T) {
	if err := validateSearchPageValue(1); err != nil {
		t.Fatalf("integer page rejected: %v", err)
	}
	if err := validateSearchFilters(nil); err != nil {
		t.Fatalf("nil filters rejected: %v", err)
	}
	if err := validateSubstitutionInputs(nil); err != nil {
		t.Fatalf("nil inputs rejected: %v", err)
	}
	if got := substitutionInputCount([]search.SubstitutionInput{{}}); got != 0 {
		t.Fatalf("typed substitution input count = %d", got)
	}
}
