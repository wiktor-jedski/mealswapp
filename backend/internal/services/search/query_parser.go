package search

import (
	"strings"

	"mealswapp/backend/internal/http/apperrors"
)

type Mode string

const (
	ModeSingle      Mode = "single"
	ModeReplacement Mode = "replacement"
	ModeDiet        Mode = "diet"
)

type IngredientInput struct {
	ItemID   string  `json:"itemId"`
	Name     string  `json:"name"`
	Quantity float64 `json:"quantity"`
	Unit     string  `json:"unit"`
}

type QueryInput struct {
	Mode        Mode              `json:"mode"`
	Query       string            `json:"query"`
	SourceItem  string            `json:"sourceItem"`
	Ingredients []IngredientInput `json:"ingredients"`
}

type ParsedQuery struct {
	Mode               Mode
	Query              string
	SourceItem         string
	Ingredients        []IngredientInput
	ImplicitSimilarity bool
}

func ParseQuery(input QueryInput) (ParsedQuery, error) {
	input.Query = strings.TrimSpace(input.Query)
	input.SourceItem = strings.TrimSpace(input.SourceItem)
	if input.Mode == "" {
		input.Mode = ModeSingle
	}

	switch input.Mode {
	case ModeSingle:
		return parseSingle(input)
	case ModeReplacement:
		return parseReplacement(input)
	case ModeDiet:
		return parseDiet(input)
	default:
		return ParsedQuery{}, validationError("mode", "unsupported")
	}
}

func parseSingle(input QueryInput) (ParsedQuery, error) {
	if input.Query == "" {
		return ParsedQuery{}, validationError("query", "required")
	}
	return ParsedQuery{Mode: ModeSingle, Query: input.Query}, nil
}

func parseReplacement(input QueryInput) (ParsedQuery, error) {
	if input.Query == "" && len(input.Ingredients) >= 2 {
		return ParsedQuery{Mode: ModeReplacement, Ingredients: input.Ingredients, ImplicitSimilarity: true}, nil
	}
	if input.SourceItem == "" {
		return ParsedQuery{}, validationError("sourceItem", "required")
	}
	if input.Query == "" {
		return ParsedQuery{}, validationError("query", "required")
	}
	return ParsedQuery{Mode: ModeReplacement, Query: input.Query, SourceItem: input.SourceItem, Ingredients: input.Ingredients}, nil
}

func parseDiet(input QueryInput) (ParsedQuery, error) {
	if len(input.Ingredients) == 0 {
		return ParsedQuery{}, validationError("ingredients", "required")
	}
	return ParsedQuery{Mode: ModeDiet, Query: input.Query, Ingredients: input.Ingredients}, nil
}

func validationError(field string, code string) error {
	return apperrors.Validation("Search query validation failed", []map[string]string{{"field": field, "code": code}})
}
