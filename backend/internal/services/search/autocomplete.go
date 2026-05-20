package search

import (
	"sort"
	"strings"
)

const defaultAutocompleteLimit = 10

type AutocompleteCandidate struct {
	ItemID string
	Label  string
}

type RankedAutocomplete struct {
	ItemID              string `json:"itemId"`
	Label               string `json:"label"`
	ExactMatch          bool   `json:"exactMatch"`
	LevenshteinDistance int    `json:"levenshteinDistance"`
	Length              int    `json:"length"`
	Rank                int    `json:"rank"`
}

func RankAutocomplete(query string, candidates []AutocompleteCandidate) []RankedAutocomplete {
	return RankAutocompleteLimit(query, candidates, defaultAutocompleteLimit)
}

func RankAutocompleteLimit(query string, candidates []AutocompleteCandidate, limit int) []RankedAutocomplete {
	normalizedQuery := normalizeAutocompleteText(query)
	if normalizedQuery == "" || limit <= 0 || len(candidates) == 0 {
		return []RankedAutocomplete{}
	}

	ranked := make([]RankedAutocomplete, 0, len(candidates))
	for _, candidate := range candidates {
		label := strings.TrimSpace(candidate.Label)
		if label == "" {
			continue
		}

		normalizedLabel := normalizeAutocompleteText(label)
		ranked = append(ranked, RankedAutocomplete{
			ItemID:              candidate.ItemID,
			Label:               label,
			ExactMatch:          normalizedLabel == normalizedQuery,
			LevenshteinDistance: levenshteinDistance(normalizedQuery, normalizedLabel),
			Length:              len([]rune(normalizedLabel)),
		})
	}

	sort.SliceStable(ranked, func(i, j int) bool {
		left := ranked[i]
		right := ranked[j]
		if left.ExactMatch != right.ExactMatch {
			return left.ExactMatch
		}
		if left.LevenshteinDistance != right.LevenshteinDistance {
			return left.LevenshteinDistance < right.LevenshteinDistance
		}
		if left.Length != right.Length {
			return left.Length < right.Length
		}
		leftLabel := normalizeAutocompleteText(left.Label)
		rightLabel := normalizeAutocompleteText(right.Label)
		if leftLabel != rightLabel {
			return leftLabel < rightLabel
		}
		return left.ItemID < right.ItemID
	})

	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	for i := range ranked {
		ranked[i].Rank = i + 1
	}
	return ranked
}

func normalizeAutocompleteText(value string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(value))), " ")
}

func levenshteinDistance(left string, right string) int {
	leftRunes := []rune(left)
	rightRunes := []rune(right)
	if len(leftRunes) == 0 {
		return len(rightRunes)
	}
	if len(rightRunes) == 0 {
		return len(leftRunes)
	}

	previous := make([]int, len(rightRunes)+1)
	current := make([]int, len(rightRunes)+1)
	for i := range previous {
		previous[i] = i
	}

	for i, leftRune := range leftRunes {
		current[0] = i + 1
		for j, rightRune := range rightRunes {
			cost := 1
			if leftRune == rightRune {
				cost = 0
			}
			current[j+1] = minInt(
				current[j]+1,
				previous[j+1]+1,
				previous[j]+cost,
			)
		}
		previous, current = current, previous
	}

	return previous[len(rightRunes)]
}

func minInt(first int, rest ...int) int {
	value := first
	for _, candidate := range rest {
		if candidate < value {
			value = candidate
		}
	}
	return value
}
