package similarity

import (
	"sort"

	"github.com/google/uuid"
)

type Candidate struct {
	ID     uuid.UUID
	Vector MacroVector
}

type ScoredCandidate struct {
	ID         uuid.UUID
	Score      float64
	Normalized NormalizedMacroVector
}

func CosineSimilarity(a NormalizedMacroVector, b NormalizedMacroVector) float64 {
	return a.Protein*b.Protein + a.Carbs*b.Carbs + a.Fat*b.Fat
}

func ScoreCandidates(source MacroVector, candidates []Candidate) ([]ScoredCandidate, error) {
	normalizedSource, err := NormalizeMacroVector(source)
	if err != nil {
		return nil, err
	}

	scored := make([]ScoredCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		normalizedCandidate, err := NormalizeMacroVector(candidate.Vector)
		if err != nil {
			if err == ErrZeroMacroVector {
				continue
			}
			return nil, err
		}
		scored = append(scored, ScoredCandidate{
			ID:         candidate.ID,
			Score:      CosineSimilarity(normalizedSource, normalizedCandidate),
			Normalized: normalizedCandidate,
		})
	}

	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].Score == scored[j].Score {
			return scored[i].ID.String() < scored[j].ID.String()
		}
		return scored[i].Score > scored[j].Score
	})

	return scored, nil
}
