package similarity

import (
	"testing"

	"github.com/google/uuid"
)

func TestFilterThresholdExcludesBelowMinimum(t *testing.T) {
	page := FilterThreshold([]ScoredCandidate{
		{ID: uuid.New(), Score: 0.39},
		{ID: uuid.New(), Score: 0.40},
		{ID: uuid.New(), Score: 0.80},
	}, DefaultMinimumSimilarity, 1, 10)

	if page.Total != 2 || len(page.Items) != 2 {
		t.Fatalf("expected two thresholded results, got %#v", page)
	}
}

func TestFilterThresholdCapsPageSizeAtTen(t *testing.T) {
	var scored []ScoredCandidate
	for i := 0; i < 12; i++ {
		scored = append(scored, ScoredCandidate{ID: uuid.New(), Score: 0.90})
	}

	page := FilterThreshold(scored, DefaultMinimumSimilarity, 1, 50)

	if page.PageSize != 10 || len(page.Items) != 10 || page.Total != 12 {
		t.Fatalf("expected capped page size, got %#v", page)
	}
}

func TestFilterThresholdPaginatesResults(t *testing.T) {
	var scored []ScoredCandidate
	for i := 0; i < 12; i++ {
		scored = append(scored, ScoredCandidate{ID: uuid.New(), Score: 0.90})
	}

	page := FilterThreshold(scored, DefaultMinimumSimilarity, 2, 5)

	if page.Page != 2 || page.PageSize != 5 || len(page.Items) != 5 || page.TotalPages != 3 {
		t.Fatalf("unexpected paginated result: %#v", page)
	}
}

func TestFilterThresholdHandlesEmptyPage(t *testing.T) {
	page := FilterThreshold([]ScoredCandidate{{ID: uuid.New(), Score: 0.20}}, DefaultMinimumSimilarity, 1, 10)

	if page.Total != 0 || len(page.Items) != 0 || page.TotalPages != 0 {
		t.Fatalf("expected empty page, got %#v", page)
	}
}
