package search

import (
	"fmt"
	"slices"
	"sort"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// DietaryPreset identifies backend-owned dietary bundles.
// Implements DESIGN-002 FilterProcessor.
type DietaryPreset string

// Implements DESIGN-002 FilterProcessor supported dietary presets.
const (
	DietaryPresetDairyFree  DietaryPreset = "dairy_free"
	DietaryPresetGlutenFree DietaryPreset = "gluten_free"
	DietaryPresetNutFree    DietaryPreset = "nut_free"
	DietaryPresetVegan      DietaryPreset = "vegan"
	DietaryPresetVegetarian DietaryPreset = "vegetarian"
)

// ExclusionRule is the expanded form of exclude filters and Dietary Presets.
// Implements DESIGN-002 FilterProcessor.
type ExclusionRule struct {
	FilterID string
	Kind     SearchFilterKind
	Source   string
}

// ProcessedFilters carries repository filters and expanded Exclusion Rules.
// Implements DESIGN-002 FilterProcessor.
type ProcessedFilters struct {
	RepositoryQuery repository.RepositoryQuery
	ExclusionRules  []ExclusionRule
}

// Implements DESIGN-002 FilterProcessor dietary preset expansion rules.
var dietaryPresetExclusionRules = map[DietaryPreset][]ExclusionRule{
	DietaryPresetDairyFree: {
		{FilterID: "dairy", Kind: SearchFilterKindAllergen, Source: string(DietaryPresetDairyFree)},
	},
	DietaryPresetGlutenFree: {
		{FilterID: "gluten", Kind: SearchFilterKindAllergen, Source: string(DietaryPresetGlutenFree)},
	},
	DietaryPresetNutFree: {
		{FilterID: "peanut", Kind: SearchFilterKindAllergen, Source: string(DietaryPresetNutFree)},
		{FilterID: "tree_nut", Kind: SearchFilterKindAllergen, Source: string(DietaryPresetNutFree)},
	},
	DietaryPresetVegan: {
		{FilterID: "animal_product", Kind: SearchFilterKindAllergen, Source: string(DietaryPresetVegan)},
		{FilterID: "dairy", Kind: SearchFilterKindAllergen, Source: string(DietaryPresetVegan)},
		{FilterID: "egg", Kind: SearchFilterKindAllergen, Source: string(DietaryPresetVegan)},
	},
	DietaryPresetVegetarian: {
		{FilterID: "meat", Kind: SearchFilterKindAllergen, Source: string(DietaryPresetVegetarian)},
	},
}

// Implements DESIGN-002 FilterProcessor supported allergen keys.
var supportedAllergenKeys = map[string]struct{}{
	"animal_product": {},
	"dairy":          {},
	"egg":            {},
	"gluten":         {},
	"meat":           {},
	"peanut":         {},
	"tree_nut":       {},
}

// ApplyFilters validates filters, expands Dietary Presets, and builds a repository query.
// Implements DESIGN-002 FilterProcessor.
func ApplyFilters(query ParsedQuery, filters []SearchFilter) (ProcessedFilters, *SearchRejection) {
	processed := ProcessedFilters{
		RepositoryQuery: repository.RepositoryQuery{
			Name:   query.NormalizedText,
			Limit:  query.Limit,
			Offset: query.Offset,
		},
	}
	includes := map[filterKey]struct{}{}
	exclusions := map[filterKey]string{}

	for _, filter := range filters {
		if filter.FilterID == "" {
			return ProcessedFilters{}, rejectedFilter("filter id is required", "filters")
		}
		if filter.Kind == SearchFilterKindDietaryPreset {
			if filter.Include {
				return ProcessedFilters{}, rejectedFilter("dietary presets can only be used as exclusion bundles", "filters")
			}
			preset := DietaryPreset(filter.FilterID)
			rules, ok := dietaryPresetExclusionRules[preset]
			if !ok {
				return ProcessedFilters{}, rejectedFilter("dietary preset is unsupported", "filters")
			}
			for _, rule := range rules {
				if rejection := addExclusionRule(&processed, exclusions, includes, rule); rejection != nil {
					return ProcessedFilters{}, rejection
				}
			}
			continue
		}

		key := filterKey{kind: filter.Kind, id: filter.FilterID}
		if filter.Include {
			if source, excluded := exclusions[key]; excluded {
				return ProcessedFilters{}, rejectedFilter(fmt.Sprintf("filter %q conflicts with exclusion rule from %s", filter.FilterID, source), "filters")
			}
			includes[key] = struct{}{}
			if rejection := addRepositoryFilter(&processed.RepositoryQuery, filter, true); rejection != nil {
				return ProcessedFilters{}, rejection
			}
			continue
		}

		rule := ExclusionRule{FilterID: filter.FilterID, Kind: filter.Kind, Source: "filter"}
		if rejection := addExclusionRule(&processed, exclusions, includes, rule); rejection != nil {
			return ProcessedFilters{}, rejection
		}
	}
	sortExclusionRules(processed.ExclusionRules)
	return processed, nil
}

// filterKey identifies a single include or exclusion constraint.
// Implements DESIGN-002 FilterProcessor.
type filterKey struct {
	kind SearchFilterKind
	id   string
}

// addExclusionRule records one exclusion and rejects include/exclude conflicts.
// Implements DESIGN-002 FilterProcessor.
func addExclusionRule(processed *ProcessedFilters, exclusions map[filterKey]string, includes map[filterKey]struct{}, rule ExclusionRule) *SearchRejection {
	key := filterKey{kind: rule.Kind, id: rule.FilterID}
	if _, included := includes[key]; included {
		return rejectedFilter(fmt.Sprintf("exclusion rule %q conflicts with an included filter", rule.FilterID), "filters")
	}
	if _, exists := exclusions[key]; exists {
		return nil
	}
	exclusions[key] = rule.Source
	processed.ExclusionRules = append(processed.ExclusionRules, rule)
	return addRepositoryFilter(&processed.RepositoryQuery, SearchFilter{FilterID: rule.FilterID, Kind: rule.Kind}, false)
}

// addRepositoryFilter maps API filters to repository query fields.
// Implements DESIGN-002 FilterProcessor.
func addRepositoryFilter(query *repository.RepositoryQuery, filter SearchFilter, include bool) *SearchRejection {
	switch filter.Kind {
	case SearchFilterKindFoodCategory:
		return addUUIDFilter(&query.FoodCategoryIDs, &query.ExcludedFoodCategoryIDs, filter.FilterID, include)
	case SearchFilterKindCulinaryRole:
		return addUUIDFilter(&query.CulinaryRoleIDs, &query.ExcludedCulinaryRoleIDs, filter.FilterID, include)
	case SearchFilterKindAllergen:
		id, err := uuid.Parse(filter.FilterID)
		if err == nil {
			if include {
				query.AllergenIDs = appendUniqueUUID(query.AllergenIDs, id)
			} else {
				query.ExcludedAllergenIDs = appendUniqueUUID(query.ExcludedAllergenIDs, id)
			}
			return nil
		}
		if _, ok := supportedAllergenKeys[filter.FilterID]; !ok {
			return rejectedFilter("allergen filter id must be a UUID or supported allergen key", "filters")
		}
		if include {
			query.AllergenKeys = appendUniqueString(query.AllergenKeys, filter.FilterID)
		} else {
			query.ExcludedAllergenKeys = appendUniqueString(query.ExcludedAllergenKeys, filter.FilterID)
		}
		return nil
	case SearchFilterKindPhysicalState:
		state, ok := physicalStateFromFilterID(filter.FilterID)
		if !ok {
			return rejectedFilter("physical state is unsupported", "filters")
		}
		if include {
			query.FoodObjectTypes = appendUniquePhysicalState(query.FoodObjectTypes, state)
		} else {
			query.ExcludedFoodObjectTypes = appendUniquePhysicalState(query.ExcludedFoodObjectTypes, state)
		}
		return nil
	default:
		return rejectedFilter("filter kind is unsupported", "filters")
	}
}

// addUUIDFilter appends a classification UUID to include or exclude filters.
// Implements DESIGN-002 FilterProcessor.
func addUUIDFilter(includes *[]uuid.UUID, excludes *[]uuid.UUID, filterID string, include bool) *SearchRejection {
	id, err := uuid.Parse(filterID)
	if err != nil {
		return rejectedFilter("classification filter id must be a UUID", "filters")
	}
	if include {
		*includes = appendUniqueUUID(*includes, id)
	} else {
		*excludes = appendUniqueUUID(*excludes, id)
	}
	return nil
}

// physicalStateFromFilterID maps food-object filter IDs to repository states.
// Implements DESIGN-002 FilterProcessor.
func physicalStateFromFilterID(filterID string) (repository.PhysicalState, bool) {
	switch filterID {
	case string(repository.PhysicalStateSolid):
		return repository.PhysicalStateSolid, true
	case string(repository.PhysicalStateLiquid):
		return repository.PhysicalStateLiquid, true
	default:
		return "", false
	}
}

// appendUniqueUUID appends UUID values without duplicates.
// Implements DESIGN-002 FilterProcessor.
func appendUniqueUUID(values []uuid.UUID, value uuid.UUID) []uuid.UUID {
	if slices.Contains(values, value) {
		return values
	}
	return append(values, value)
}

// appendUniquePhysicalState appends physical states without duplicates.
// Implements DESIGN-002 FilterProcessor.
func appendUniquePhysicalState(values []repository.PhysicalState, value repository.PhysicalState) []repository.PhysicalState {
	if slices.Contains(values, value) {
		return values
	}
	return append(values, value)
}

// appendUniqueString appends string values without duplicates.
// Implements DESIGN-002 FilterProcessor.
func appendUniqueString(values []string, value string) []string {
	if slices.Contains(values, value) {
		return values
	}
	return append(values, value)
}

// sortExclusionRules orders exclusion warnings deterministically.
// Implements DESIGN-002 FilterProcessor.
func sortExclusionRules(rules []ExclusionRule) {
	sort.SliceStable(rules, func(i, j int) bool {
		if rules[i].Kind == rules[j].Kind {
			return rules[i].FilterID < rules[j].FilterID
		}
		return rules[i].Kind < rules[j].Kind
	})
}

// rejectedFilter builds a consistent rejected-search response.
// Implements DESIGN-002 FilterProcessor.
func rejectedFilter(message string, field string) *SearchRejection {
	return &SearchRejection{Code: "rejected_search", Message: message, Field: field}
}
