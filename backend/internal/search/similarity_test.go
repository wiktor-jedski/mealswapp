package search

import (
	"context"
	"errors"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// Implements DESIGN-003 CosineSimilarityCalculator verification.

type fakeMacroAggregator struct {
	macros repository.MacroValues
	err    error
	calls  int
}

func (f *fakeMacroAggregator) CalculateMacros(context.Context, uuid.UUID) (repository.MacroValues, error) {
	f.calls++
	return f.macros, f.err
}

func TestNormalizeMacroVectorValidatesFiniteNonNegativeAndZeroSource(t *testing.T) {
	normalized, err := NormalizeMacroVector(repository.MacroValues{Protein: 3, Carbohydrates: 4})
	if err != nil {
		t.Fatal(err)
	}
	if !nearlyEqual(normalized.Protein, 0.6) || !nearlyEqual(normalized.Carbohydrates, 0.8) || normalized.Magnitude != 5 {
		t.Fatalf("normalized vector = %+v", normalized)
	}

	for name, macros := range map[string]repository.MacroValues{
		"negative": {Protein: -1},
		"nan":      {Protein: math.NaN()},
		"infinite": {Protein: math.Inf(1)},
		"zero":     {},
	} {
		if _, err := NormalizeMacroVector(macros); err == nil {
			t.Fatalf("%s vector accepted", name)
		}
	}
}

func TestCompareMacrosIgnoresMicronutrientsAndRanksByCosineScore(t *testing.T) {
	source := repository.FoodItemEntity{
		MacrosPer100: repository.MacroValues{Protein: 10, Carbohydrates: 20, Fat: 5},
		Micros:       repository.MicroValues{"Sodium": 999999},
	}
	highID := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	lowID := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	belowID := uuid.MustParse("33333333-3333-4333-8333-333333333333")
	zeroID := uuid.MustParse("44444444-4444-4444-8444-444444444444")

	results, diagnostics, err := CompareMacros(context.Background(), ComparisonRequest{
		SourceMacros:   source.MacrosPer100,
		SourceCalories: 250,
		MatchType:      MatchTypeCalorie,
		Targets: []TargetMacroVector{
			{ItemID: lowID, Macros: repository.MacroValues{Protein: 20, Carbohydrates: 5, Fat: 5}, CaloriesPerBaseUnit: 50},
			{ItemID: zeroID, Macros: repository.MacroValues{}, CaloriesPerBaseUnit: 50},
			{ItemID: highID, Macros: repository.MacroValues{Protein: 10, Carbohydrates: 20, Fat: 5}, CaloriesPerBaseUnit: 25},
			{ItemID: belowID, Macros: repository.MacroValues{Protein: 0, Carbohydrates: 0, Fat: 10}, CaloriesPerBaseUnit: 50},
		},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("result count = %d, results = %+v diagnostics = %+v", len(results), results, diagnostics)
	}
	if results[0].ItemID != highID || results[1].ItemID != lowID {
		t.Fatalf("results not sorted by score descending: %+v", results)
	}
	if !nearlyEqual(results[0].Score, 1) {
		t.Fatalf("top score = %f", results[0].Score)
	}
	if results[0].MatchingQuantity != 10 {
		t.Fatalf("calorie matching quantity = %f", results[0].MatchingQuantity)
	}
	if results[0].Tier != SimilarityTierExcellent || results[0].ColorHex == "" || results[0].ImageURL == "" {
		t.Fatalf("tier metadata = %+v", results[0])
	}
	assertDiagnostic(t, diagnostics, zeroID, "zero_target_vector")
	assertDiagnostic(t, diagnostics, belowID, "below_threshold")
}

func TestCompareMacrosAppliesThresholdAndProteinMatchingQuantity(t *testing.T) {
	targetID := uuid.MustParse("55555555-5555-4555-8555-555555555555")
	filteredID := uuid.MustParse("66666666-6666-4666-8666-666666666666")
	results, diagnostics, err := CompareMacros(context.Background(), ComparisonRequest{
		SourceMacros:        repository.MacroValues{Protein: 40, Carbohydrates: 40, Fat: 20},
		MatchType:           MatchTypeProtein,
		SimilarityThreshold: 0.40,
		Targets: []TargetMacroVector{
			{ItemID: targetID, Macros: repository.MacroValues{Protein: 20, Carbohydrates: 20, Fat: 10}, ProteinPerBaseUnit: 8},
			{ItemID: filteredID, Macros: repository.MacroValues{Protein: 0, Carbohydrates: 0, Fat: 10}, ProteinPerBaseUnit: 8},
		},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].ItemID != targetID {
		t.Fatalf("results = %+v", results)
	}
	if results[0].MatchingQuantity != 5 {
		t.Fatalf("protein matching quantity = %f", results[0].MatchingQuantity)
	}
	assertDiagnostic(t, diagnostics, filteredID, "below_threshold")
}

func TestCompareMacrosAggregatesRecipeTargetsAndBubblesRepositoryErrors(t *testing.T) {
	itemID := uuid.MustParse("77777777-7777-4777-8777-777777777777")
	mealID := uuid.MustParse("88888888-8888-4888-8888-888888888888")
	aggregator := &fakeMacroAggregator{macros: repository.MacroValues{Protein: 5, Carbohydrates: 10, Fat: 5}}

	results, _, err := CompareMacros(context.Background(), ComparisonRequest{
		SourceMacros: repository.MacroValues{Protein: 5, Carbohydrates: 10, Fat: 5},
		MatchType:    MatchTypeProtein,
		Targets:      []TargetMacroVector{{ItemID: itemID, RecipeMealID: &mealID, ProteinPerBaseUnit: 2}},
	}, aggregator)
	if err != nil {
		t.Fatal(err)
	}
	if aggregator.calls != 1 || len(results) != 1 || results[0].ItemID != itemID {
		t.Fatalf("aggregation result calls=%d results=%+v", aggregator.calls, results)
	}

	repoErr := errors.New("repository unavailable")
	_, _, err = CompareMacros(context.Background(), ComparisonRequest{
		SourceMacros: repository.MacroValues{Protein: 5, Carbohydrates: 10, Fat: 5},
		Targets:      []TargetMacroVector{{ItemID: itemID, RecipeMealID: &mealID}},
	}, &fakeMacroAggregator{err: repoErr})
	if !errors.Is(err, repoErr) {
		t.Fatalf("repository error = %v", err)
	}
}

func TestSimilarityHelpersFilterAssetsAndUncalculableQuantity(t *testing.T) {
	results := FilterByThreshold([]SimilarityResult{{Score: 0.39}, {Score: 0.40}, {Score: 0.90}}, 0.40)
	if len(results) != 2 || results[0].Score != 0.40 || results[1].Score != 0.90 {
		t.Fatalf("filtered results = %+v", results)
	}
	if CosineSimilarity(NormalizedMacroVector{Protein: 1}, NormalizedMacroVector{Carbohydrates: 1}) != 0 {
		t.Fatal("orthogonal cosine should be zero")
	}
	if rule := MapSimilarityTier(0.70); rule.Tier != SimilarityTierGood {
		t.Fatalf("tier rule = %+v", rule)
	}
	color, image := ResolveIndicatorAsset(SimilarityTierFair)
	if color == "" || !strings.Contains(image, "fair") {
		t.Fatalf("asset = %q %q", color, image)
	}
	if quantity := CalculateMatchingQuantity(repository.MacroValues{Protein: 10}, 100, TargetMacroVector{}, MatchTypeCalorie); quantity != 0 {
		t.Fatalf("zero calorie denominator quantity = %f", quantity)
	}
	if quantity := CalculateMatchingQuantity(repository.MacroValues{Protein: 10}, 100, TargetMacroVector{}, MatchTypeProtein); quantity != 0 {
		t.Fatalf("zero protein denominator quantity = %f", quantity)
	}
}

func TestMapSimilarityTierBoundariesColorsAndAssetURLs(t *testing.T) {
	for _, tc := range []struct {
		score float64
		tier  SimilarityTier
		color string
		image string
	}{
		{score: 0.85, tier: SimilarityTierExcellent, color: "#1B7F4C", image: "/assets/similarity/excellent.svg"},
		{score: 0.849999, tier: SimilarityTierGood, color: "#2F80ED", image: "/assets/similarity/good.svg"},
		{score: 0.70, tier: SimilarityTierGood, color: "#2F80ED", image: "/assets/similarity/good.svg"},
		{score: 0.699999, tier: SimilarityTierFair, color: "#B7791F", image: "/assets/similarity/fair.svg"},
		{score: 0.55, tier: SimilarityTierFair, color: "#B7791F", image: "/assets/similarity/fair.svg"},
		{score: 0.549999, tier: SimilarityTierPoor, color: "#A23B3B", image: "/assets/similarity/poor.svg"},
	} {
		rule := MapSimilarityTier(tc.score)
		if rule.Tier != tc.tier || rule.ColorHex != tc.color || rule.ImageURL != tc.image {
			t.Fatalf("MapSimilarityTier(%f) = %+v, want tier=%s color=%s image=%s", tc.score, rule, tc.tier, tc.color, tc.image)
		}
	}
}

func TestResolveIndicatorAssetFallsBackWhenTierFileIsMissing(t *testing.T) {
	assetPath := filepath.Join(StaticAssetRoot(), "similarity", "excellent.svg")
	renamedPath := assetPath + ".test-missing"
	if err := os.Rename(assetPath, renamedPath); err != nil {
		t.Fatalf("rename fixture asset: %v", err)
	}
	defer func() {
		if err := os.Rename(renamedPath, assetPath); err != nil {
			t.Fatalf("restore fixture asset: %v", err)
		}
	}()

	color, image := ResolveIndicatorAsset(SimilarityTierExcellent)
	if color != "#1B7F4C" || image != "/assets/similarity/poor.svg" {
		t.Fatalf("missing excellent asset resolved color=%q image=%q", color, image)
	}
}

func assertDiagnostic(t *testing.T, diagnostics []SimilarityDiagnostic, itemID uuid.UUID, code string) {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if diagnostic.ItemID == itemID && diagnostic.Code == code {
			return
		}
	}
	t.Fatalf("missing diagnostic %s for %s in %+v", code, itemID, diagnostics)
}

func nearlyEqual(left float64, right float64) bool {
	return math.Abs(left-right) < 0.000001
}
