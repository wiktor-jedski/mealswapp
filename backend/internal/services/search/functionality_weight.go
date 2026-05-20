package search

import (
	"sort"

	"github.com/google/uuid"
)

const functionalityTagBoost = 0.2

type SearchCandidate struct {
	ItemID            uuid.UUID
	TextScore         float64
	SimilarityScore   float64
	FunctionalityTags []uuid.UUID
	TagMatchCount     int
	FinalScore        float64
}

func ApplyFunctionalityWeight(candidates []SearchCandidate, sourceTags []uuid.UUID) []SearchCandidate {
	weighted := make([]SearchCandidate, 0, len(candidates))
	sourceSet := uuidSet(sourceTags)

	for _, candidate := range candidates {
		candidate.TagMatchCount = countSharedTags(candidate.FunctionalityTags, sourceSet)
		candidate.FinalScore = candidate.SimilarityScore * (1 + functionalityTagBoost*float64(candidate.TagMatchCount))
		weighted = append(weighted, candidate)
	}

	sort.SliceStable(weighted, func(i, j int) bool {
		left := weighted[i]
		right := weighted[j]
		if left.FinalScore != right.FinalScore {
			return left.FinalScore > right.FinalScore
		}
		if left.SimilarityScore != right.SimilarityScore {
			return left.SimilarityScore > right.SimilarityScore
		}
		if left.TagMatchCount != right.TagMatchCount {
			return left.TagMatchCount > right.TagMatchCount
		}
		return left.ItemID.String() < right.ItemID.String()
	})

	return weighted
}

func uuidSet(values []uuid.UUID) map[uuid.UUID]struct{} {
	set := make(map[uuid.UUID]struct{}, len(values))
	for _, value := range values {
		if value != uuid.Nil {
			set[value] = struct{}{}
		}
	}
	return set
}

func countSharedTags(candidateTags []uuid.UUID, sourceSet map[uuid.UUID]struct{}) int {
	seen := make(map[uuid.UUID]struct{}, len(candidateTags))
	count := 0
	for _, candidateTag := range candidateTags {
		if candidateTag == uuid.Nil {
			continue
		}
		if _, duplicate := seen[candidateTag]; duplicate {
			continue
		}
		seen[candidateTag] = struct{}{}
		if _, ok := sourceSet[candidateTag]; ok {
			count++
		}
	}
	return count
}
