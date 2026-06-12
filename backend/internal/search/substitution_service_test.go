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
			{FoodObjectID: sourceAID, Quantity: 100, Unit: "gram"},
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
	if response.SimilarityMetadata[0].Tier != SimilarityTierExcellent || response.SimilarityMetadata[0].ColorHex == "" || response.SimilarityMetadata[0].ImageURL == "" || response.SimilarityMetadata[0].MatchingQuantity <= 0 {
		t.Fatalf("multi-input tier metadata = %+v", response.SimilarityMetadata[0])
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
	if response.SimilarityMetadata[0].Tier != SimilarityTierFair || response.SimilarityMetadata[0].ColorHex != "#B7791F" || response.SimilarityMetadata[0].ImageURL != "/assets/similarity/fair.svg" || response.SimilarityMetadata[0].MatchingQuantity <= 0 {
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
	assertWarningContains(t, response.Warnings, "skipped source 50000000-0000-4000-8000-000000000099 load_failed")

	rejected, err := NewSubstitutionService(repo, nil).Search(context.Background(), SearchRequest{Query: "swap", Mode: SearchModeSubstitution, Page: 1})
	if err != nil || rejected.Rejection == nil || rejected.Rejection.Field != "substitutionInputs" {
		t.Fatalf("empty input rejection = %+v err=%v", rejected, err)
	}
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

func assertWarningContains(t *testing.T, warnings []string, want string) {
	t.Helper()
	for _, warning := range warnings {
		if strings.Contains(warning, want) {
			return
		}
	}
	t.Fatalf("missing warning %q in %#v", want, warnings)
}
