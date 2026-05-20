package cache

import (
	"context"
	"slices"
	"testing"

	"github.com/google/uuid"
)

func TestInvalidatorDeletesSearchAndSimilarityForFoodUpdates(t *testing.T) {
	store := newMemoryTaggedStore()
	itemID := uuid.MustParse("a0000000-0000-0000-0000-000000000001")
	searchKey := BuildRawIDKey(NamespaceSearch, "query-1", "")
	similarityKey := BuildRawIDKey(NamespaceSimilarity, "similarity-1", "")
	itemKey := BuildRawIDKey(NamespaceItem, itemID.String(), "")

	store.Set(searchKey, "old search", []string{"namespace:search", "item:" + itemID.String()})
	store.Set(similarityKey, "old similarity", []string{"namespace:similarity", "item:" + itemID.String()})
	store.Set(itemKey, "old item", []string{"namespace:item", "item:" + itemID.String()})

	result, err := NewInvalidator(store).Invalidate(context.Background(), InvalidationEvent{
		Reason:  ReasonFoodUpdated,
		ItemIDs: []uuid.UUID{itemID},
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.KeysDeleted != 3 {
		t.Fatalf("expected three affected keys deleted, got %#v", result)
	}
	if store.Has(searchKey) || store.Has(similarityKey) || store.Has(itemKey) {
		t.Fatalf("expected stale item/search/similarity entries removed, got %#v", store.values)
	}
}

func TestInvalidatorDeletesSearchForTagUpdates(t *testing.T) {
	store := newMemoryTaggedStore()
	tagID := uuid.MustParse("b0000000-0000-0000-0000-000000000001")
	searchKey := BuildRawIDKey(NamespaceSearch, "vegan-query", "")
	unrelatedKey := BuildRawIDKey(NamespaceItem, "food-1", "")

	store.Set(searchKey, "old vegan search", []string{"namespace:search", "tag:" + tagID.String()})
	store.Set(unrelatedKey, "food", []string{"namespace:item"})

	result, err := NewInvalidator(store).Invalidate(context.Background(), InvalidationEvent{
		Reason: ReasonTagUpdated,
		TagIDs: []uuid.UUID{tagID},
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.KeysDeleted != 1 {
		t.Fatalf("expected one tag-affected key deleted, got %#v", result)
	}
	if store.Has(searchKey) {
		t.Fatal("expected stale tagged search result removed")
	}
	if !store.Has(unrelatedKey) {
		t.Fatal("expected unrelated item cache to remain")
	}
}

func TestInvalidationTagsAreDeterministicAndDeduplicated(t *testing.T) {
	itemID := uuid.MustParse("c0000000-0000-0000-0000-000000000001")
	tagID := uuid.MustParse("c0000000-0000-0000-0000-000000000002")
	userID := uuid.MustParse("c0000000-0000-0000-0000-000000000003")

	tags := InvalidationTags(InvalidationEvent{
		Reason:  ReasonImportChanged,
		ItemIDs: []uuid.UUID{itemID, itemID},
		TagIDs:  []uuid.UUID{tagID, tagID},
		UserID:  &userID,
	})

	expected := []string{
		"namespace:search",
		"namespace:similarity",
		"item:" + itemID.String(),
		"tag:" + tagID.String(),
		"user:" + userID.String(),
	}
	if !slices.Equal(tags, expected) {
		t.Fatalf("expected tags %v, got %v", expected, tags)
	}
}

func TestInvalidatorNoopsWhenNoTagsAreDerived(t *testing.T) {
	store := newMemoryTaggedStore()
	result, err := NewInvalidator(store).Invalidate(context.Background(), InvalidationEvent{})
	if err != nil {
		t.Fatal(err)
	}
	if result.KeysDeleted != 0 || len(result.TagsDeleted) != 0 || store.deleteCalls != 0 {
		t.Fatalf("expected noop invalidation, got result=%#v calls=%d", result, store.deleteCalls)
	}
}

type memoryTaggedStore struct {
	values      map[string]string
	tagsByKey   map[string][]string
	deleteCalls int
}

func newMemoryTaggedStore() *memoryTaggedStore {
	return &memoryTaggedStore{
		values:    map[string]string{},
		tagsByKey: map[string][]string{},
	}
}

func (store *memoryTaggedStore) Set(key RedisCacheKey, value string, tags []string) {
	rendered := key.String()
	store.values[rendered] = value
	store.tagsByKey[rendered] = tags
}

func (store *memoryTaggedStore) Has(key RedisCacheKey) bool {
	_, ok := store.values[key.String()]
	return ok
}

func (store *memoryTaggedStore) DeleteByTags(ctx context.Context, tags []string) (int, error) {
	store.deleteCalls++
	deleted := 0
	for key, keyTags := range store.tagsByKey {
		if intersects(keyTags, tags) {
			delete(store.tagsByKey, key)
			delete(store.values, key)
			deleted++
		}
	}
	return deleted, nil
}

func intersects(left []string, right []string) bool {
	for _, l := range left {
		for _, r := range right {
			if l == r {
				return true
			}
		}
	}
	return false
}
