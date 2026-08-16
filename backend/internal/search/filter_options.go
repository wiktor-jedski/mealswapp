package search

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// FilterOptionReference identifies a backend policy dependency without a display label.
// Implements DESIGN-009 TagManager filter-option policy projection.
type FilterOptionReference struct {
	FilterID string
	Kind     SearchFilterKind
}

// FilterOption is one localized-label-ready search filter choice.
// Implements DESIGN-009 TagManager filter-option service.
type FilterOption struct {
	FilterID       string
	Kind           SearchFilterKind
	Label          string
	LabelKey       string
	IncludeAllowed bool
	ExcludeAllowed bool
	Excludes       []FilterOptionReference
}

// FilterOptionsResponse is the deterministic option inventory for one supported mode.
// Implements DESIGN-009 TagManager filter-option service.
type FilterOptionsResponse struct {
	Mode    SearchMode
	Options []FilterOption
}

// FilterOptionClassificationRepository is the narrow active-classification read boundary.
// Implements DESIGN-009 TagManager filter-option service.
type FilterOptionClassificationRepository interface {
	List(context.Context, repository.ClassificationKind) ([]repository.ClassificationEntity, error)
}

// FilterOptionAllergenRepository is the narrow active-allergen read boundary.
// Implements DESIGN-009 TagManager filter-option service.
type FilterOptionAllergenRepository interface {
	ListActive(context.Context) ([]repository.AllergenVocabularyEntry, error)
}

// FilterOptionGenerationSource exposes shared classification invalidation state.
// Implements DESIGN-009 TagManager cross-instance cache invalidation.
type FilterOptionGenerationSource interface {
	Current(context.Context) (uint64, error)
}

// FilterOptionService owns backend filter policy and caches persisted vocabulary projections.
// Implements DESIGN-009 TagManager filter-option service.
type FilterOptionService struct {
	classifications  FilterOptionClassificationRepository
	allergens        FilterOptionAllergenRepository
	sharedGeneration FilterOptionGenerationSource
	mu               sync.RWMutex
	generation       uint64
	cachedGeneration uint64
	cached           *FilterOptionsResponse
}

// NewFilterOptionService creates the backend-owned filter-option service.
// Implements DESIGN-009 TagManager filter-option service.
func NewFilterOptionService(classifications FilterOptionClassificationRepository, allergens FilterOptionAllergenRepository) *FilterOptionService {
	return &FilterOptionService{classifications: classifications, allergens: allergens}
}

// NewVersionedFilterOptionService creates a filter service invalidated across API instances.
// Implements DESIGN-009 TagManager cross-instance cache invalidation.
func NewVersionedFilterOptionService(classifications FilterOptionClassificationRepository, allergens FilterOptionAllergenRepository, generation FilterOptionGenerationSource) *FilterOptionService {
	return &FilterOptionService{classifications: classifications, allergens: allergens, sharedGeneration: generation}
}

// Options validates mode and returns a copy of the deterministic option inventory.
// Implements DESIGN-009 TagManager filter-option service.
func (s *FilterOptionService) Options(ctx context.Context, mode SearchMode) (FilterOptionsResponse, error) {
	if mode != SearchModeSubstitution {
		return FilterOptionsResponse{}, repository.NewError(repository.ErrorKindValidation, "filter option mode is unsupported", nil)
	}
	generation, cacheable := s.cacheGeneration(ctx)
	if cacheable {
		if cached, ok := s.cachedOptions(generation); ok {
			return cached, nil
		}
	}

	response, err := s.load(ctx, mode)
	if err != nil {
		return FilterOptionsResponse{}, err
	}
	currentGeneration, stillCacheable := s.cacheGeneration(ctx)
	if !cacheable || !stillCacheable || currentGeneration != generation {
		return cloneFilterOptionsResponse(response), nil
	}

	s.mu.Lock()
	if s.sharedGeneration != nil || s.generation == generation {
		copy := cloneFilterOptionsResponse(response)
		s.cached = &copy
		s.cachedGeneration = generation
	}
	s.mu.Unlock()
	return cloneFilterOptionsResponse(response), nil
}

// cacheGeneration returns the shared generation or the local fallback generation.
// Implements DESIGN-009 TagManager cache invalidation.
func (s *FilterOptionService) cacheGeneration(ctx context.Context) (uint64, bool) {
	if s.sharedGeneration != nil {
		generation, err := s.sharedGeneration.Current(ctx)
		return generation, err == nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.generation, true
}

// Invalidate discards cached vocabulary after an administrative classification change.
// Implements DESIGN-009 TagManager filter-option cache invalidation seam.
func (s *FilterOptionService) Invalidate() {
	s.mu.Lock()
	s.generation++
	s.cached = nil
	s.mu.Unlock()
}

// cachedOptions returns an isolated cache copy.
// Implements DESIGN-009 TagManager filter-option service.
func (s *FilterOptionService) cachedOptions(generation uint64) (FilterOptionsResponse, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cached == nil || s.cachedGeneration != generation {
		return FilterOptionsResponse{}, false
	}
	return cloneFilterOptionsResponse(*s.cached), true
}

// load reads active vocabularies and projects backend policy.
// Implements DESIGN-009 TagManager filter-option service.
func (s *FilterOptionService) load(ctx context.Context, mode SearchMode) (FilterOptionsResponse, error) {
	foodCategories, err := s.classifications.List(ctx, repository.ClassificationKindFoodCategory)
	if err != nil {
		return FilterOptionsResponse{}, err
	}
	culinaryRoles, err := s.classifications.List(ctx, repository.ClassificationKindCulinaryRole)
	if err != nil {
		return FilterOptionsResponse{}, err
	}
	allergens, err := s.allergens.ListActive(ctx)
	if err != nil {
		return FilterOptionsResponse{}, err
	}

	options := physicalStateFilterOptions()
	options = append(options, classificationFilterOptions(foodCategories)...)
	options = append(options, classificationFilterOptions(culinaryRoles)...)
	for _, allergen := range allergens {
		options = append(options, FilterOption{FilterID: allergen.Key, Kind: SearchFilterKindAllergen, Label: allergen.Name, LabelKey: allergen.LabelKey, ExcludeAllowed: true})
	}
	options = append(options, dietaryPresetFilterOptions()...)
	sortFilterOptions(options)
	return FilterOptionsResponse{Mode: mode, Options: options}, nil
}

// classificationFilterOptions maps persisted identities without inventing route IDs.
// Implements DESIGN-009 TagManager filter-option service.
func classificationFilterOptions(entries []repository.ClassificationEntity) []FilterOption {
	options := make([]FilterOption, 0, len(entries))
	for _, entry := range entries {
		options = append(options, FilterOption{FilterID: entry.ID.String(), Kind: SearchFilterKind(entry.Kind), Label: entry.Name, IncludeAllowed: true, ExcludeAllowed: true})
	}
	return options
}

// physicalStateFilterOptions projects repository physical-state policy.
// Implements DESIGN-009 TagManager filter-option service.
func physicalStateFilterOptions() []FilterOption {
	return []FilterOption{
		{FilterID: string(repository.PhysicalStateLiquid), Kind: SearchFilterKindPhysicalState, Label: "Liquids", LabelKey: "filter.physical_state.liquid", IncludeAllowed: true, ExcludeAllowed: true},
		{FilterID: string(repository.PhysicalStateSolid), Kind: SearchFilterKindPhysicalState, Label: "Solid foods", LabelKey: "filter.physical_state.solid", IncludeAllowed: true, ExcludeAllowed: true},
	}
}

// dietaryPresetFilterOptions projects the same exclusion rules accepted by ApplyFilters.
// Implements DESIGN-009 TagManager and DESIGN-002 FilterProcessor.
func dietaryPresetFilterOptions() []FilterOption {
	presets := []struct {
		id       DietaryPreset
		label    string
		labelKey string
	}{
		{DietaryPresetDairyFree, "Dairy-free", "filter.dietary_preset.dairy_free"},
		{DietaryPresetGlutenFree, "Gluten-free", "filter.dietary_preset.gluten_free"},
		{DietaryPresetNutFree, "Nut-free", "filter.dietary_preset.nut_free"},
		{DietaryPresetVegan, "Vegan", "filter.dietary_preset.vegan"},
		{DietaryPresetVegetarian, "Vegetarian", "filter.dietary_preset.vegetarian"},
	}
	options := make([]FilterOption, 0, len(presets))
	for _, preset := range presets {
		rules := dietaryPresetExclusionRules[preset.id]
		excludes := make([]FilterOptionReference, 0, len(rules))
		for _, rule := range rules {
			excludes = append(excludes, FilterOptionReference{FilterID: rule.FilterID, Kind: rule.Kind})
		}
		options = append(options, FilterOption{FilterID: string(preset.id), Kind: SearchFilterKindDietaryPreset, Label: preset.label, LabelKey: preset.labelKey, ExcludeAllowed: true, Excludes: excludes})
	}
	return options
}

// sortFilterOptions orders groups and labels independently of repository order.
// Implements DESIGN-009 TagManager deterministic filter-option ordering.
func sortFilterOptions(options []FilterOption) {
	kindOrder := map[SearchFilterKind]int{
		SearchFilterKindPhysicalState: 0,
		SearchFilterKindFoodCategory:  1,
		SearchFilterKindCulinaryRole:  2,
		SearchFilterKindAllergen:      3,
		SearchFilterKindDietaryPreset: 4,
	}
	sort.SliceStable(options, func(i, j int) bool {
		if kindOrder[options[i].Kind] != kindOrder[options[j].Kind] {
			return kindOrder[options[i].Kind] < kindOrder[options[j].Kind]
		}
		left, right := strings.ToLower(options[i].Label), strings.ToLower(options[j].Label)
		if left != right {
			return left < right
		}
		return options[i].FilterID < options[j].FilterID
	})
}

// cloneFilterOptionsResponse prevents callers from mutating service cache state.
// Implements DESIGN-009 TagManager filter-option service.
func cloneFilterOptionsResponse(response FilterOptionsResponse) FilterOptionsResponse {
	clone := FilterOptionsResponse{Mode: response.Mode, Options: make([]FilterOption, len(response.Options))}
	for i, option := range response.Options {
		clone.Options[i] = option
		clone.Options[i].Excludes = append([]FilterOptionReference(nil), option.Excludes...)
	}
	return clone
}
