package search

import (
	"context"
	"fmt"
	"sort"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// SubstitutionFoodRepository is the repository primitive used by Substitution Search orchestration.
// Implements DESIGN-002 SearchController and CulinaryRoleWeighter.
type SubstitutionFoodRepository interface {
	GetByID(ctx context.Context, id uuid.UUID, rc repository.RepositoryContext) (repository.FoodItemEntity, error)
	Search(ctx context.Context, q repository.RepositoryQuery) ([]repository.FoodItemEntity, int, error)
}

// SubstitutionService orchestrates Substitution Search over source macros, filters, similarity, and cache.
// Implements DESIGN-002 SearchController and CulinaryRoleWeighter.
type SubstitutionService struct {
	repository SubstitutionFoodRepository
	cache      SearchResponseCache
}

// NewSubstitutionService creates Substitution Search orchestration.
// Implements DESIGN-002 SearchController.
func NewSubstitutionService(repository SubstitutionFoodRepository, cache SearchResponseCache) *SubstitutionService {
	return &SubstitutionService{repository: repository, cache: cache}
}

// Search executes Substitution Search and returns ranked food substitutes.
// Implements DESIGN-002 SearchController and CulinaryRoleWeighter.
func (s *SubstitutionService) Search(ctx context.Context, req SearchRequest) (SearchResponse, error) {
	parsed, err := BuildParsedQuery(req)
	if err != nil {
		return SearchResponse{}, err
	}
	if parsed.Strategy != SearchStrategySubstitution {
		return SearchResponse{Items: []repository.FoodItemEntity{}, TotalCount: 0, Page: normalizedPage(req.Page), SimilarityScores: []float64{}, Warnings: []string{}, Rejection: &SearchRejection{Code: "rejected_search", Message: "search mode is not available for substitution results", Field: "mode"}}, nil
	}
	normalizedReq := req
	normalizedReq.Query = parsed.NormalizedText
	normalizedReq.Page = normalizedPage(req.Page)

	cacheWarnings := []string{}
	if s.cache != nil {
		if cached, hit, err := s.cache.GetSearchResponse(ctx, normalizedReq); err == nil && hit {
			return cached, nil
		} else if err != nil {
			cacheWarnings = append(cacheWarnings, WarningCacheUnavailable)
		}
	}

	response, err := s.loadSubstitutions(ctx, parsed, normalizedReq)
	if err != nil {
		return SearchResponse{}, err
	}
	response.Warnings = append(response.Warnings, cacheWarnings...)
	if response.Rejection == nil && s.cache != nil {
		if err := s.cache.SetSearchResponse(ctx, normalizedReq, responseWithoutCache(response)); err != nil {
			response.Warnings = appendWarningOnce(response.Warnings, WarningCacheUnavailable)
		}
	}
	return response, nil
}

// loadSubstitutions performs uncached substitution filtering, similarity ranking, and warning assembly.
// Implements DESIGN-002 SearchController and CulinaryRoleWeighter.
func (s *SubstitutionService) loadSubstitutions(ctx context.Context, parsed ParsedQuery, req SearchRequest) (SearchResponse, error) {
	if len(req.SubstitutionInputs) == 0 {
		return SearchResponse{Items: []repository.FoodItemEntity{}, TotalCount: 0, Page: req.Page, SimilarityScores: []float64{}, Warnings: []string{}, Rejection: &SearchRejection{Code: "rejected_search", Message: "at least one substitution input is required", Field: "substitutionInputs"}}, nil
	}
	processed, rejection := ApplyFilters(parsed, req.Filters)
	if rejection != nil {
		return SearchResponse{Items: []repository.FoodItemEntity{}, TotalCount: 0, Page: req.Page, SimilarityScores: []float64{}, Warnings: []string{}, Rejection: rejection}, nil
	}

	source, sourceRoles, sourceWarnings, rejection := s.combineSourceMacros(ctx, req.SubstitutionInputs)
	if rejection != nil {
		return SearchResponse{Items: []repository.FoodItemEntity{}, TotalCount: 0, Page: req.Page, SimilarityScores: []float64{}, Warnings: sourceWarnings, Rejection: rejection}, nil
	}
	candidates, _, err := s.repository.Search(ctx, processed.RepositoryQuery)
	if err != nil {
		return SearchResponse{}, err
	}

	results, diagnostics, err := CompareMacros(ctx, ComparisonRequest{
		SourceMacros:        source.macros,
		SourceCalories:      macroCalories(source.macros),
		Targets:             substitutionTargets(candidates),
		MatchType:           MatchTypeCalorie,
		SimilarityThreshold: defaultSimilarityThreshold,
	}, nil)
	if err != nil {
		return SearchResponse{}, SimilarityUnavailableError{Cause: err}
	}
	ranked := rankSubstitutionCandidates(candidates, results, len(req.SubstitutionInputs) == 1, sourceRoles)
	warnings := append(catalogWarnings(processed.ExclusionRules), sourceWarnings...)
	for _, diagnostic := range diagnostics {
		warnings = append(warnings, "skipped target "+diagnostic.ItemID.String()+" "+diagnostic.Code)
	}

	return SearchResponse{
		Items:              ranked.items,
		TotalCount:         len(ranked.items),
		Page:               req.Page,
		SimilarityScores:   ranked.scores,
		SimilarityMetadata: ranked.metadata,
		Warnings:           warnings,
	}, nil
}

// substitutionSource carries the combined source Macro Profile for Substitution Search.
// Implements DESIGN-002 SearchController.
type substitutionSource struct {
	macros repository.MacroValues
}

// combineSourceMacros combines one or more Substitution Inputs into one Macro Profile.
// Implements DESIGN-002 SearchController.
func (s *SubstitutionService) combineSourceMacros(ctx context.Context, inputs []SubstitutionInput) (substitutionSource, []repository.ClassificationEntity, []string, *SearchRejection) {
	total := repository.MacroValues{}
	var firstRoles []repository.ClassificationEntity
	warnings := []string{}
	for index, input := range inputs {
		if input.FoodObjectID == uuid.Nil || input.Quantity <= 0 || input.Unit == "" {
			return substitutionSource{}, nil, warnings, &SearchRejection{Code: "rejected_search", Message: "substitution input requires foodObjectId, positive quantity, and unit", Field: "substitutionInputs"}
		}
		food, err := s.repository.GetByID(ctx, input.FoodObjectID, repository.RepositoryContext{UnitSystem: repository.UnitSystemMetric})
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("skipped source %s load_failed", input.FoodObjectID))
			continue
		}
		baseQuantity, _, err := sourceBaseQuantity(input, food)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("skipped source %s conversion_failed", input.FoodObjectID))
			continue
		}
		scaled := repository.ScaleMacros(food.MacrosPer100, baseQuantity, 100)
		total.Protein += scaled.Protein
		total.Carbohydrates += scaled.Carbohydrates
		total.Fat += scaled.Fat
		if index == 0 {
			firstRoles = food.CulinaryRoles
		}
	}
	if _, err := NormalizeMacroVector(total); err != nil {
		return substitutionSource{}, nil, warnings, &SearchRejection{Code: "rejected_search", Message: "substitution inputs do not produce a usable macro profile", Field: "substitutionInputs"}
	}
	return substitutionSource{macros: total}, firstRoles, warnings, nil
}

// sourceBaseQuantity converts an input quantity into the food item's macro storage basis.
// Implements DESIGN-002 SearchController.
func sourceBaseQuantity(input SubstitutionInput, food repository.FoodItemEntity) (float64, string, error) {
	unit := canonicalQuantityUnit(input.Unit)
	baseUnit := "g"
	if food.PhysicalState == repository.PhysicalStateLiquid {
		baseUnit = "ml"
	}
	if unit == "serving" {
		return repository.ConvertServingToBase(input.Quantity, food.AverageUnitWeightGrams, food.AverageServingVolumeMilliliters, food.PhysicalState)
	}
	quantity, err := repository.ConvertUnit(input.Quantity, unit, baseUnit)
	return quantity, baseUnit, err
}

// canonicalQuantityUnit maps API-friendly unit aliases to repository units.
// Implements DESIGN-002 QueryParser.
func canonicalQuantityUnit(unit string) string {
	switch unit {
	case "gram", "grams":
		return "g"
	case "milliliter", "milliliters":
		return "ml"
	case "ounce", "ounces":
		return "oz"
	case "fluid_ounce", "fluid_ounces":
		return "fl_oz"
	default:
		return unit
	}
}

// substitutionTargets adapts repository food items into DESIGN-003 comparison targets.
// Implements DESIGN-003 CosineSimilarityCalculator.
func substitutionTargets(items []repository.FoodItemEntity) []TargetMacroVector {
	targets := make([]TargetMacroVector, 0, len(items))
	for _, item := range items {
		targets = append(targets, TargetMacroVector{
			ItemID:              item.ID,
			Macros:              item.MacrosPer100,
			CaloriesPerBaseUnit: macroCalories(item.MacrosPer100) / 100,
			ProteinPerBaseUnit:  item.MacrosPer100.Protein / 100,
		})
	}
	return targets
}

// rankedSubstitutionResponse carries response items and their final substitution scores.
// Implements DESIGN-002 SearchController.
type rankedSubstitutionResponse struct {
	items    []repository.FoodItemEntity
	scores   []float64
	metadata []SimilarityMetadata
}

// substitutionCandidate stores intermediate similarity and final ranking scores.
// Implements DESIGN-002 CulinaryRoleWeighter.
type substitutionCandidate struct {
	item            repository.FoodItemEntity
	metadata        SimilarityMetadata
	similarityScore float64
	finalScore      float64
}

// rankSubstitutionCandidates sorts accepted targets by final substitution score.
// Implements DESIGN-002 CulinaryRoleWeighter.
func rankSubstitutionCandidates(items []repository.FoodItemEntity, results []SimilarityResult, applyRoleWeight bool, sourceRoles []repository.ClassificationEntity) rankedSubstitutionResponse {
	itemByID := make(map[uuid.UUID]repository.FoodItemEntity, len(items))
	for _, item := range items {
		itemByID[item.ID] = item
	}
	candidates := make([]substitutionCandidate, 0, len(results))
	for _, result := range results {
		item, ok := itemByID[result.ItemID]
		if !ok {
			continue
		}
		finalScore := result.Score
		if applyRoleWeight {
			finalScore = ApplyCulinaryRoleWeight(result.Score, item.CulinaryRoles, sourceRoles)
		}
		candidates = append(candidates, substitutionCandidate{
			item:            item,
			metadata:        similarityMetadataFromResult(result),
			similarityScore: result.Score,
			finalScore:      finalScore,
		})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].finalScore != candidates[j].finalScore {
			return candidates[i].finalScore > candidates[j].finalScore
		}
		if candidates[i].item.Name != candidates[j].item.Name {
			return candidates[i].item.Name < candidates[j].item.Name
		}
		return candidates[i].item.ID.String() < candidates[j].item.ID.String()
	})

	response := rankedSubstitutionResponse{
		items:    make([]repository.FoodItemEntity, 0, len(candidates)),
		scores:   make([]float64, 0, len(candidates)),
		metadata: make([]SimilarityMetadata, 0, len(candidates)),
	}
	for _, candidate := range candidates {
		response.items = append(response.items, candidate.item)
		response.scores = append(response.scores, candidate.finalScore)
		response.metadata = append(response.metadata, candidate.metadata)
	}
	return response
}

// similarityMetadataFromResult preserves DESIGN-003 metadata for the ordered response path.
// Implements DESIGN-003 SimilarityIndicatorMapper.
func similarityMetadataFromResult(result SimilarityResult) SimilarityMetadata {
	return SimilarityMetadata{
		ItemID:           result.ItemID,
		Score:            result.Score,
		Tier:             result.Tier,
		ColorHex:         result.ColorHex,
		ImageURL:         result.ImageURL,
		MatchingQuantity: result.MatchingQuantity,
	}
}

// ApplyCulinaryRoleWeight boosts a single-input substitution candidate for shared culinary roles.
// Implements DESIGN-002 CulinaryRoleWeighter.
func ApplyCulinaryRoleWeight(similarityScore float64, candidateRoles []repository.ClassificationEntity, sourceRoles []repository.ClassificationEntity) float64 {
	return similarityScore * (1 + 0.2*float64(culinaryRoleMatchCount(candidateRoles, sourceRoles)))
}

// culinaryRoleMatchCount counts unique shared Culinary Role classifications.
// Implements DESIGN-002 CulinaryRoleWeighter.
func culinaryRoleMatchCount(candidateRoles []repository.ClassificationEntity, sourceRoles []repository.ClassificationEntity) int {
	sourceIDs := make(map[uuid.UUID]struct{}, len(sourceRoles))
	for _, role := range sourceRoles {
		sourceIDs[role.ID] = struct{}{}
	}
	count := 0
	seen := map[uuid.UUID]struct{}{}
	for _, role := range candidateRoles {
		if _, duplicate := seen[role.ID]; duplicate {
			continue
		}
		seen[role.ID] = struct{}{}
		if _, ok := sourceIDs[role.ID]; ok {
			count++
		}
	}
	return count
}

// macroCalories derives calories from protein, carbohydrate, and fat grams.
// Implements DESIGN-003 CosineSimilarityCalculator.
func macroCalories(macros repository.MacroValues) float64 {
	return macros.Protein*4 + macros.Carbohydrates*4 + macros.Fat*9
}
