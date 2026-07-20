package search

import (
	"context"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/security"
)

// AutocompleteCandidate identifies one food or meal name available for ranking.
// Implements DESIGN-002 AutocompleteRanker.
type AutocompleteCandidate struct {
	ItemID     uuid.UUID
	Label      string
	ObjectType repository.FoodObjectType
}

// RankedAutocomplete carries deterministic autocomplete ranking metadata.
// Implements DESIGN-002 AutocompleteRanker.
type RankedAutocomplete struct {
	ItemID              string
	Label               string
	ExactMatch          bool
	LevenshteinDistance int
	Length              int
	Rank                int
	ObjectType          repository.FoodObjectType
}

// AutocompleteService retrieves food and meal candidates and ranks them.
// Implements DESIGN-002 AutocompleteRanker.
type AutocompleteService struct {
	foods repository.FoodItemRepository
	meals repository.MealRepository
}

// Implements DESIGN-002 AutocompleteRanker candidate window.
const autocompleteCandidateLimit = PageSize * 3

// NewAutocompleteService creates a repository-backed autocomplete service.
// Implements DESIGN-002 AutocompleteRanker.
func NewAutocompleteService(foods repository.FoodItemRepository, meals repository.MealRepository) AutocompleteService {
	return AutocompleteService{foods: foods, meals: meals}
}

// Autocomplete retrieves active food and meal candidates and returns a bounded ranked page.
// Implements DESIGN-002 AutocompleteRanker.
func (s AutocompleteService) Autocomplete(ctx context.Context, query string, rc repository.RepositoryContext) ([]RankedAutocomplete, error) {
	normalized, err := security.NormalizeInput(security.InputFieldAutocompleteQuery, query)
	if err != nil {
		return nil, err
	}

	candidates := make([]AutocompleteCandidate, 0, autocompleteCandidateLimit)
	seen := map[uuid.UUID]struct{}{}
	for _, candidateQuery := range autocompleteCandidateQueries(normalized.Value) {
		repoQuery := repository.RepositoryQuery{
			RepositoryContext: repository.RepositoryContext{
				UserID:     rc.UserID,
				UnitSystem: rc.UnitSystem,
			},
			Name:   candidateQuery,
			Limit:  autocompleteCandidateLimit,
			Offset: 0,
		}

		foods, _, err := s.foods.Search(ctx, repoQuery)
		if err != nil {
			return nil, err
		}
		meals, _, err := s.meals.Search(ctx, repoQuery)
		if err != nil {
			return nil, err
		}

		for _, food := range foods {
			if _, ok := seen[food.ID]; ok {
				continue
			}
			seen[food.ID] = struct{}{}
			candidates = append(candidates, AutocompleteCandidate{ItemID: food.ID, Label: food.Name, ObjectType: repository.FoodObjectTypeFoodItem})
		}
		for _, meal := range meals {
			if _, ok := seen[meal.ID]; ok {
				continue
			}
			seen[meal.ID] = struct{}{}
			candidates = append(candidates, AutocompleteCandidate{ItemID: meal.ID, Label: meal.Name, ObjectType: repository.FoodObjectTypeMeal})
		}
	}
	return RankAutocomplete(normalized.Value, candidates, PageSize), nil
}

// autocompleteCandidateQueries derives exact and short-prefix repository probes.
// Implements DESIGN-002 AutocompleteRanker.
func autocompleteCandidateQueries(query string) []string {
	queries := []string{query}
	runes := []rune(query)
	if len(runes) > 2 {
		shortPrefix := string(runes[:2])
		if shortPrefix != query {
			queries = append(queries, shortPrefix)
		}
	}
	return queries
}

// RankAutocomplete sorts candidates by exact match, Levenshtein distance, length, and stable tie-breakers.
// Implements DESIGN-002 AutocompleteRanker.
func RankAutocomplete(query string, candidates []AutocompleteCandidate, limit int) []RankedAutocomplete {
	normalizedQuery := strings.ToLower(strings.Join(strings.Fields(query), " "))
	ranked := make([]RankedAutocomplete, 0, len(candidates))
	for _, candidate := range candidates {
		normalizedLabel := strings.ToLower(strings.Join(strings.Fields(candidate.Label), " "))
		ranked = append(ranked, RankedAutocomplete{
			ItemID:              candidate.ItemID.String(),
			Label:               candidate.Label,
			ExactMatch:          normalizedLabel == normalizedQuery,
			LevenshteinDistance: levenshteinDistance(normalizedQuery, normalizedLabel),
			Length:              len([]rune(candidate.Label)),
			ObjectType:          candidate.ObjectType,
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
		if left.Label != right.Label {
			return left.Label < right.Label
		}
		return left.ItemID < right.ItemID
	})

	if limit <= 0 || limit > PageSize {
		limit = PageSize
	}
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	for i := range ranked {
		ranked[i].Rank = i + 1
	}
	return ranked
}

// levenshteinDistance computes edit distance for autocomplete ranking.
// Implements DESIGN-002 AutocompleteRanker.
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
	for j := range previous {
		previous[j] = j
	}
	for i, leftRune := range leftRunes {
		current[0] = i + 1
		for j, rightRune := range rightRunes {
			cost := 0
			if leftRune != rightRune {
				cost = 1
			}
			current[j+1] = min(previous[j+1]+1, current[j]+1, previous[j]+cost)
		}
		previous, current = current, previous
	}
	return previous[len(rightRunes)]
}
