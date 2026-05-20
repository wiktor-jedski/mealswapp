package externaldata

import (
	"context"
	"errors"
	"testing"

	"mealswapp/backend/internal/cache"
	"mealswapp/backend/internal/domain/food"
	"mealswapp/backend/internal/repositories"

	"github.com/google/uuid"
)

func TestItemCuratorCRUDAndInvalidation(t *testing.T) {
	store := &fakeCuratorFoodStore{}
	invalidator := &fakeCacheInvalidator{}
	curator := NewItemCurator(store, invalidator)

	created, err := curator.Create(context.Background(), validCuratorFood("Draft Tofu"))
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}
	if created.ID == uuid.Nil || created.Source.CurationState != "draft" {
		t.Fatalf("unexpected created item: %#v", created)
	}

	patch := validCuratorFood("Edited Tofu")
	patch.ID = uuid.New()
	patch.Source.Provider = "openfoodfacts"
	patch.Source.ExternalID = "737628064502"
	updated, err := curator.Update(context.Background(), created.ID, patch)
	if err != nil {
		t.Fatalf("unexpected update error: %v", err)
	}
	if updated.Name != "Edited Tofu" || updated.Source.Provider != "openfoodfacts" {
		t.Fatalf("unexpected update: %#v", updated)
	}

	if err := curator.Delete(context.Background(), created.ID); err != nil {
		t.Fatalf("unexpected delete error: %v", err)
	}
	if len(invalidator.events) != 3 {
		t.Fatalf("expected create/update/delete invalidations, got %#v", invalidator.events)
	}
	for _, event := range invalidator.events {
		if event.Reason != cache.ReasonFoodUpdated || event.ItemIDs[0] != created.ID {
			t.Fatalf("unexpected invalidation event: %#v", event)
		}
	}
}

func TestItemCuratorTransitionsCurationState(t *testing.T) {
	store := &fakeCuratorFoodStore{}
	curator := NewItemCurator(store, nil)
	created, err := curator.Create(context.Background(), validCuratorFood("Draft Tofu"))
	if err != nil {
		t.Fatal(err)
	}

	approved, err := curator.Transition(context.Background(), created.ID, TransitionApprove)
	if err != nil {
		t.Fatalf("unexpected approve error: %v", err)
	}
	if approved.Source.CurationState != "approved" {
		t.Fatalf("expected approved, got %#v", approved.Source.CurationState)
	}
	rejected, err := curator.Transition(context.Background(), created.ID, TransitionReject)
	if err != nil {
		t.Fatalf("unexpected reject error: %v", err)
	}
	if rejected.Source.CurationState != "rejected" {
		t.Fatalf("expected rejected, got %#v", rejected.Source.CurationState)
	}
	inactive, err := curator.Transition(context.Background(), created.ID, TransitionDeactivate)
	if err != nil {
		t.Fatalf("unexpected deactivate error: %v", err)
	}
	if inactive.Source.CurationState != "inactive" {
		t.Fatalf("expected inactive, got %#v", inactive.Source.CurationState)
	}

	_, err = curator.Transition(context.Background(), created.ID, "archive")
	if !errors.Is(err, ErrItemCuratorInvalidTransition) {
		t.Fatalf("expected invalid transition, got %v", err)
	}
}

func TestItemCuratorSearchVisibilityHidesRejectedAndInactiveItems(t *testing.T) {
	store := &fakeCuratorFoodStore{}
	curator := NewItemCurator(store, nil)
	visible, err := curator.Create(context.Background(), validCuratorFood("Visible Tofu"))
	if err != nil {
		t.Fatal(err)
	}
	rejected, err := curator.Create(context.Background(), validCuratorFood("Rejected Tofu"))
	if err != nil {
		t.Fatal(err)
	}
	inactive, err := curator.Create(context.Background(), validCuratorFood("Inactive Tofu"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := curator.Transition(context.Background(), visible.ID, TransitionApprove); err != nil {
		t.Fatal(err)
	}
	if _, err := curator.Transition(context.Background(), rejected.ID, TransitionReject); err != nil {
		t.Fatal(err)
	}
	if _, err := curator.Transition(context.Background(), inactive.ID, TransitionDeactivate); err != nil {
		t.Fatal(err)
	}

	result, err := curator.List(context.Background(), "tofu", 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 1 || result.Items[0].ID != visible.ID {
		t.Fatalf("expected only approved/draft items in search visibility, got %#v", result.Items)
	}
}

func TestItemCuratorListUsesSearchPagination(t *testing.T) {
	store := &fakeCuratorFoodStore{items: []food.FoodItemEntity{validCuratorFood("Tofu")}}
	curator := NewItemCurator(store, nil)

	result, err := curator.List(context.Background(), "tof", 2, 5)
	if err != nil {
		t.Fatalf("unexpected list error: %v", err)
	}
	if result.Total != 1 || result.Page != 2 || result.Limit != 5 {
		t.Fatalf("unexpected result: %#v", result)
	}
	if store.lastQuery.Text != "tof" || store.lastQuery.Limit != 5 || store.lastQuery.Offset != 5 {
		t.Fatalf("unexpected repository query: %#v", store.lastQuery)
	}
}

func validCuratorFood(name string) food.FoodItemEntity {
	return food.FoodItemEntity{
		Name:           name,
		PhysicalState:  food.PhysicalStateSolid,
		ServingUnit:    food.ServingUnitGram,
		ServingSize:    100,
		CaloriesPer100: 120,
		MacrosPer100:   food.MacroValues{ProteinGrams: 12, CarbsGrams: 2, FatGrams: 6},
		Micros:         map[string]float64{},
		ImageURL:       "https://example.test/tofu.jpg",
		Source:         food.SourceMetadata{CurationState: "draft"},
	}
}

type fakeCuratorFoodStore struct {
	items     []food.FoodItemEntity
	lastQuery repositories.FoodItemQuery
}

func (store *fakeCuratorFoodStore) Create(ctx context.Context, item food.FoodItemEntity) (uuid.UUID, error) {
	id := uuid.New()
	item.ID = id
	store.items = append(store.items, item)
	return id, nil
}

func (store *fakeCuratorFoodStore) GetByID(ctx context.Context, id uuid.UUID, rc repositories.RepositoryContext) (food.FoodItemEntity, error) {
	for _, item := range store.items {
		if item.ID == id {
			return item, nil
		}
	}
	return food.FoodItemEntity{}, errors.New("not found")
}

func (store *fakeCuratorFoodStore) Search(ctx context.Context, query repositories.FoodItemQuery) ([]food.FoodItemEntity, int, error) {
	store.lastQuery = query
	items := make([]food.FoodItemEntity, 0, len(store.items))
	for _, item := range store.items {
		if item.Source.CurationState == "rejected" || item.Source.CurationState == "inactive" {
			continue
		}
		items = append(items, item)
	}
	return items, len(items), nil
}

func (store *fakeCuratorFoodStore) Update(ctx context.Context, item food.FoodItemEntity) error {
	for index := range store.items {
		if store.items[index].ID == item.ID {
			store.items[index] = item
			return nil
		}
	}
	return errors.New("not found")
}

func (store *fakeCuratorFoodStore) Delete(ctx context.Context, id uuid.UUID) error {
	for index := range store.items {
		if store.items[index].ID == id {
			store.items = append(store.items[:index], store.items[index+1:]...)
			return nil
		}
	}
	return errors.New("not found")
}
