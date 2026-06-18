package search

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// Implements DESIGN-002 AutocompleteRanker verification.

func TestRankAutocompleteOrdersByExactDistanceLengthAndStableTieBreakers(t *testing.T) {
	candidates := []AutocompleteCandidate{
		{ItemID: uuid.MustParse("00000000-0000-0000-0000-000000000004"), Label: "ax"},
		{ItemID: uuid.MustParse("00000000-0000-0000-0000-000000000003"), Label: "abx"},
		{ItemID: uuid.MustParse("00000000-0000-0000-0000-000000000005"), Label: "abcc"},
		{ItemID: uuid.MustParse("00000000-0000-0000-0000-000000000001"), Label: "abc"},
		{ItemID: uuid.MustParse("00000000-0000-0000-0000-000000000002"), Label: "ABC"},
	}

	ranked := RankAutocomplete("abc", candidates, PageSize)

	got := labels(ranked)
	want := []string{"ABC", "abc", "abx", "abcc", "ax"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ranked labels = %#v, want %#v", got, want)
	}
	for i, item := range ranked {
		if item.Rank != i+1 {
			t.Fatalf("ranked[%d].Rank = %d", i, item.Rank)
		}
	}
	if !ranked[0].ExactMatch || !ranked[1].ExactMatch {
		t.Fatalf("exact matches did not rank first: %#v", ranked[:2])
	}
	if ranked[3].Length <= ranked[4].Length || ranked[3].LevenshteinDistance >= ranked[4].LevenshteinDistance {
		t.Fatalf("Levenshtein distance did not rank before length: %#v", ranked[3:])
	}
	if ranked[2].LevenshteinDistance != ranked[3].LevenshteinDistance || ranked[2].Length >= ranked[3].Length {
		t.Fatalf("shorter equal-distance label did not rank first: %#v", ranked[2:4])
	}
}

func TestAutocompleteServiceRetrievesFoodAndMealCandidatesWithBoundedQueries(t *testing.T) {
	ctx := context.Background()
	foodRepo := &autocompleteFoodRepo{
		items: []repository.FoodItemEntity{
			{ID: uuid.MustParse("10000000-0000-0000-0000-000000000001"), Name: "Pear"},
			{ID: uuid.MustParse("10000000-0000-0000-0000-000000000002"), Name: "Peach"},
		},
	}
	mealRepo := &autocompleteMealRepo{
		items: []repository.MealEntity{
			{ID: uuid.MustParse("20000000-0000-0000-0000-000000000001"), Name: "Pear Salad"},
		},
	}

	service := NewAutocompleteService(foodRepo, mealRepo)
	ranked, err := service.Autocomplete(ctx, "  PEar  ", repository.RepositoryContext{IncludeDeleted: true, UnitSystem: repository.UnitSystemImperial})
	if err != nil {
		t.Fatal(err)
	}

	if got := labels(ranked); !reflect.DeepEqual(got, []string{"Pear", "Peach", "Pear Salad"}) {
		t.Fatalf("labels = %#v", got)
	}
	for _, call := range []repository.RepositoryQuery{foodRepo.calls[0], mealRepo.calls[0]} {
		if call.Name != "pear" {
			t.Fatalf("repository query name = %q", call.Name)
		}
		if call.Limit != autocompleteCandidateLimit || call.Offset != 0 {
			t.Fatalf("repository query pagination = limit %d offset %d", call.Limit, call.Offset)
		}
		if call.IncludeDeleted {
			t.Fatalf("autocomplete should not include deleted rows")
		}
		if call.UnitSystem != repository.UnitSystemImperial {
			t.Fatalf("repository query unit system = %q", call.UnitSystem)
		}
	}
}

func TestAutocompleteServiceKeepsSpecialCharactersParameterizedByRepository(t *testing.T) {
	ctx := context.Background()
	foodRepo := &autocompleteFoodRepo{}
	mealRepo := &autocompleteMealRepo{}
	service := NewAutocompleteService(foodRepo, mealRepo)

	query := " crème brûlée ' OR 1=1 -- "
	if _, err := service.Autocomplete(ctx, query, repository.RepositoryContext{}); err != nil {
		t.Fatal(err)
	}
	want := "crème brûlée ' or 1=1 --"
	if foodRepo.calls[0].Name != want || mealRepo.calls[0].Name != want {
		t.Fatalf("repository query names = food %q meal %q, want %q", foodRepo.calls[0].Name, mealRepo.calls[0].Name, want)
	}
}

func TestAutocompleteServiceIsDeterministicAndPageBounded(t *testing.T) {
	ctx := context.Background()
	foodItems := make([]repository.FoodItemEntity, 0, PageSize+4)
	for i := 0; i < PageSize+4; i++ {
		id := uuid.MustParse(fmt.Sprintf("30000000-0000-0000-0000-%012x", i+1))
		foodItems = append(foodItems, repository.FoodItemEntity{ID: id, Name: "Apple"})
	}
	foodRepo := &autocompleteFoodRepo{items: foodItems}
	mealRepo := &autocompleteMealRepo{items: []repository.MealEntity{
		{ID: uuid.MustParse("40000000-0000-0000-0000-000000000001"), Name: "Apple"},
	}}
	service := NewAutocompleteService(foodRepo, mealRepo)

	first, err := service.Autocomplete(ctx, "apple", repository.RepositoryContext{})
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Autocomplete(ctx, "apple", repository.RepositoryContext{})
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != PageSize {
		t.Fatalf("result length = %d, want %d", len(first), PageSize)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("repeated calls differ:\nfirst=%#v\nsecond=%#v", first, second)
	}
}

func TestAutocompleteServicePropagatesRepositoryErrors(t *testing.T) {
	ctx := context.Background()
	foodErr := errors.New("food unavailable")
	service := NewAutocompleteService(&autocompleteFoodRepo{err: foodErr}, &autocompleteMealRepo{})
	if _, err := service.Autocomplete(ctx, "pear", repository.RepositoryContext{}); !errors.Is(err, foodErr) {
		t.Fatalf("food error = %v, want %v", err, foodErr)
	}

	mealErr := errors.New("meal unavailable")
	service = NewAutocompleteService(&autocompleteFoodRepo{}, &autocompleteMealRepo{err: mealErr})
	if _, err := service.Autocomplete(ctx, "pear", repository.RepositoryContext{}); !errors.Is(err, mealErr) {
		t.Fatalf("meal error = %v, want %v", err, mealErr)
	}
}

func TestAutocompleteValidationLimitAndEmptyDistanceBoundaries(t *testing.T) {
	if _, err := NewAutocompleteService(nil, nil).Autocomplete(context.Background(), "", repository.RepositoryContext{}); err == nil {
		t.Fatal("Autocomplete() accepted an empty query")
	}
	candidates := []AutocompleteCandidate{{ItemID: uuid.New(), Label: "apple"}}
	if ranked := RankAutocomplete("apple", candidates, 0); len(ranked) != 1 || ranked[0].Rank != 1 {
		t.Fatalf("RankAutocomplete() = %+v", ranked)
	}
	if distance := levenshteinDistance("", "apple"); distance != 5 {
		t.Fatalf("empty-left distance = %d", distance)
	}
	if distance := levenshteinDistance("apple", ""); distance != 5 {
		t.Fatalf("empty-right distance = %d", distance)
	}
}

func labels(items []RankedAutocomplete) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.Label)
	}
	return out
}

type autocompleteFoodRepo struct {
	items []repository.FoodItemEntity
	calls []repository.RepositoryQuery
	err   error
}

func (r *autocompleteFoodRepo) GetByID(context.Context, uuid.UUID, repository.RepositoryContext) (repository.FoodItemEntity, error) {
	return repository.FoodItemEntity{}, nil
}

func (r *autocompleteFoodRepo) Search(_ context.Context, q repository.RepositoryQuery) ([]repository.FoodItemEntity, int, error) {
	r.calls = append(r.calls, q)
	return r.items, len(r.items), r.err
}

func (r *autocompleteFoodRepo) Create(context.Context, repository.FoodItemEntity) (uuid.UUID, error) {
	return uuid.Nil, nil
}

func (r *autocompleteFoodRepo) Update(context.Context, repository.FoodItemEntity) error {
	return nil
}

func (r *autocompleteFoodRepo) Delete(context.Context, uuid.UUID) error {
	return nil
}

type autocompleteMealRepo struct {
	items []repository.MealEntity
	calls []repository.RepositoryQuery
	err   error
}

func (r *autocompleteMealRepo) GetByID(context.Context, uuid.UUID, repository.RepositoryContext) (repository.MealEntity, error) {
	return repository.MealEntity{}, nil
}

func (r *autocompleteMealRepo) Search(_ context.Context, q repository.RepositoryQuery) ([]repository.MealEntity, int, error) {
	r.calls = append(r.calls, q)
	return r.items, len(r.items), r.err
}

func (r *autocompleteMealRepo) CalculateMacros(context.Context, uuid.UUID) (repository.MacroValues, error) {
	return repository.MacroValues{}, nil
}

func (r *autocompleteMealRepo) Create(context.Context, repository.MealEntity) (uuid.UUID, error) {
	return uuid.Nil, nil
}

func (r *autocompleteMealRepo) Update(context.Context, repository.MealEntity) error {
	return nil
}

func (r *autocompleteMealRepo) Delete(context.Context, uuid.UUID) error {
	return nil
}
