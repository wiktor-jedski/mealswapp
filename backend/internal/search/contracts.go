// Package search defines backend search contracts and query parsing.
package search

import (
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// SearchMode identifies the user-facing search operation.
// Implements DESIGN-002 QueryParser.
type SearchMode string

// Implements DESIGN-002 QueryParser supported search modes.
// Implements DESIGN-017 ErrorMessageMapper degraded feature warnings.
const (
	SearchModeCatalog              SearchMode = "catalog"
	SearchModeSubstitution         SearchMode = "substitution"
	SearchModeDailyDiet            SearchMode = "daily_diet"
	SearchModeDailyDietAlternative SearchMode = "daily_diet_alternative"
)

// SearchStrategy identifies the resolved backend search strategy.
// Implements DESIGN-002 QueryParser.
type SearchStrategy string

// Implements DESIGN-002 QueryParser supported backend strategies.
const (
	SearchStrategyCatalog              SearchStrategy = "catalog"
	SearchStrategySubstitution         SearchStrategy = "substitution"
	SearchStrategyDailyDiet            SearchStrategy = "daily_diet"
	SearchStrategyDailyDietAlternative SearchStrategy = "daily_diet_alternative"
)

// SearchFilterKind identifies supported include/exclude filter groups.
// Implements DESIGN-002 FilterProcessor.
type SearchFilterKind string

// Implements DESIGN-002 FilterProcessor supported filter kinds.
const (
	SearchFilterKindFoodCategory  SearchFilterKind = "food_category"
	SearchFilterKindCulinaryRole  SearchFilterKind = "culinary_role"
	SearchFilterKindPhysicalState SearchFilterKind = "physical_state"
	SearchFilterKindAllergen      SearchFilterKind = "allergen"
	SearchFilterKindDietaryPreset SearchFilterKind = "dietary_preset"
)

// SearchFilter carries one include/exclude search constraint.
// Implements DESIGN-002 FilterProcessor.
type SearchFilter struct {
	FilterID string
	Kind     SearchFilterKind
	Include  bool
}

// SubstitutionInput carries one source food quantity for Substitution Search.
// Implements DESIGN-002 QueryParser.
type SubstitutionInput struct {
	FoodObjectID   uuid.UUID
	FoodObjectType repository.FoodObjectType
	Quantity       float64
	Unit           string
}

// SearchRequest carries normalized backend search input.
// Implements DESIGN-002 QueryParser.
type SearchRequest struct {
	Query              string
	Mode               SearchMode
	Filters            []SearchFilter
	Page               int
	SubstitutionInputs []SubstitutionInput
	DailyDietID        *uuid.UUID
}

// SearchResponse carries deterministic paged internal service/cache output.
// It is not the public HTTP response DTO.
// Implements DESIGN-002 QueryParser.
type SearchResponse struct {
	Items              []repository.FoodItemEntity
	ItemTypes          []repository.FoodObjectType
	TotalCount         int
	Page               int
	SimilarityScores   []float64
	SimilarityMetadata []SimilarityMetadata
	SourceSummary      *SubstitutionSourceSummary
	Warnings           []string
	Rejection          *SearchRejection
	Cache              *CacheMetadata `json:"cache,omitempty"`
}

// SubstitutionSourceSummary reports the user's selected input list after quantity scaling.
// It keeps mass and volume separate because density is item-specific and may be unavailable.
// Implements DESIGN-002 SearchController.
type SubstitutionSourceSummary struct {
	Macros           repository.MacroValues
	Calories         float64
	TotalGrams       float64
	TotalMilliliters float64
}

// WarningCacheUnavailable reports cache read/write degradation while preserving catalog fallback.
// Implements DESIGN-017 ErrorMessageMapper degraded feature metadata.
const WarningCacheUnavailable = "cache_unavailable"

// SimilarityMetadata carries ordered substitution display and replacement metadata.
// Implements DESIGN-003 SimilarityIndicatorMapper and DESIGN-002 SearchController.
type SimilarityMetadata struct {
	ItemID           uuid.UUID
	Score            float64
	Tier             SimilarityTier
	ImageURL         string
	MatchingQuantity float64
}

// AutocompleteResponse carries ranked autocomplete output and cache metadata.
// Implements DESIGN-011 RedisCache response metadata for autocomplete.
type AutocompleteResponse struct {
	Items []RankedAutocomplete
	Cache *CacheMetadata `json:",omitempty"`
}

// CacheStatus identifies whether a response came from Redis.
// Implements DESIGN-011 RedisCache cache-hit and cache-miss metadata.
type CacheStatus string

// Implements DESIGN-011 RedisCache cache metadata states.
const (
	CacheStatusHit  CacheStatus = "hit"
	CacheStatusMiss CacheStatus = "miss"
)

// CacheMetadata reports server-side Redis cache behavior without exposing raw inputs.
// Implements DESIGN-011 RedisCache response metadata.
type CacheMetadata struct {
	Status        CacheStatus
	Namespace     string
	SchemaVersion string
	TTLSeconds    int64
}

// SearchRejection reports validly shaped but unsatisfiable search constraints.
// Implements DESIGN-002 QueryParser.
type SearchRejection struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

// DailyDietDataStatus reports whether Phase 07 saved-diet data can seed alternatives.
// Implements DESIGN-002 QueryParser daily diet alternative boundary.
type DailyDietDataStatus string

// Implements DESIGN-002 QueryParser daily diet alternative boundary states.
const (
	DailyDietDataAvailable   DailyDietDataStatus = "available"
	DailyDietDataUnavailable DailyDietDataStatus = "unavailable"
)

// ParsedQuery carries normalized query text and deterministic pagination.
// Implements DESIGN-002 QueryParser.
type ParsedQuery struct {
	NormalizedText string
	Tokens         []string
	Strategy       SearchStrategy
	Limit          int
	Offset         int
}
