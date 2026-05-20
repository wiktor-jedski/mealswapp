package externaldata

import (
	"context"
	"errors"
	"testing"

	"mealswapp/backend/internal/cache"
	"mealswapp/backend/internal/domain/tag"

	"github.com/google/uuid"
)

func TestTagManagerCRUDAssignmentMergeAndInvalidation(t *testing.T) {
	store := &fakeTagStore{}
	invalidator := &fakeCacheInvalidator{}
	manager := NewTagManager(store, invalidator)

	vegan, err := manager.Upsert(context.Background(), tag.TagEntity{Name: " Vegan ", Kind: tag.KindDiet})
	if err != nil {
		t.Fatalf("unexpected upsert error: %v", err)
	}
	if vegan.ID == uuid.Nil || vegan.Name != "Vegan" || !vegan.Active {
		t.Fatalf("unexpected tag: %#v", vegan)
	}

	foodID := uuid.New()
	if err := manager.Assign(context.Background(), foodID, vegan.ID); err != nil {
		t.Fatalf("unexpected assign error: %v", err)
	}
	if err := manager.Remove(context.Background(), foodID, vegan.ID); err != nil {
		t.Fatalf("unexpected remove error: %v", err)
	}
	target, err := manager.Upsert(context.Background(), tag.TagEntity{Name: "Plant based", Kind: tag.KindDiet})
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.Merge(context.Background(), vegan.ID, target.ID); err != nil {
		t.Fatalf("unexpected merge error: %v", err)
	}
	if store.merged[vegan.ID].targetID != target.ID {
		t.Fatalf("expected merge source to target, got %#v", store.merged)
	}
	if err := manager.Deactivate(context.Background(), target.ID); err != nil {
		t.Fatalf("unexpected deactivate error: %v", err)
	}
	if store.tags[target.ID].Active {
		t.Fatalf("expected target deactivated")
	}
	if len(invalidator.events) < 5 {
		t.Fatalf("expected tag invalidation events, got %#v", invalidator.events)
	}
	for _, event := range invalidator.events {
		if event.Reason != cache.ReasonTagUpdated {
			t.Fatalf("unexpected invalidation reason: %#v", event)
		}
	}
}

func TestTagManagerRejectsInvalidTaxonomyAndMerge(t *testing.T) {
	manager := NewTagManager(&fakeTagStore{}, nil)
	if _, err := manager.Upsert(context.Background(), tag.TagEntity{Name: "Category", Kind: "category"}); !errors.Is(err, tag.ErrInvalidKind) {
		t.Fatalf("expected invalid kind, got %v", err)
	}
	id := uuid.New()
	if err := manager.Merge(context.Background(), id, id); !errors.Is(err, ErrTagMergeInvalid) {
		t.Fatalf("expected invalid merge, got %v", err)
	}
}

type fakeTagStore struct {
	tags   map[uuid.UUID]tag.TagEntity
	links  map[uuid.UUID][]uuid.UUID
	merged map[uuid.UUID]struct{ targetID uuid.UUID }
}

func (store *fakeTagStore) ensure() {
	if store.tags == nil {
		store.tags = map[uuid.UUID]tag.TagEntity{}
		store.links = map[uuid.UUID][]uuid.UUID{}
		store.merged = map[uuid.UUID]struct{ targetID uuid.UUID }{}
	}
}

func (store *fakeTagStore) List(ctx context.Context, kind tag.Kind) ([]tag.TagEntity, error) {
	store.ensure()
	tags := []tag.TagEntity{}
	for _, entity := range store.tags {
		if entity.Kind == kind {
			tags = append(tags, entity)
		}
	}
	return tags, nil
}

func (store *fakeTagStore) Upsert(ctx context.Context, entity tag.TagEntity) (uuid.UUID, error) {
	store.ensure()
	if entity.ID == uuid.Nil {
		entity.ID = uuid.New()
	}
	store.tags[entity.ID] = entity
	return entity.ID, nil
}

func (store *fakeTagStore) AttachToFoodItem(ctx context.Context, foodItemID uuid.UUID, tagID uuid.UUID) error {
	store.ensure()
	store.links[foodItemID] = append(store.links[foodItemID], tagID)
	return nil
}

func (store *fakeTagStore) RemoveFromFoodItem(ctx context.Context, foodItemID uuid.UUID, tagID uuid.UUID) error {
	store.ensure()
	store.links[foodItemID] = nil
	return nil
}

func (store *fakeTagStore) Deactivate(ctx context.Context, id uuid.UUID) error {
	store.ensure()
	entity := store.tags[id]
	entity.Active = false
	store.tags[id] = entity
	return nil
}

func (store *fakeTagStore) Merge(ctx context.Context, sourceID uuid.UUID, targetID uuid.UUID) error {
	store.ensure()
	store.merged[sourceID] = struct{ targetID uuid.UUID }{targetID: targetID}
	return store.Deactivate(ctx, sourceID)
}
