package search

import (
	"slices"
	"strconv"
	"strings"

	"mealswapp/backend/internal/domain/tag"
	"mealswapp/backend/internal/http/apperrors"
	"mealswapp/backend/internal/repositories"

	"github.com/google/uuid"
)

var allowedFilterMacroKeys = []string{"protein", "carbs", "fat"}

type TagFilterKind string

const (
	TagFilterKindDiet          TagFilterKind = "diet"
	TagFilterKindAllergen      TagFilterKind = "allergen"
	TagFilterKindFunctionality TagFilterKind = "functionality"
	TagFilterKindCuration      TagFilterKind = "curation"
)

type TagFilter struct {
	TagID   string        `json:"tagId"`
	Kind    TagFilterKind `json:"kind"`
	Include bool          `json:"include"`
}

type FilterInput struct {
	TagFilters       []TagFilter     `json:"filters"`
	EnabledMacros    map[string]bool `json:"enabledMacros"`
	DietaryTagIDs    []string        `json:"dietaryTagIds"`
	AllergenTagIDs   []string        `json:"allergenTagIds"`
	SourceProviders  []string        `json:"sourceProviders"`
	IncludeFoodIDs   []uuid.UUID
	Limit            int
	Offset           int
	NormalizedSearch string
}

type RepositoryQuery struct {
	repositories.FoodItemQuery
	EnabledMacros   map[string]bool
	SourceProviders []string
}

func ApplyFilters(input FilterInput) (RepositoryQuery, error) {
	var fields []map[string]string
	includeTagIDs := make([]uuid.UUID, 0)
	excludeTagIDs := make([]uuid.UUID, 0)

	for i, filter := range input.TagFilters {
		tagID, err := parseFilterUUID(filter.TagID)
		if err != nil {
			fields = append(fields, map[string]string{"field": indexedField("filters", i, "tagId"), "code": "invalid"})
		}
		if !filter.Kind.Valid() {
			fields = append(fields, map[string]string{"field": indexedField("filters", i, "kind"), "code": "unsupported"})
			continue
		}
		if err != nil {
			continue
		}
		if filter.Include {
			includeTagIDs = appendUniqueUUID(includeTagIDs, tagID)
		} else {
			excludeTagIDs = appendUniqueUUID(excludeTagIDs, tagID)
		}
	}

	for i, rawID := range input.DietaryTagIDs {
		tagID, err := parseFilterUUID(rawID)
		if err != nil {
			fields = append(fields, map[string]string{"field": indexedField("dietaryTagIds", i, ""), "code": "invalid"})
			continue
		}
		includeTagIDs = appendUniqueUUID(includeTagIDs, tagID)
	}

	for i, rawID := range input.AllergenTagIDs {
		tagID, err := parseFilterUUID(rawID)
		if err != nil {
			fields = append(fields, map[string]string{"field": indexedField("allergenTagIds", i, ""), "code": "invalid"})
			continue
		}
		excludeTagIDs = appendUniqueUUID(excludeTagIDs, tagID)
	}

	enabledMacros := normalizeEnabledMacros(input.EnabledMacros, &fields)
	sourceProviders := normalizeSourceProviders(input.SourceProviders, &fields)
	if len(fields) > 0 {
		return RepositoryQuery{}, apperrors.Validation("Search filter validation failed", fields)
	}

	return RepositoryQuery{
		FoodItemQuery: repositories.FoodItemQuery{
			Text:          strings.TrimSpace(input.NormalizedSearch),
			IncludeTagIDs: includeTagIDs,
			ExcludeTagIDs: excludeTagIDs,
			Limit:         input.Limit,
			Offset:        input.Offset,
		},
		EnabledMacros:   enabledMacros,
		SourceProviders: sourceProviders,
	}, nil
}

func (kind TagFilterKind) Valid() bool {
	switch tag.Kind(kind) {
	case tag.KindDiet, tag.KindAllergen, tag.KindFunctionality, tag.KindCuration:
		return true
	default:
		return false
	}
}

func parseFilterUUID(value string) (uuid.UUID, error) {
	return uuid.Parse(strings.TrimSpace(value))
}

func normalizeEnabledMacros(input map[string]bool, fields *[]map[string]string) map[string]bool {
	if input == nil {
		return map[string]bool{"protein": true, "carbs": true, "fat": true}
	}

	enabled := make(map[string]bool, len(allowedFilterMacroKeys))
	for _, key := range allowedFilterMacroKeys {
		value, ok := input[key]
		if !ok {
			*fields = append(*fields, map[string]string{"field": "enabledMacros." + key, "code": "required"})
			continue
		}
		enabled[key] = value
	}
	for key := range input {
		if !slices.Contains(allowedFilterMacroKeys, key) {
			*fields = append(*fields, map[string]string{"field": "enabledMacros." + key, "code": "unsupported"})
		}
	}
	if len(enabled) == len(allowedFilterMacroKeys) && !enabled["protein"] && !enabled["carbs"] && !enabled["fat"] {
		*fields = append(*fields, map[string]string{"field": "enabledMacros", "code": "at_least_one_required"})
	}
	return enabled
}

func normalizeSourceProviders(input []string, fields *[]map[string]string) []string {
	providers := make([]string, 0, len(input))
	for i, provider := range input {
		normalized := strings.ToLower(strings.TrimSpace(provider))
		if normalized == "" {
			*fields = append(*fields, map[string]string{"field": indexedField("sourceProviders", i, ""), "code": "required"})
			continue
		}
		providers = appendUniqueString(providers, normalized)
	}
	return providers
}

func appendUniqueUUID(values []uuid.UUID, value uuid.UUID) []uuid.UUID {
	if slices.Contains(values, value) {
		return values
	}
	return append(values, value)
}

func appendUniqueString(values []string, value string) []string {
	if slices.Contains(values, value) {
		return values
	}
	return append(values, value)
}

func indexedField(collection string, index int, field string) string {
	if field == "" {
		return collection + "." + strconv.Itoa(index)
	}
	return collection + "." + strconv.Itoa(index) + "." + field
}
