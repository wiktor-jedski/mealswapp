package similarity

import (
	"math"
	"testing"

	"github.com/google/uuid"
)

func TestCosineSimilarityExactMatch(t *testing.T) {
	source, err := NormalizeMacroVector(MacroVector{Protein: 10, Carbs: 20, Fat: 5})
	if err != nil {
		t.Fatal(err)
	}
	target, err := NormalizeMacroVector(MacroVector{Protein: 10, Carbs: 20, Fat: 5})
	if err != nil {
		t.Fatal(err)
	}

	if score := CosineSimilarity(source, target); math.Abs(score-1) > 0.0001 {
		t.Fatalf("expected exact match score 1, got %f", score)
	}
}

func TestCosineSimilarityOrthogonalProfiles(t *testing.T) {
	protein, err := NormalizeMacroVector(MacroVector{Protein: 10})
	if err != nil {
		t.Fatal(err)
	}
	carbs, err := NormalizeMacroVector(MacroVector{Carbs: 10})
	if err != nil {
		t.Fatal(err)
	}

	if score := CosineSimilarity(protein, carbs); score != 0 {
		t.Fatalf("expected orthogonal score 0, got %f", score)
	}
}

func TestScoreCandidatesSortsDeterministically(t *testing.T) {
	source := MacroVector{Protein: 10, Carbs: 0, Fat: 0}
	lowID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	highID := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	tieLaterID := uuid.MustParse("00000000-0000-4000-8000-000000000003")

	scored, err := ScoreCandidates(source, []Candidate{
		{ID: tieLaterID, Vector: MacroVector{Protein: 5}},
		{ID: lowID, Vector: MacroVector{Protein: 5}},
		{ID: highID, Vector: MacroVector{Carbs: 5}},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(scored) != 3 {
		t.Fatalf("expected 3 scored candidates, got %d", len(scored))
	}
	if scored[0].ID != lowID || scored[1].ID != tieLaterID || scored[2].ID != highID {
		t.Fatalf("unexpected deterministic order: %#v", scored)
	}
}

func TestScoreCandidatesSkipsZeroTargets(t *testing.T) {
	scored, err := ScoreCandidates(MacroVector{Protein: 10}, []Candidate{
		{ID: uuid.New(), Vector: MacroVector{}},
		{ID: uuid.New(), Vector: MacroVector{Protein: 10}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(scored) != 1 {
		t.Fatalf("expected zero target skipped, got %#v", scored)
	}
}

func TestScoreCandidatesRejectsInvalidSource(t *testing.T) {
	_, err := ScoreCandidates(MacroVector{}, nil)
	if err != ErrZeroMacroVector {
		t.Fatalf("expected zero source vector error, got %v", err)
	}
}
