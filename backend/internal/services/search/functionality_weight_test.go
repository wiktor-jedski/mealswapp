package search

import (
	"math"
	"testing"

	"github.com/google/uuid"
)

func TestApplyFunctionalityWeightBoostsSharedTagMatches(t *testing.T) {
	crunchy := uuid.MustParse("50000000-0000-0000-0000-000000000001")
	spreadable := uuid.MustParse("50000000-0000-0000-0000-000000000002")
	unrelated := uuid.MustParse("50000000-0000-0000-0000-000000000003")
	plainID := uuid.MustParse("50000000-0000-0000-0000-000000000010")
	boostedID := uuid.MustParse("50000000-0000-0000-0000-000000000011")

	weighted := ApplyFunctionalityWeight([]SearchCandidate{
		{ItemID: plainID, SimilarityScore: 0.90, FunctionalityTags: []uuid.UUID{unrelated}},
		{ItemID: boostedID, SimilarityScore: 0.80, FunctionalityTags: []uuid.UUID{crunchy, spreadable}},
	}, []uuid.UUID{crunchy, spreadable})

	if weighted[0].ItemID != boostedID {
		t.Fatalf("expected shared functionality tags to boost candidate first, got %#v", weighted)
	}
	if weighted[0].TagMatchCount != 2 {
		t.Fatalf("expected two tag matches, got %#v", weighted[0])
	}
	if math.Abs(weighted[0].SimilarityScore-0.80) > 0.0001 {
		t.Fatalf("similarity score should be preserved, got %#v", weighted[0])
	}
	if math.Abs(weighted[0].FinalScore-1.12) > 0.0001 {
		t.Fatalf("expected boosted final score 1.12, got %#v", weighted[0])
	}
}

func TestApplyFunctionalityWeightDoesNotBoostWithoutSharedTags(t *testing.T) {
	firstID := uuid.MustParse("60000000-0000-0000-0000-000000000001")
	secondID := uuid.MustParse("60000000-0000-0000-0000-000000000002")
	sourceTag := uuid.MustParse("60000000-0000-0000-0000-000000000010")
	otherTag := uuid.MustParse("60000000-0000-0000-0000-000000000011")

	weighted := ApplyFunctionalityWeight([]SearchCandidate{
		{ItemID: secondID, SimilarityScore: 0.75, FunctionalityTags: []uuid.UUID{otherTag}},
		{ItemID: firstID, SimilarityScore: 0.80},
	}, []uuid.UUID{sourceTag})

	if weighted[0].ItemID != firstID {
		t.Fatalf("expected unboosted candidates to keep similarity ordering, got %#v", weighted)
	}
	if weighted[0].FinalScore != weighted[0].SimilarityScore {
		t.Fatalf("expected no boost when tags do not match, got %#v", weighted[0])
	}
}

func TestApplyFunctionalityWeightSortsDeterministically(t *testing.T) {
	leftID := uuid.MustParse("70000000-0000-0000-0000-000000000001")
	rightID := uuid.MustParse("70000000-0000-0000-0000-000000000002")

	weighted := ApplyFunctionalityWeight([]SearchCandidate{
		{ItemID: rightID, SimilarityScore: 0.70},
		{ItemID: leftID, SimilarityScore: 0.70},
	}, nil)

	if weighted[0].ItemID != leftID || weighted[1].ItemID != rightID {
		t.Fatalf("expected UUID tie-break ordering, got %#v", weighted)
	}
}

func TestApplyFunctionalityWeightCountsDuplicateTagsOnce(t *testing.T) {
	tagID := uuid.MustParse("80000000-0000-0000-0000-000000000001")
	itemID := uuid.MustParse("80000000-0000-0000-0000-000000000010")

	weighted := ApplyFunctionalityWeight([]SearchCandidate{
		{ItemID: itemID, SimilarityScore: 0.50, FunctionalityTags: []uuid.UUID{tagID, tagID}},
	}, []uuid.UUID{tagID})

	if weighted[0].TagMatchCount != 1 {
		t.Fatalf("expected duplicate functionality tags to count once, got %#v", weighted[0])
	}
	if math.Abs(weighted[0].FinalScore-0.60) > 0.0001 {
		t.Fatalf("expected one 20%% boost, got %#v", weighted[0])
	}
}
