package search

import (
	"testing"

	"mealswapp/backend/internal/http/apperrors"
)

func TestParseSingleQuery(t *testing.T) {
	parsed, err := ParseQuery(QueryInput{Mode: ModeSingle, Query: " tofu "})
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Mode != ModeSingle || parsed.Query != "tofu" {
		t.Fatalf("unexpected parsed query: %#v", parsed)
	}
}

func TestParseReplacementQuery(t *testing.T) {
	parsed, err := ParseQuery(QueryInput{Mode: ModeReplacement, SourceItem: "butter", Query: "olive oil"})
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Mode != ModeReplacement || parsed.SourceItem != "butter" || parsed.Query != "olive oil" {
		t.Fatalf("unexpected replacement query: %#v", parsed)
	}
}

func TestParseDietQuery(t *testing.T) {
	parsed, err := ParseQuery(QueryInput{Mode: ModeDiet, Ingredients: []IngredientInput{{Name: "tofu", Quantity: 100, Unit: "gram"}}})
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Mode != ModeDiet || len(parsed.Ingredients) != 1 {
		t.Fatalf("unexpected diet query: %#v", parsed)
	}
}

func TestParseImplicitSimilarityForEmptyQueryWithIngredients(t *testing.T) {
	parsed, err := ParseQuery(QueryInput{Mode: ModeReplacement, Ingredients: []IngredientInput{{Name: "tofu"}, {Name: "lentils"}}})
	if err != nil {
		t.Fatal(err)
	}
	if !parsed.ImplicitSimilarity || parsed.Mode != ModeReplacement {
		t.Fatalf("expected implicit similarity, got %#v", parsed)
	}
}

func TestParseQueryValidationFailures(t *testing.T) {
	cases := []QueryInput{
		{Mode: ModeSingle},
		{Mode: ModeReplacement, Query: "olive oil"},
		{Mode: ModeDiet},
		{Mode: "bad", Query: "tofu"},
	}

	for _, input := range cases {
		_, err := ParseQuery(input)
		appErr, ok := apperrors.As(err)
		if !ok || appErr.Code != "validation_error" {
			t.Fatalf("expected validation error for %#v, got %v", input, err)
		}
	}
}
