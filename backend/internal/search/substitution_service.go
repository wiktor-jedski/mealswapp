package search

import (
	"context"
	"fmt"
	"sort"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// SubstitutionPageSize keeps replacement result pages intentionally compact.
// Implements DESIGN-002 PaginationHandler Substitution Search page size.
const SubstitutionPageSize = 3

// SubstitutionFoodRepository is the repository primitive used by Substitution Search orchestration.
// Implements DESIGN-002 SearchController and CulinaryRoleWeighter.
type SubstitutionFoodRepository interface {
	GetByID(ctx context.Context, id uuid.UUID, rc repository.RepositoryContext) (repository.FoodItemEntity, error)
	Search(ctx context.Context, q repository.RepositoryQuery) ([]repository.FoodItemEntity, int, error)
}

// SubstitutionMealRepository resolves meal inputs without coupling substitution output queries to meal persistence.
// Implements DESIGN-002 SearchController Food Object substitution input resolution.
type SubstitutionMealRepository interface {
	GetByID(ctx context.Context, id uuid.UUID, rc repository.RepositoryContext) (repository.MealEntity, error)
	Search(ctx context.Context, q repository.RepositoryQuery) ([]repository.MealEntity, int, error)
}

// SubstitutionService orchestrates Substitution Search over source macros, filters, similarity, and cache.
// Implements DESIGN-002 SearchController and CulinaryRoleWeighter.
type SubstitutionService struct {
	repository      SubstitutionFoodRepository
	mealRepository  SubstitutionMealRepository
	cache           SearchResponseCache
	similarityCache SimilarityCalculationCache
}

// WithMealRepository enables Meal inputs returned by the shared autocomplete surface.
// Implements DESIGN-002 SearchController Food Object substitution input resolution.
func (s *SubstitutionService) WithMealRepository(meals SubstitutionMealRepository) *SubstitutionService {
	s.mealRepository = meals
	return s
}

// NewSubstitutionService creates Substitution Search orchestration.
// Implements DESIGN-002 SearchController.
func NewSubstitutionService(repository SubstitutionFoodRepository, cache SearchResponseCache, similarityCache ...SimilarityCalculationCache) *SubstitutionService {
	service := &SubstitutionService{repository: repository, cache: cache}
	if len(similarityCache) > 0 {
		service.similarityCache = similarityCache[0]
	}
	return service
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
		} else {
			response.Cache = searchResponseCacheMetadata(s.cache, normalizedReq, CacheStatusMiss)
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
	candidates, err := s.loadCandidateObjects(ctx, processed.RepositoryQuery)
	if err != nil {
		return SearchResponse{}, err
	}
	candidates = excludeSubstitutionSources(candidates, req.SubstitutionInputs)

	calculation, similarityWarnings, err := s.compareMacrosWithCache(ctx, req.SubstitutionInputs, ComparisonRequest{
		SourceMacros:        source.macros,
		SourceCalories:      CalculateCalories(source.macros),
		Targets:             substitutionTargets(candidates),
		MatchType:           MatchTypeCalorie,
		SimilarityThreshold: defaultSimilarityThreshold,
	})
	if err != nil {
		return SearchResponse{}, SimilarityUnavailableError{Cause: err}
	}
	ranked := rankSubstitutionCandidates(candidates, calculation.Results, len(req.SubstitutionInputs) == 1, sourceRoles)
	totalCount := len(ranked.items)
	ranked = paginateRankedSubstitutions(ranked, req.Page)
	warnings := append(catalogWarnings(processed.ExclusionRules), sourceWarnings...)
	warnings = append(warnings, similarityWarnings...)
	for _, diagnostic := range calculation.Diagnostics {
		warnings = append(warnings, "skipped target "+diagnostic.ItemID.String()+" "+diagnostic.Code)
	}

	return SearchResponse{
		Items:              ranked.items,
		ItemTypes:          ranked.itemTypes,
		TotalCount:         totalCount,
		Page:               req.Page,
		SimilarityScores:   ranked.scores,
		SimilarityMetadata: ranked.metadata,
		SourceSummary:      source.summary,
		Warnings:           warnings,
	}, nil
}

// compareMacrosWithCache checks Redis-backed similarity calculations before macro comparison.
// Implements DESIGN-003 CosineSimilarityCalculator and DESIGN-011 RedisCache.
func (s *SubstitutionService) compareMacrosWithCache(ctx context.Context, inputs []SubstitutionInput, req ComparisonRequest) (SimilarityCalculation, []string, error) {
	warnings := []string{}
	if s.similarityCache != nil {
		if cached, hit, err := s.similarityCache.GetSimilarityCalculation(ctx, inputs); err == nil && hit {
			if similarityCalculationCoversTargets(cached, req.Targets) {
				return cached, warnings, nil
			}
		} else if err != nil {
			warnings = append(warnings, WarningCacheUnavailable)
		}
	}

	results, diagnostics, err := CompareMacros(ctx, req, nil)
	if err != nil {
		return SimilarityCalculation{}, warnings, err
	}
	calculation := SimilarityCalculation{Results: results, Diagnostics: diagnostics}
	if s.similarityCache != nil {
		if err := s.similarityCache.SetSimilarityCalculation(ctx, inputs, calculation); err != nil {
			warnings = appendWarningOnce(warnings, WarningCacheUnavailable)
		}
	}
	return calculation, warnings, nil
}

func similarityCalculationCoversTargets(calculation SimilarityCalculation, targets []TargetMacroVector) bool {
	covered := make(map[uuid.UUID]struct{}, len(calculation.Results)+len(calculation.Diagnostics))
	for _, result := range calculation.Results {
		covered[result.ItemID] = struct{}{}
	}
	for _, diagnostic := range calculation.Diagnostics {
		covered[diagnostic.ItemID] = struct{}{}
	}
	for _, target := range targets {
		if _, ok := covered[target.ItemID]; !ok {
			return false
		}
	}
	return true
}

// substitutionSource carries the combined source Macro Profile for Substitution Search.
// Implements DESIGN-002 SearchController.
type substitutionSource struct {
	macros  repository.MacroValues
	summary *SubstitutionSourceSummary
}

type substitutionSourceObject struct {
	physicalState repository.PhysicalState
	macros        repository.MacroValues
	culinaryRoles []repository.ClassificationEntity
}

// combineSourceMacros combines one or more Substitution Inputs into one Macro Profile.
// Implements DESIGN-002 SearchController.
func (s *SubstitutionService) combineSourceMacros(ctx context.Context, inputs []SubstitutionInput) (substitutionSource, []repository.ClassificationEntity, []string, *SearchRejection) {
	total := repository.MacroValues{}
	totalGrams := 0.0
	totalMilliliters := 0.0
	var firstRoles []repository.ClassificationEntity
	warnings := []string{}
	for index, input := range inputs {
		if input.FoodObjectID == uuid.Nil || input.Quantity <= 0 || input.Unit == "" {
			return substitutionSource{}, nil, warnings, &SearchRejection{Code: "rejected_search", Message: "substitution input requires foodObjectId, positive quantity, and unit", Field: "substitutionInputs"}
		}
		food, err := s.loadSourceObject(ctx, input)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("skipped source %s load_failed", input.FoodObjectID))
			continue
		}
		baseQuantity, baseUnit, err := sourceBaseQuantity(input, food.physicalState)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("skipped source %s conversion_failed", input.FoodObjectID))
			continue
		}
		scaled := repository.ScaleMacros(food.macros, baseQuantity, 100)
		total.Protein += scaled.Protein
		total.Carbohydrates += scaled.Carbohydrates
		total.Fat += scaled.Fat
		if baseUnit == "ml" {
			totalMilliliters += baseQuantity
		} else {
			totalGrams += baseQuantity
		}
		if index == 0 {
			firstRoles = food.culinaryRoles
		}
	}
	if _, err := NormalizeMacroVector(total); err != nil {
		return substitutionSource{}, nil, warnings, &SearchRejection{Code: "rejected_search", Message: "substitution inputs do not produce a usable macro profile", Field: "substitutionInputs"}
	}
	return substitutionSource{
		macros: total,
		summary: &SubstitutionSourceSummary{
			Macros:           total,
			Calories:         CalculateCalories(total),
			TotalGrams:       totalGrams,
			TotalMilliliters: totalMilliliters,
		},
	}, firstRoles, warnings, nil
}

// loadSourceObject resolves the explicitly selected Food Object subtype into the common macro surface.
// Implements DESIGN-002 SearchController Food Object substitution input resolution.
func (s *SubstitutionService) loadSourceObject(ctx context.Context, input SubstitutionInput) (substitutionSourceObject, error) {
	if input.FoodObjectType == repository.FoodObjectTypeMeal {
		if s.mealRepository == nil {
			return substitutionSourceObject{}, fmt.Errorf("meal repository unavailable")
		}
		meal, err := s.mealRepository.GetByID(ctx, input.FoodObjectID, repository.RepositoryContext{UnitSystem: repository.UnitSystemMetric})
		if err != nil {
			return substitutionSourceObject{}, err
		}
		roles := make([]repository.ClassificationEntity, 0, len(meal.Classifications))
		for _, classification := range meal.Classifications {
			if classification.Kind == repository.ClassificationKindCulinaryRole {
				roles = append(roles, classification)
			}
		}
		return substitutionSourceObject{physicalState: meal.PhysicalState, macros: meal.MacrosPer100, culinaryRoles: roles}, nil
	}
	food, err := s.repository.GetByID(ctx, input.FoodObjectID, repository.RepositoryContext{UnitSystem: repository.UnitSystemMetric})
	if err != nil {
		return substitutionSourceObject{}, err
	}
	return substitutionSourceObject{physicalState: food.PhysicalState, macros: food.MacrosPer100, culinaryRoles: food.CulinaryRoles}, nil
}

// sourceBaseQuantity converts an input quantity into the food item's macro storage basis.
// Implements DESIGN-002 SearchController.
func sourceBaseQuantity(input SubstitutionInput, physicalState repository.PhysicalState) (float64, string, error) {
	if err := repository.ValidateQuantityUnit(input.Unit); err != nil {
		return 0, "", err
	}
	baseUnit := "g"
	if physicalState == repository.PhysicalStateLiquid {
		baseUnit = "ml"
	}
	quantity, err := repository.ConvertUnit(input.Quantity, input.Unit, baseUnit)
	return quantity, baseUnit, err
}

// excludeSubstitutionSources removes input Food Objects from substitute candidates.
// Implements DESIGN-002 SearchController.
type substitutionObjectCandidate struct {
	item       repository.FoodItemEntity
	objectType repository.FoodObjectType
}

func (s *SubstitutionService) loadCandidateObjects(ctx context.Context, query repository.RepositoryQuery) ([]substitutionObjectCandidate, error) {
	query.Offset = 0
	query.Limit = PageSize
	foods, foodTotal, err := s.repository.Search(ctx, query)
	if err != nil {
		return nil, err
	}
	if foodTotal > len(foods) {
		query.Limit = foodTotal
		foods, _, err = s.repository.Search(ctx, query)
		if err != nil {
			return nil, err
		}
	}
	candidates := make([]substitutionObjectCandidate, 0, len(foods))
	for _, food := range foods {
		candidates = append(candidates, substitutionObjectCandidate{item: food, objectType: repository.FoodObjectTypeFoodItem})
	}
	if s.mealRepository == nil {
		return candidates, nil
	}
	query.Limit = PageSize
	meals, mealTotal, err := s.mealRepository.Search(ctx, query)
	if err != nil {
		return nil, err
	}
	if mealTotal > len(meals) {
		query.Limit = mealTotal
		meals, _, err = s.mealRepository.Search(ctx, query)
		if err != nil {
			return nil, err
		}
	}
	for _, meal := range meals {
		candidates = append(candidates, mealSubstitutionCandidate(meal))
	}
	return candidates, nil
}

func mealSubstitutionCandidate(meal repository.MealEntity) substitutionObjectCandidate {
	item := repository.FoodItemEntity{
		ID: meal.ID, Name: meal.Name, PhysicalState: meal.PhysicalState,
		PrepTimeMinutes: meal.PrepTimeMinutes, AverageUnitWeightGrams: meal.AverageUnitWeightGrams,
		MacrosPer100: meal.MacrosPer100, CreatedAt: meal.CreatedAt, UpdatedAt: meal.UpdatedAt,
	}
	for _, classification := range meal.Classifications {
		switch classification.Kind {
		case repository.ClassificationKindFoodCategory:
			item.FoodCategories = append(item.FoodCategories, classification)
		case repository.ClassificationKindCulinaryRole:
			item.CulinaryRoles = append(item.CulinaryRoles, classification)
		}
	}
	return substitutionObjectCandidate{item: item, objectType: repository.FoodObjectTypeMeal}
}

func excludeSubstitutionSources(items []substitutionObjectCandidate, inputs []SubstitutionInput) []substitutionObjectCandidate {
	sourceIDs := make(map[uuid.UUID]struct{}, len(inputs))
	for _, input := range inputs {
		sourceIDs[input.FoodObjectID] = struct{}{}
	}
	filtered := make([]substitutionObjectCandidate, 0, len(items))
	for _, item := range items {
		if _, isSource := sourceIDs[item.item.ID]; isSource {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

// substitutionTargets adapts repository food items into DESIGN-003 comparison targets.
// Implements DESIGN-003 CosineSimilarityCalculator.
func substitutionTargets(items []substitutionObjectCandidate) []TargetMacroVector {
	targets := make([]TargetMacroVector, 0, len(items))
	for _, item := range items {
		targets = append(targets, TargetMacroVector{
			ItemID:              item.item.ID,
			Macros:              item.item.MacrosPer100,
			CaloriesPerBaseUnit: CalculateCalories(item.item.MacrosPer100) / 100,
			ProteinPerBaseUnit:  item.item.MacrosPer100.Protein / 100,
		})
	}
	return targets
}

// rankedSubstitutionResponse carries response items and their final substitution scores.
// Implements DESIGN-002 SearchController.
type rankedSubstitutionResponse struct {
	items     []repository.FoodItemEntity
	itemTypes []repository.FoodObjectType
	scores    []float64
	metadata  []SimilarityMetadata
}

// substitutionCandidate stores intermediate similarity and final ranking scores.
// Implements DESIGN-002 CulinaryRoleWeighter.
type substitutionCandidate struct {
	item            repository.FoodItemEntity
	objectType      repository.FoodObjectType
	metadata        SimilarityMetadata
	similarityScore float64
	finalScore      float64
}

// rankSubstitutionCandidates sorts accepted targets by score and user-facing name.
// Implements DESIGN-002 CulinaryRoleWeighter.
func rankSubstitutionCandidates(items []substitutionObjectCandidate, results []SimilarityResult, applyRoleWeight bool, sourceRoles []repository.ClassificationEntity) rankedSubstitutionResponse {
	itemByID := make(map[uuid.UUID]substitutionObjectCandidate, len(items))
	for _, item := range items {
		itemByID[item.item.ID] = item
	}
	candidates := make([]substitutionCandidate, 0, len(results))
	for _, result := range results {
		item, ok := itemByID[result.ItemID]
		if !ok {
			continue
		}
		finalScore := result.Score
		if applyRoleWeight {
			finalScore = ApplyCulinaryRoleWeight(result.Score, item.item.CulinaryRoles, sourceRoles)
		}
		candidates = append(candidates, substitutionCandidate{
			item:            item.item,
			objectType:      item.objectType,
			metadata:        similarityMetadataFromResult(result),
			similarityScore: result.Score,
			finalScore:      finalScore,
		})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].finalScore != candidates[j].finalScore {
			return candidates[i].finalScore > candidates[j].finalScore
		}
		return candidates[i].item.Name < candidates[j].item.Name
	})

	response := rankedSubstitutionResponse{
		items:     make([]repository.FoodItemEntity, 0, len(candidates)),
		itemTypes: make([]repository.FoodObjectType, 0, len(candidates)),
		scores:    make([]float64, 0, len(candidates)),
		metadata:  make([]SimilarityMetadata, 0, len(candidates)),
	}
	for _, candidate := range candidates {
		response.items = append(response.items, candidate.item)
		response.itemTypes = append(response.itemTypes, candidate.objectType)
		response.scores = append(response.scores, candidate.finalScore)
		response.metadata = append(response.metadata, candidate.metadata)
	}
	return response
}

func paginateRankedSubstitutions(response rankedSubstitutionResponse, page int) rankedSubstitutionResponse {
	offset := (max(page, 1) - 1) * SubstitutionPageSize
	if offset >= len(response.items) {
		return rankedSubstitutionResponse{items: []repository.FoodItemEntity{}, itemTypes: []repository.FoodObjectType{}, scores: []float64{}, metadata: []SimilarityMetadata{}}
	}
	end := min(offset+SubstitutionPageSize, len(response.items))
	response.items = response.items[offset:end]
	response.itemTypes = response.itemTypes[offset:end]
	response.scores = response.scores[offset:end]
	response.metadata = response.metadata[offset:end]
	return response
}

// similarityMetadataFromResult preserves DESIGN-003 metadata for the ordered response path.
// Implements DESIGN-003 SimilarityIndicatorMapper.
func similarityMetadataFromResult(result SimilarityResult) SimilarityMetadata {
	return SimilarityMetadata{
		ItemID:           result.ItemID,
		Score:            result.Score,
		Tier:             result.Tier,
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

// CalculateCalories derives calories from protein, carbohydrate, and fat grams.
// Implements DESIGN-003 CosineSimilarityCalculator.
func CalculateCalories(macros repository.MacroValues) float64 {
	return macros.Protein*4 + macros.Carbohydrates*4 + macros.Fat*9
}
