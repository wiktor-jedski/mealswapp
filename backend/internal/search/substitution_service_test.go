package search

import (
	"context"
	"errors"
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// Implements DESIGN-002 CulinaryRoleWeighter and Substitution Search service verification.

type substitutionRepositoryStub struct {
	byID      map[uuid.UUID]repository.FoodItemEntity
	items     []repository.FoodItemEntity
	query     repository.RepositoryQuery
	getCalls  int
	searches  int
	searchErr error
}

type similarityCacheStub struct {
	calculation SimilarityCalculation
	hit         bool
	getErr      error
	setErr      error
	gets        int
	sets        int
	getInputs   []SubstitutionInput
	setInputs   []SubstitutionInput
}

func (c *similarityCacheStub) GetSimilarityCalculation(_ context.Context, inputs []SubstitutionInput) (SimilarityCalculation, bool, error) {
	c.gets++
	c.getInputs = slices.Clone(inputs)
	return c.calculation, c.hit, c.getErr
}

func (c *similarityCacheStub) SetSimilarityCalculation(_ context.Context, inputs []SubstitutionInput, calculation SimilarityCalculation) error {
	c.sets++
	c.setInputs = slices.Clone(inputs)
	c.calculation = calculation
	c.hit = true
	return c.setErr
}

func (r *substitutionRepositoryStub) GetByID(_ context.Context, id uuid.UUID, _ repository.RepositoryContext) (repository.FoodItemEntity, error) {
	r.getCalls++
	item, ok := r.byID[id]
	if !ok {
		return repository.FoodItemEntity{}, errors.New("missing source")
	}
	return item, nil
}

func (r *substitutionRepositoryStub) Search(_ context.Context, q repository.RepositoryQuery) ([]repository.FoodItemEntity, int, error) {
	r.searches++
	r.query = q
	return slices.Clone(r.items), len(r.items), r.searchErr
}

func TestSubstitutionServiceCombinesMultipleInputsWithoutPerInputCulinaryOrdering(t *testing.T) {
	roleID := uuid.MustParse("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa")
	sourceAID := uuid.MustParse("10000000-0000-4000-8000-000000000001")
	sourceBID := uuid.MustParse("10000000-0000-4000-8000-000000000002")
	weightedCandidateID := uuid.MustParse("20000000-0000-4000-8000-000000000001")
	bestMacroCandidateID := uuid.MustParse("20000000-0000-4000-8000-000000000002")

	role := repository.ClassificationEntity{ID: roleID, Kind: repository.ClassificationKindCulinaryRole, Name: "filling"}
	repo := &substitutionRepositoryStub{
		byID: map[uuid.UUID]repository.FoodItemEntity{
			sourceAID: {ID: sourceAID, Name: "Source Beans", PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{Protein: 20, Carbohydrates: 20, Fat: 10}, CulinaryRoles: []repository.ClassificationEntity{role}},
			sourceBID: {ID: sourceBID, Name: "Source Oil", PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{Fat: 100}},
		},
		items: []repository.FoodItemEntity{
			{ID: weightedCandidateID, Name: "Role Match", PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{Protein: 20, Carbohydrates: 20, Fat: 10}, CulinaryRoles: []repository.ClassificationEntity{role}},
			{ID: bestMacroCandidateID, Name: "Macro Match", PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{Protein: 10, Carbohydrates: 10, Fat: 55}},
		},
	}

	response, err := NewSubstitutionService(repo, nil).Search(context.Background(), SearchRequest{
		Query: "  alternative ",
		Mode:  SearchModeSubstitution,
		Page:  1,
		SubstitutionInputs: []SubstitutionInput{
			{FoodObjectID: sourceAID, Quantity: 100, Unit: "g"},
			{FoodObjectID: sourceBID, Quantity: 100, Unit: "g"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if repo.searches != 1 || repo.query.Name != "alternative" {
		t.Fatalf("repository query = %+v searches=%d", repo.query, repo.searches)
	}
	if len(response.Items) != 2 || response.Items[0].ID != bestMacroCandidateID || response.Items[1].ID != weightedCandidateID {
		t.Fatalf("multi-input response order = %+v scores=%+v", response.Items, response.SimilarityScores)
	}
	if response.SimilarityScores[1] > 1 {
		t.Fatalf("multi-input score received role boost: %+v", response.SimilarityScores)
	}
	if response.TotalCount != 2 || len(response.Warnings) != 0 {
		t.Fatalf("response metadata = %+v", response)
	}
	if len(response.SimilarityMetadata) != 2 || response.SimilarityMetadata[0].ItemID != bestMacroCandidateID || response.SimilarityMetadata[1].ItemID != weightedCandidateID {
		t.Fatalf("multi-input similarity metadata order = %+v", response.SimilarityMetadata)
	}
	if response.SimilarityMetadata[0].Tier != SimilarityTierExcellent || response.SimilarityMetadata[0].ImageURL == "" || response.SimilarityMetadata[0].MatchingQuantity <= 0 {
		t.Fatalf("multi-input tier metadata = %+v", response.SimilarityMetadata[0])
	}
	if response.SourceSummary == nil || response.SourceSummary.TotalGrams != 200 || response.SourceSummary.TotalMilliliters != 0 || response.SourceSummary.Macros.Fat != 110 {
		t.Fatalf("multi-input source summary = %+v", response.SourceSummary)
	}
}

func TestSubstitutionServiceSourceSummarySeparatesMassAndVolume(t *testing.T) {
	solidID := uuid.MustParse("11000000-0000-4000-8000-000000000001")
	liquidID := uuid.MustParse("11000000-0000-4000-8000-000000000002")
	candidateID := uuid.MustParse("21000000-0000-4000-8000-000000000001")
	repo := &substitutionRepositoryStub{
		byID: map[uuid.UUID]repository.FoodItemEntity{
			solidID:  {ID: solidID, Name: "Apple", PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{Protein: 1, Carbohydrates: 10, Fat: 1}},
			liquidID: {ID: liquidID, Name: "Milk", PhysicalState: repository.PhysicalStateLiquid, MacrosPer100: repository.MacroValues{Protein: 3, Carbohydrates: 5, Fat: 1}},
		},
		items: []repository.FoodItemEntity{
			{ID: candidateID, Name: "Candidate", PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{Protein: 4, Carbohydrates: 15, Fat: 2}},
		},
	}

	response, err := NewSubstitutionService(repo, nil).Search(context.Background(), SearchRequest{
		Query: "swap",
		Mode:  SearchModeSubstitution,
		Page:  1,
		SubstitutionInputs: []SubstitutionInput{
			{FoodObjectID: solidID, Quantity: 150, Unit: "g"},
			{FoodObjectID: liquidID, Quantity: 125, Unit: "ml"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.SourceSummary == nil {
		t.Fatal("source summary is nil")
	}
	if response.SourceSummary.TotalGrams != 150 || response.SourceSummary.TotalMilliliters != 125 {
		t.Fatalf("source summary amounts = %+v", response.SourceSummary)
	}
	if response.SourceSummary.Macros.Protein != 5.25 || response.SourceSummary.Macros.Carbohydrates != 21.25 || response.SourceSummary.Macros.Fat != 2.75 {
		t.Fatalf("source summary macros = %+v", response.SourceSummary.Macros)
	}
	if response.SourceSummary.Calories != CalculateCalories(response.SourceSummary.Macros) {
		t.Fatalf("source summary calories = %v macros=%+v", response.SourceSummary.Calories, response.SourceSummary.Macros)
	}
}

func TestSubstitutionServiceAppliesSingleInputCulinaryRoleWeightThresholdWarningsAndTieSort(t *testing.T) {
	roleA := repository.ClassificationEntity{ID: uuid.MustParse("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"), Kind: repository.ClassificationKindCulinaryRole, Name: "spread"}
	roleB := repository.ClassificationEntity{ID: uuid.MustParse("bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"), Kind: repository.ClassificationKindCulinaryRole, Name: "protein"}
	roleC := repository.ClassificationEntity{ID: uuid.MustParse("cccccccc-cccc-4ccc-8ccc-cccccccccccc"), Kind: repository.ClassificationKindCulinaryRole, Name: "base"}
	sourceID := uuid.MustParse("30000000-0000-4000-8000-000000000001")
	boostedID := uuid.MustParse("40000000-0000-4000-8000-000000000001")
	tieAID := uuid.MustParse("40000000-0000-4000-8000-000000000002")
	tieBID := uuid.MustParse("40000000-0000-4000-8000-000000000003")
	belowID := uuid.MustParse("40000000-0000-4000-8000-000000000004")
	zeroID := uuid.MustParse("40000000-0000-4000-8000-000000000005")
	repo := &substitutionRepositoryStub{
		byID: map[uuid.UUID]repository.FoodItemEntity{
			sourceID: {ID: sourceID, Name: "Source", PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{Protein: 10, Carbohydrates: 20, Fat: 5}, CulinaryRoles: []repository.ClassificationEntity{roleA, roleB, roleC}},
		},
		items: []repository.FoodItemEntity{
			{ID: tieBID, Name: "Beta", PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{Protein: 10, Carbohydrates: 20, Fat: 5}},
			{ID: zeroID, Name: "Zero", PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{}},
			{ID: belowID, Name: "Below", PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{Fat: 20}},
			{ID: tieAID, Name: "Alpha", PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{Protein: 10, Carbohydrates: 20, Fat: 5}},
			{ID: boostedID, Name: "Boosted", PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{Protein: 20, Carbohydrates: 5, Fat: 5}, CulinaryRoles: []repository.ClassificationEntity{roleA, roleB, roleC}},
		},
	}

	response, err := NewSubstitutionService(repo, nil).Search(context.Background(), SearchRequest{
		Query: "swap",
		Mode:  SearchModeSubstitution,
		Page:  1,
		SubstitutionInputs: []SubstitutionInput{
			{FoodObjectID: sourceID, Quantity: 100, Unit: "g"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Items) != 3 {
		t.Fatalf("response = %+v", response)
	}
	if response.Items[0].ID != boostedID {
		t.Fatalf("boosted candidate was not first: %+v scores=%+v", response.Items, response.SimilarityScores)
	}
	wantBoost := 0.668647849835731 * (1 + 0.2*3)
	if math.Abs(response.SimilarityScores[0]-wantBoost) > 0.0000001 {
		t.Fatalf("boosted score = %.12f want %.12f", response.SimilarityScores[0], wantBoost)
	}
	if len(response.SimilarityMetadata) != 3 {
		t.Fatalf("similarity metadata = %+v", response.SimilarityMetadata)
	}
	if response.SimilarityMetadata[0].ItemID != boostedID {
		t.Fatalf("metadata order does not match ranked items: items=%+v metadata=%+v", response.Items, response.SimilarityMetadata)
	}
	if math.Abs(response.SimilarityMetadata[0].Score-0.668647849835731) > 0.0000001 {
		t.Fatalf("metadata raw score = %.12f", response.SimilarityMetadata[0].Score)
	}
	if response.SimilarityMetadata[0].Tier != SimilarityTierFair || response.SimilarityMetadata[0].ImageURL != "/assets/similarity/fair.svg" || response.SimilarityMetadata[0].MatchingQuantity <= 0 {
		t.Fatalf("boosted tier metadata = %+v", response.SimilarityMetadata[0])
	}
	if response.Items[1].ID != tieAID || response.Items[2].ID != tieBID {
		t.Fatalf("tie sort = %+v", response.Items)
	}
	if response.SimilarityMetadata[1].ItemID != tieAID || response.SimilarityMetadata[1].Tier != SimilarityTierExcellent || response.SimilarityMetadata[1].MatchingQuantity != 100 {
		t.Fatalf("tie metadata = %+v", response.SimilarityMetadata[1])
	}
	assertWarningContains(t, response.Warnings, "skipped target "+belowID.String()+" below_threshold")
	assertWarningContains(t, response.Warnings, "skipped target "+zeroID.String()+" zero_target_vector")
}

func TestSubstitutionServiceExcludesSourceInputsFromResults(t *testing.T) {
	sourceID := uuid.MustParse("35000000-0000-4000-8000-000000000001")
	alternativeID := uuid.MustParse("35000000-0000-4000-8000-000000000002")
	repo := &substitutionRepositoryStub{
		byID: map[uuid.UUID]repository.FoodItemEntity{
			sourceID: {ID: sourceID, Name: "Apple", PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{Protein: 1, Carbohydrates: 10, Fat: 1}},
		},
		items: []repository.FoodItemEntity{
			{ID: sourceID, Name: "Apple", PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{Protein: 1, Carbohydrates: 10, Fat: 1}},
			{ID: alternativeID, Name: "Pear", PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{Protein: 1, Carbohydrates: 10, Fat: 1}},
		},
	}

	response, err := NewSubstitutionService(repo, nil).Search(context.Background(), SearchRequest{
		Query: "",
		Mode:  SearchModeSubstitution,
		Page:  1,
		SubstitutionInputs: []SubstitutionInput{
			{FoodObjectID: sourceID, Quantity: 100, Unit: "g"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Items) != 1 || response.Items[0].ID != alternativeID {
		t.Fatalf("source item was not excluded: %+v", response.Items)
	}
	if len(response.SimilarityMetadata) != 1 || response.SimilarityMetadata[0].ItemID != alternativeID {
		t.Fatalf("source metadata was not excluded: %+v", response.SimilarityMetadata)
	}
}

func TestRankSubstitutionCandidatesPreservesResultOrderForDuplicateNameFixture(t *testing.T) {
	firstID := uuid.MustParse("40000000-0000-4000-8000-000000000003")
	secondID := uuid.MustParse("40000000-0000-4000-8000-000000000001")
	items := []repository.FoodItemEntity{
		{ID: secondID, Name: "Apple"},
		{ID: firstID, Name: "Apple"},
	}
	results := []SimilarityResult{
		{ItemID: firstID, Score: 0.9, Tier: SimilarityTierExcellent, ImageURL: "/assets/similarity/excellent.svg", MatchingQuantity: 100},
		{ItemID: secondID, Score: 0.9, Tier: SimilarityTierExcellent, ImageURL: "/assets/similarity/excellent.svg", MatchingQuantity: 100},
	}

	ranked := rankSubstitutionCandidates(items, results, false, nil)

	if len(ranked.items) != 2 || ranked.items[0].ID != firstID || ranked.items[1].ID != secondID {
		t.Fatalf("ranked duplicate-name fixture = %+v", ranked.items)
	}
	if ranked.metadata[0].ItemID != firstID || ranked.metadata[1].ItemID != secondID {
		t.Fatalf("ranked metadata = %+v", ranked.metadata)
	}
}

func TestRankSubstitutionCandidatesSkipsMissingRepositoryItems(t *testing.T) {
	ranked := rankSubstitutionCandidates(nil, []SimilarityResult{{ItemID: uuid.New(), Score: 1}}, false, nil)
	if len(ranked.items) != 0 || len(ranked.scores) != 0 || len(ranked.metadata) != 0 {
		t.Fatalf("ranked missing item = %+v", ranked)
	}
}

func TestSubstitutionServiceCachesRejectionsAndSkippedSources(t *testing.T) {
	sourceID := uuid.MustParse("50000000-0000-4000-8000-000000000001")
	candidateID := uuid.MustParse("50000000-0000-4000-8000-000000000002")
	cache := &searchCacheStub{}
	repo := &substitutionRepositoryStub{
		byID: map[uuid.UUID]repository.FoodItemEntity{
			sourceID: {ID: sourceID, Name: "Milk", PhysicalState: repository.PhysicalStateLiquid, AverageServingVolumeMilliliters: 250, MacrosPer100: repository.MacroValues{Protein: 3, Carbohydrates: 5, Fat: 1}},
		},
		items: []repository.FoodItemEntity{{ID: candidateID, Name: "Soy", PhysicalState: repository.PhysicalStateLiquid, MacrosPer100: repository.MacroValues{Protein: 3, Carbohydrates: 5, Fat: 1}}},
	}

	response, err := NewSubstitutionService(repo, cache).Search(context.Background(), SearchRequest{
		Query: "  MILK ",
		Mode:  SearchModeSubstitution,
		Page:  1,
		SubstitutionInputs: []SubstitutionInput{
			{FoodObjectID: uuid.MustParse("50000000-0000-4000-8000-000000000099"), Quantity: 1, Unit: "serving"},
			{FoodObjectID: sourceID, Quantity: 1, Unit: "serving"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.Page != 1 || len(response.Items) != 1 || cache.gets != 1 || cache.sets != 1 || cache.setReq.Query != "milk" || cache.setReq.Page != 1 {
		t.Fatalf("response=%+v cache gets=%d sets=%d req=%+v", response, cache.gets, cache.sets, cache.setReq)
	}
	if response.Cache == nil || response.Cache.Status != CacheStatusMiss || response.Cache.Namespace != "search" || response.Cache.SchemaVersion != "search-response-v2" || response.Cache.TTLSeconds != 300 {
		t.Fatalf("cache miss metadata = %+v", response.Cache)
	}
	assertWarningContains(t, response.Warnings, "skipped source 50000000-0000-4000-8000-000000000099 load_failed")

	rejected, err := NewSubstitutionService(repo, nil).Search(context.Background(), SearchRequest{Query: "swap", Mode: SearchModeSubstitution, Page: 1})
	if err != nil || rejected.Rejection == nil || rejected.Rejection.Field != "substitutionInputs" {
		t.Fatalf("empty input rejection = %+v err=%v", rejected, err)
	}
	if rejected.Cache != nil {
		t.Fatalf("rejection advertised cache metadata = %+v", rejected.Cache)
	}
}

func TestSubstitutionServiceCachesSimilarityCalculationsBeforeMacroComparison(t *testing.T) {
	sourceID := uuid.MustParse("60000000-0000-4000-8000-000000000001")
	candidateID := uuid.MustParse("60000000-0000-4000-8000-000000000002")
	repo := &substitutionRepositoryStub{
		byID: map[uuid.UUID]repository.FoodItemEntity{
			sourceID: {ID: sourceID, Name: "Source", PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{Protein: 10, Carbohydrates: 20, Fat: 5}},
		},
		items: []repository.FoodItemEntity{
			{ID: candidateID, Name: "Cached Target", PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{}},
		},
	}
	cache := &similarityCacheStub{hit: true, calculation: SimilarityCalculation{Results: []SimilarityResult{{
		ItemID:           candidateID,
		Score:            0.91,
		Tier:             SimilarityTierExcellent,
		ImageURL:         "/assets/similarity/excellent.svg",
		MatchingQuantity: 88,
	}}}}

	response, err := NewSubstitutionService(repo, nil, cache).Search(context.Background(), SearchRequest{
		Query: "swap",
		Mode:  SearchModeSubstitution,
		Page:  1,
		SubstitutionInputs: []SubstitutionInput{
			{FoodObjectID: sourceID, Quantity: 100, Unit: "g"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if cache.gets != 1 || cache.sets != 0 || len(cache.getInputs) != 1 {
		t.Fatalf("similarity cache calls gets=%d sets=%d inputs=%+v", cache.gets, cache.sets, cache.getInputs)
	}
	if len(response.Items) != 1 || response.Items[0].ID != candidateID || response.SimilarityMetadata[0].Score != 0.91 || response.SimilarityMetadata[0].MatchingQuantity != 88 {
		t.Fatalf("cached similarity response = %+v", response)
	}
	if len(response.Warnings) != 0 {
		t.Fatalf("cached similarity produced warnings: %+v", response.Warnings)
	}
}

func TestSubstitutionServiceWritesAndReusesSimilarityCache(t *testing.T) {
	sourceID := uuid.MustParse("61000000-0000-4000-8000-000000000001")
	candidateID := uuid.MustParse("61000000-0000-4000-8000-000000000002")
	repo := &substitutionRepositoryStub{
		byID: map[uuid.UUID]repository.FoodItemEntity{
			sourceID: {ID: sourceID, Name: "Source", PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{Protein: 10, Carbohydrates: 20, Fat: 5}},
		},
		items: []repository.FoodItemEntity{
			{ID: candidateID, Name: "Target", PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{Protein: 10, Carbohydrates: 20, Fat: 5}},
		},
	}
	cache := &similarityCacheStub{}
	service := NewSubstitutionService(repo, nil, cache)
	req := SearchRequest{Query: "swap", Mode: SearchModeSubstitution, Page: 1, SubstitutionInputs: []SubstitutionInput{{FoodObjectID: sourceID, Quantity: 100, Unit: "g"}}}

	first, err := service.Search(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	secondReq := req
	secondReq.SubstitutionInputs = []SubstitutionInput{{FoodObjectID: sourceID, Quantity: 100, Unit: "g"}}
	second, err := service.Search(context.Background(), secondReq)
	if err != nil {
		t.Fatal(err)
	}
	if cache.gets != 2 || cache.sets != 1 {
		t.Fatalf("similarity cache calls gets=%d sets=%d", cache.gets, cache.sets)
	}
	if len(first.Items) != 1 || len(second.Items) != 1 || first.Items[0].ID != second.Items[0].ID || first.SimilarityMetadata[0] != second.SimilarityMetadata[0] {
		t.Fatalf("cached repeat mismatch first=%+v second=%+v", first, second)
	}
}

func TestSubstitutionServiceWarnsAndFallsBackWhenSimilarityCacheUnavailable(t *testing.T) {
	sourceID := uuid.MustParse("62000000-0000-4000-8000-000000000001")
	candidateID := uuid.MustParse("62000000-0000-4000-8000-000000000002")
	repo := &substitutionRepositoryStub{
		byID: map[uuid.UUID]repository.FoodItemEntity{
			sourceID: {ID: sourceID, Name: "Source", PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{Protein: 10, Carbohydrates: 20, Fat: 5}},
		},
		items: []repository.FoodItemEntity{
			{ID: candidateID, Name: "Target", PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{Protein: 10, Carbohydrates: 20, Fat: 5}},
		},
	}
	cache := &similarityCacheStub{getErr: errors.New("redis down"), setErr: errors.New("redis still down")}

	response, err := NewSubstitutionService(repo, nil, cache).Search(context.Background(), SearchRequest{
		Query: "swap",
		Mode:  SearchModeSubstitution,
		Page:  1,
		SubstitutionInputs: []SubstitutionInput{
			{FoodObjectID: sourceID, Quantity: 100, Unit: "g"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Items) != 1 || cache.gets != 1 || cache.sets != 1 {
		t.Fatalf("fallback response=%+v cache gets=%d sets=%d", response, cache.gets, cache.sets)
	}
	assertWarningContains(t, response.Warnings, WarningCacheUnavailable)
}

func TestApplyCulinaryRoleWeightCountsUniqueSharedRoles(t *testing.T) {
	roleID := uuid.MustParse("cccccccc-cccc-4ccc-8ccc-cccccccccccc")
	otherID := uuid.MustParse("dddddddd-dddd-4ddd-8ddd-dddddddddddd")
	candidateRoles := []repository.ClassificationEntity{{ID: roleID}, {ID: roleID}, {ID: otherID}}
	sourceRoles := []repository.ClassificationEntity{{ID: roleID}}
	if score := ApplyCulinaryRoleWeight(0.5, candidateRoles, sourceRoles); score != 0.6 {
		t.Fatalf("weighted score = %f", score)
	}
}

func TestSubstitutionServiceFailureAndDegradationPaths(t *testing.T) {
	validSourceID := uuid.New()
	validCandidateID := uuid.New()
	validSource := repository.FoodItemEntity{ID: validSourceID, PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{Protein: 10}}
	validCandidate := repository.FoodItemEntity{ID: validCandidateID, Name: "Candidate", PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{Protein: 10}}
	validRequest := SearchRequest{Query: "swap", Mode: SearchModeSubstitution, Page: 1, SubstitutionInputs: []SubstitutionInput{{FoodObjectID: validSourceID, Quantity: 100, Unit: "g"}}}

	rejected, err := NewSubstitutionService(&substitutionRepositoryStub{}, nil).Search(context.Background(), SearchRequest{Mode: SearchModeSubstitution, Page: 1})
	if err != nil || rejected.Rejection == nil || rejected.Rejection.Field != "substitutionInputs" {
		t.Fatalf("empty input rejection = %+v err=%v", rejected, err)
	}
	response, err := NewSubstitutionService(&substitutionRepositoryStub{}, nil).Search(context.Background(), SearchRequest{Query: "apple", Mode: SearchModeCatalog, Page: 1})
	if err != nil || response.Rejection == nil || response.Rejection.Field != "mode" {
		t.Fatalf("wrong-mode response=%+v err=%v", response, err)
	}

	cached := SearchResponse{Items: []repository.FoodItemEntity{validCandidate}, TotalCount: 1}
	cache := &searchCacheStub{hit: true, response: searchCacheEntry{value: cached}}
	repo := &substitutionRepositoryStub{}
	response, err = NewSubstitutionService(repo, cache).Search(context.Background(), validRequest)
	if err != nil || response.TotalCount != 1 || repo.searches != 0 {
		t.Fatalf("cache-hit response=%+v err=%v searches=%d", response, err, repo.searches)
	}

	repo = &substitutionRepositoryStub{byID: map[uuid.UUID]repository.FoodItemEntity{validSourceID: validSource}, searchErr: errors.New("database down")}
	if _, err := NewSubstitutionService(repo, nil).Search(context.Background(), validRequest); err == nil {
		t.Fatal("Search() swallowed repository error")
	}

	repo = &substitutionRepositoryStub{byID: map[uuid.UUID]repository.FoodItemEntity{validSourceID: validSource}}
	conflicting := validRequest
	conflicting.Filters = []SearchFilter{
		{FilterID: "dairy", Kind: SearchFilterKindAllergen, Include: true},
		{FilterID: string(DietaryPresetDairyFree), Kind: SearchFilterKindDietaryPreset, Include: false},
	}
	response, err = NewSubstitutionService(repo, nil).Search(context.Background(), conflicting)
	if err != nil || response.Rejection == nil {
		t.Fatalf("filter rejection response=%+v err=%v", response, err)
	}

	invalidInput := validRequest
	invalidInput.SubstitutionInputs = []SubstitutionInput{{Quantity: 1, Unit: "g"}}
	response, err = NewSubstitutionService(repo, nil).Search(context.Background(), invalidInput)
	if err != nil || response.Rejection == nil {
		t.Fatalf("invalid-input response=%+v err=%v", response, err)
	}

	conversionFailure := validRequest
	conversionFailure.SubstitutionInputs = []SubstitutionInput{{FoodObjectID: validSourceID, Quantity: 1, Unit: "ml"}}
	response, err = NewSubstitutionService(repo, nil).Search(context.Background(), conversionFailure)
	if err != nil || response.Rejection == nil {
		t.Fatalf("conversion response=%+v err=%v", response, err)
	}
	assertWarningContains(t, response.Warnings, "conversion_failed")

	repo = &substitutionRepositoryStub{
		byID:  map[uuid.UUID]repository.FoodItemEntity{validSourceID: validSource},
		items: []repository.FoodItemEntity{{ID: validCandidateID, MacrosPer100: repository.MacroValues{Protein: -1}}},
	}
	_, err = NewSubstitutionService(repo, nil).Search(context.Background(), validRequest)
	var similarityErr SimilarityUnavailableError
	if !errors.As(err, &similarityErr) || similarityErr.Error() != "similarity_unavailable" || similarityErr.Unwrap() == nil {
		t.Fatalf("similarity error = %v", err)
	}

	repo = &substitutionRepositoryStub{byID: map[uuid.UUID]repository.FoodItemEntity{validSourceID: validSource}, items: []repository.FoodItemEntity{validCandidate}}
	cache = &searchCacheStub{getErr: errors.New("redis get failed"), setErr: errors.New("redis set failed")}
	response, err = NewSubstitutionService(repo, cache).Search(context.Background(), validRequest)
	if err != nil || countWarnings(response.Warnings, WarningCacheUnavailable) != 1 || response.Cache != nil {
		t.Fatalf("cache degradation response=%+v err=%v", response, err)
	}
}

func assertWarningContains(t *testing.T, warnings []string, want string) {
	t.Helper()
	for _, warning := range warnings {
		if strings.Contains(warning, want) {
			return
		}
	}
	t.Fatalf("missing warning %q in %#v", want, warnings)
}
