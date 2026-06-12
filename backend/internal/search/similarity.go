package search

import (
	"context"
	"errors"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// Implements DESIGN-003 CosineSimilarityCalculator default threshold.
const defaultSimilarityThreshold = 0.40

// Implements DESIGN-003 SimilarityIndicatorMapper asset URL namespace.
const similarityAssetURLPrefix = "/assets/similarity"

// Implements DESIGN-003 SimilarityIndicatorMapper tier rules.
var similarityTierRules = []TierRule{
	{Tier: SimilarityTierExcellent, MinScore: 0.85, MaxScore: 1, ColorHex: "#1B7F4C", ImageURL: similarityAssetURLPrefix + "/excellent.svg"},
	{Tier: SimilarityTierGood, MinScore: 0.70, MaxScore: 0.85, ColorHex: "#2F80ED", ImageURL: similarityAssetURLPrefix + "/good.svg"},
	{Tier: SimilarityTierFair, MinScore: 0.55, MaxScore: 0.70, ColorHex: "#B7791F", ImageURL: similarityAssetURLPrefix + "/fair.svg"},
	{Tier: SimilarityTierPoor, MinScore: 0, MaxScore: 0.55, ColorHex: "#A23B3B", ImageURL: similarityAssetURLPrefix + "/poor.svg"},
}

// MatchType selects the replacement quantity denominator.
// Implements DESIGN-003 CosineSimilarityCalculator.
type MatchType string

// Implements DESIGN-003 CosineSimilarityCalculator supported quantity modes.
const (
	MatchTypeCalorie MatchType = "calorie"
	MatchTypeProtein MatchType = "protein"
)

// SimilarityTier identifies the visual match tier for a similarity score.
// Implements DESIGN-003 SimilarityIndicatorMapper.
type SimilarityTier string

// Implements DESIGN-003 SimilarityIndicatorMapper score tiers.
const (
	SimilarityTierExcellent SimilarityTier = "excellent"
	SimilarityTierGood      SimilarityTier = "good"
	SimilarityTierFair      SimilarityTier = "fair"
	SimilarityTierPoor      SimilarityTier = "poor"
)

// NormalizedMacroVector stores a unit vector and its original magnitude.
// Implements DESIGN-003 MacroVectorNormalizer.
type NormalizedMacroVector struct {
	Protein       float64
	Carbohydrates float64
	Fat           float64
	Magnitude     float64
}

// TargetMacroVector carries one direct food or aggregate recipe comparison input.
// Implements DESIGN-003 CosineSimilarityCalculator and DESIGN-005 RepositoryInterfaces.
type TargetMacroVector struct {
	ItemID              uuid.UUID
	RecipeMealID        *uuid.UUID
	Macros              repository.MacroValues
	CaloriesPerBaseUnit float64
	ProteinPerBaseUnit  float64
}

// ComparisonRequest carries a source vector and candidate targets for comparison.
// Implements DESIGN-003 CosineSimilarityCalculator.
type ComparisonRequest struct {
	SourceMacros        repository.MacroValues
	SourceCalories      float64
	Targets             []TargetMacroVector
	MatchType           MatchType
	SimilarityThreshold float64
}

// SimilarityResult carries one accepted target and replacement metadata.
// Implements DESIGN-003 CosineSimilarityCalculator.
type SimilarityResult struct {
	ItemID           uuid.UUID
	Score            float64
	Tier             SimilarityTier
	MatchingQuantity float64
	ColorHex         string
	ImageURL         string
}

// SimilarityDiagnostic reports skipped targets that are not hard request failures.
// Implements DESIGN-003 ThresholdFilter.
type SimilarityDiagnostic struct {
	ItemID uuid.UUID
	Code   string
}

// TierRule maps a score range to display metadata.
// Implements DESIGN-003 SimilarityIndicatorMapper.
type TierRule struct {
	Tier     SimilarityTier
	MinScore float64
	MaxScore float64
	ColorHex string
	ImageURL string
}

// MacroAggregator loads recipe aggregate macro inputs before similarity ranking.
// Implements DESIGN-003 CosineSimilarityCalculator and DESIGN-005 RepositoryInterfaces.
type MacroAggregator interface {
	CalculateMacros(ctx context.Context, mealID uuid.UUID) (repository.MacroValues, error)
}

// NormalizeMacroVector validates and converts macro values to a unit vector.
// Implements DESIGN-003 MacroVectorNormalizer.
func NormalizeMacroVector(v repository.MacroValues) (NormalizedMacroVector, error) {
	if err := validateSimilarityMacros(v); err != nil {
		return NormalizedMacroVector{}, err
	}
	magnitude := math.Sqrt(v.Protein*v.Protein + v.Carbohydrates*v.Carbohydrates + v.Fat*v.Fat)
	if magnitude == 0 {
		return NormalizedMacroVector{}, errors.New("zero macro vector cannot be normalized")
	}
	return NormalizedMacroVector{
		Protein:       v.Protein / magnitude,
		Carbohydrates: v.Carbohydrates / magnitude,
		Fat:           v.Fat / magnitude,
		Magnitude:     magnitude,
	}, nil
}

// CosineSimilarity computes the dot product between normalized macro vectors.
// Implements DESIGN-003 CosineSimilarityCalculator.
func CosineSimilarity(a NormalizedMacroVector, b NormalizedMacroVector) float64 {
	return a.Protein*b.Protein + a.Carbohydrates*b.Carbohydrates + a.Fat*b.Fat
}

// CompareMacros ranks targets by macro-vector similarity and returns skip diagnostics.
// Implements DESIGN-003 CosineSimilarityCalculator.
func CompareMacros(ctx context.Context, req ComparisonRequest, aggregator MacroAggregator) ([]SimilarityResult, []SimilarityDiagnostic, error) {
	source, err := NormalizeMacroVector(req.SourceMacros)
	if err != nil {
		return nil, nil, err
	}
	threshold := req.SimilarityThreshold
	if threshold == 0 {
		threshold = defaultSimilarityThreshold
	}

	results := make([]SimilarityResult, 0, len(req.Targets))
	diagnostics := []SimilarityDiagnostic{}
	for _, target := range req.Targets {
		macros := target.Macros
		if target.RecipeMealID != nil {
			if aggregator == nil {
				return nil, diagnostics, errors.New("macro aggregator is required for recipe targets")
			}
			macros, err = aggregator.CalculateMacros(ctx, *target.RecipeMealID)
			if err != nil {
				return nil, diagnostics, err
			}
		}
		normalizedTarget, err := NormalizeMacroVector(macros)
		if err != nil {
			if isZeroMacroVector(macros) {
				diagnostics = append(diagnostics, SimilarityDiagnostic{ItemID: target.ItemID, Code: "zero_target_vector"})
				continue
			}
			return nil, diagnostics, err
		}
		score := CosineSimilarity(source, normalizedTarget)
		if score < threshold {
			diagnostics = append(diagnostics, SimilarityDiagnostic{ItemID: target.ItemID, Code: "below_threshold"})
			continue
		}
		rule := MapSimilarityTier(score)
		results = append(results, SimilarityResult{
			ItemID:           target.ItemID,
			Score:            score,
			Tier:             rule.Tier,
			MatchingQuantity: CalculateMatchingQuantity(req.SourceMacros, req.SourceCalories, target, req.MatchType),
			ColorHex:         rule.ColorHex,
			ImageURL:         rule.ImageURL,
		})
	}

	sort.SliceStable(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	return results, diagnostics, nil
}

// FilterByThreshold keeps results at or above the provided minimum score.
// Implements DESIGN-003 ThresholdFilter.
func FilterByThreshold(results []SimilarityResult, minScore float64) []SimilarityResult {
	filtered := results[:0]
	for _, result := range results {
		if result.Score >= minScore {
			filtered = append(filtered, result)
		}
	}
	return filtered
}

// MapSimilarityTier maps a similarity score to display metadata.
// Implements DESIGN-003 SimilarityIndicatorMapper.
func MapSimilarityTier(score float64) TierRule {
	for _, rule := range similarityTierRules {
		if score >= rule.MinScore {
			rule.ColorHex, rule.ImageURL = ResolveIndicatorAsset(rule.Tier)
			return rule
		}
	}
	rule := similarityTierRules[len(similarityTierRules)-1]
	rule.ColorHex, rule.ImageURL = ResolveIndicatorAsset(rule.Tier)
	return rule
}

// ResolveIndicatorAsset returns the color and static asset URL for a tier.
// Implements DESIGN-003 SimilarityAssetResolver.
func ResolveIndicatorAsset(tier SimilarityTier) (string, string) {
	fallback := similarityTierRules[len(similarityTierRules)-1]
	for _, rule := range similarityTierRules {
		if rule.Tier == tier {
			if indicatorAssetExists(rule.ImageURL) {
				return rule.ColorHex, rule.ImageURL
			}
			return rule.ColorHex, fallback.ImageURL
		}
	}
	return fallback.ColorHex, fallback.ImageURL
}

// StaticAssetRoot returns the backend-served asset directory.
// Implements DESIGN-003 SimilarityAssetResolver static asset root.
func StaticAssetRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return filepath.Join("static", "assets")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "static", "assets"))
}

// indicatorAssetExists checks whether a similarity indicator asset can be served.
// Implements DESIGN-003 SimilarityAssetResolver.
func indicatorAssetExists(assetURL string) bool {
	rel, ok := strings.CutPrefix(assetURL, "/assets/")
	if !ok {
		return false
	}
	info, err := os.Stat(filepath.Join(StaticAssetRoot(), filepath.FromSlash(rel)))
	return err == nil && !info.IsDir()
}

// CalculateMatchingQuantity derives a replacement quantity from calorie or protein matching.
// Implements DESIGN-003 CosineSimilarityCalculator.
func CalculateMatchingQuantity(source repository.MacroValues, sourceCalories float64, target TargetMacroVector, matchType MatchType) float64 {
	switch matchType {
	case MatchTypeCalorie:
		if target.CaloriesPerBaseUnit == 0 {
			return 0
		}
		return sourceCalories / target.CaloriesPerBaseUnit
	case MatchTypeProtein:
		if target.ProteinPerBaseUnit == 0 {
			return 0
		}
		return source.Protein / target.ProteinPerBaseUnit
	default:
		return 0
	}
}

// validateSimilarityMacros rejects invalid macro vectors before similarity math.
// Implements DESIGN-003 CosineSimilarityCalculator.
func validateSimilarityMacros(v repository.MacroValues) error {
	if math.IsNaN(v.Protein) || math.IsNaN(v.Carbohydrates) || math.IsNaN(v.Fat) {
		return errors.New("macro values must be finite")
	}
	if math.IsInf(v.Protein, 0) || math.IsInf(v.Carbohydrates, 0) || math.IsInf(v.Fat, 0) {
		return errors.New("macro values must be finite")
	}
	if v.Protein < 0 || v.Carbohydrates < 0 || v.Fat < 0 {
		return errors.New("macro values cannot be negative")
	}
	return nil
}

// isZeroMacroVector detects macro vectors with no comparison signal.
// Implements DESIGN-003 CosineSimilarityCalculator.
func isZeroMacroVector(v repository.MacroValues) bool {
	return v.Protein == 0 && v.Carbohydrates == 0 && v.Fat == 0
}
