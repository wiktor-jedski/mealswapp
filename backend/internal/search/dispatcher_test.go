package search

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// Implements DESIGN-002 SearchController dispatcher verification.

type fakeDispatcherSearcher struct {
	response SearchResponse
	request  SearchRequest
	calls    int
}

func (s *fakeDispatcherSearcher) Search(_ context.Context, req SearchRequest) (SearchResponse, error) {
	s.calls++
	s.request = req
	return s.response, nil
}

func TestSearchDispatcherRoutesSubstitutionModeToSubstitutionService(t *testing.T) {
	catalog := &fakeDispatcherSearcher{}
	substitution := &fakeDispatcherSearcher{response: SearchResponse{
		Items:            []repository.FoodItemEntity{{Name: "Soy Milk"}},
		TotalCount:       1,
		SimilarityScores: []float64{1},
		SimilarityMetadata: []SimilarityMetadata{{
			Tier: SimilarityTierExcellent,
		}},
	}}
	req := SearchRequest{
		Query: "milk",
		Mode:  SearchModeSubstitution,
		Page:  1,
		SubstitutionInputs: []SubstitutionInput{{
			FoodObjectID: mustUUID("60000000-0000-4000-8000-000000000001"),
			Quantity:     100,
			Unit:         "g",
		}},
	}

	response, err := NewSearchDispatcher(catalog, substitution).Search(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if catalog.calls != 0 || substitution.calls != 1 {
		t.Fatalf("dispatcher calls catalog=%d substitution=%d", catalog.calls, substitution.calls)
	}
	if substitution.request.SubstitutionInputs[0].FoodObjectID != req.SubstitutionInputs[0].FoodObjectID {
		t.Fatalf("substitution request = %+v", substitution.request)
	}
	if len(response.SimilarityMetadata) != 1 || response.SimilarityMetadata[0].Tier != SimilarityTierExcellent {
		t.Fatalf("response metadata = %+v", response.SimilarityMetadata)
	}
}

func TestSearchDispatcherRoutesCatalogModeToCatalogServiceEvenWithSubstitutionInputs(t *testing.T) {
	catalog := &fakeDispatcherSearcher{response: SearchResponse{
		Items:      []repository.FoodItemEntity{{Name: "Milk"}},
		TotalCount: 1,
	}}
	substitution := &fakeDispatcherSearcher{}
	req := SearchRequest{
		Query: "milk",
		Mode:  SearchModeCatalog,
		Page:  1,
		SubstitutionInputs: []SubstitutionInput{{
			FoodObjectID: mustUUID("60000000-0000-4000-8000-000000000001"),
			Quantity:     100,
			Unit:         "g",
		}},
	}

	response, err := NewSearchDispatcher(catalog, substitution).Search(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if catalog.calls != 1 || substitution.calls != 0 {
		t.Fatalf("dispatcher calls catalog=%d substitution=%d", catalog.calls, substitution.calls)
	}
	if catalog.request.SubstitutionInputs[0].FoodObjectID != req.SubstitutionInputs[0].FoodObjectID {
		t.Fatalf("catalog request = %+v", catalog.request)
	}
	if len(response.Items) != 1 || response.Items[0].Name != "Milk" {
		t.Fatalf("response = %+v", response)
	}
}

func TestSearchDispatcherPreservesCatalogAndDailyDietDispatch(t *testing.T) {
	catalog := &fakeDispatcherSearcher{}
	substitution := &fakeDispatcherSearcher{}
	dispatcher := NewSearchDispatcher(catalog, substitution)

	if _, err := dispatcher.Search(context.Background(), SearchRequest{Query: "apple", Mode: SearchModeCatalog, Page: 1}); err != nil {
		t.Fatal(err)
	}
	dailyDietID := mustUUID("70000000-0000-4000-8000-000000000001")
	if _, err := dispatcher.Search(context.Background(), SearchRequest{Query: "lentil", Mode: SearchModeDailyDietAlternative, Page: 1, DailyDietID: &dailyDietID}); err != nil {
		t.Fatal(err)
	}

	if catalog.calls != 2 || substitution.calls != 0 {
		t.Fatalf("dispatcher calls catalog=%d substitution=%d", catalog.calls, substitution.calls)
	}
}

func TestSearchDispatcherRejectsInvalidRequestBeforeDispatch(t *testing.T) {
	catalog := &fakeDispatcherSearcher{}
	substitution := &fakeDispatcherSearcher{}
	if _, err := NewSearchDispatcher(catalog, substitution).Search(context.Background(), SearchRequest{Mode: SearchModeCatalog, Page: 1}); err == nil {
		t.Fatal("Search() accepted an empty query")
	}
	if catalog.calls != 0 || substitution.calls != 0 {
		t.Fatalf("invalid request dispatched catalog=%d substitution=%d", catalog.calls, substitution.calls)
	}
}

func mustUUID(value string) uuid.UUID {
	id, err := uuid.Parse(value)
	if err != nil {
		panic(err)
	}
	return id
}
