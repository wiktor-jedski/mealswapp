package cache

import (
	"strings"
	"testing"
	"time"
)

func TestBuildHashedPayloadKeyIsStable(t *testing.T) {
	payload := map[string]any{
		"query": "tofu",
		"mode":  "single",
		"page":  float64(1),
	}

	first, err := BuildSearchCacheKey(payload)
	if err != nil {
		t.Fatal(err)
	}
	second, err := BuildSearchCacheKey(payload)
	if err != nil {
		t.Fatal(err)
	}

	if first != second {
		t.Fatalf("expected stable search cache key, got %#v and %#v", first, second)
	}
	if got := first.String(); !strings.HasPrefix(got, "mealswapp:v1:search:") {
		t.Fatalf("unexpected key string %q", got)
	}
	if len(first.ID) != 64 {
		t.Fatalf("expected sha256 id length, got %q", first.ID)
	}
}

func TestCacheKeysSeparateNamespaces(t *testing.T) {
	id := "food-1"
	keys := []RedisCacheKey{
		BuildKey(NamespaceSearch, id, ""),
		BuildKey(NamespaceItem, id, ""),
		BuildKey(NamespaceSimilarity, id, ""),
		BuildKey(NamespaceSession, id, ""),
		BuildKey(NamespaceJob, id, ""),
		BuildKey(NamespaceUser, id, ""),
	}

	seen := map[string]bool{}
	for _, key := range keys {
		rendered := key.String()
		if seen[rendered] {
			t.Fatalf("expected namespace-separated key, duplicate %q", rendered)
		}
		seen[rendered] = true
	}
}

func TestCacheSchemaVersionInvalidatesKeyString(t *testing.T) {
	v1 := BuildKey(NamespaceItem, "food-1", "v1")
	v2 := BuildKey(NamespaceItem, "food-1", "v2")

	if v1.ID != v2.ID {
		t.Fatalf("schema version should not change stable id hash, got %#v and %#v", v1, v2)
	}
	if v1.String() == v2.String() {
		t.Fatalf("expected schema version to change redis key string, got %q", v1.String())
	}
}

func TestCacheTTLPolicy(t *testing.T) {
	cases := map[Namespace]time.Duration{
		NamespaceSearch:     5 * time.Minute,
		NamespaceItem:       30 * time.Minute,
		NamespaceSimilarity: 15 * time.Minute,
		NamespaceSession:    24 * time.Hour,
		NamespaceJob:        time.Hour,
		NamespaceUser:       10 * time.Minute,
	}

	for namespace, expected := range cases {
		if got := TTL(namespace); got != expected {
			t.Fatalf("expected %s TTL %s, got %s", namespace, expected, got)
		}
	}
}

func TestBuildRawIDKeyNormalizesReadableIDs(t *testing.T) {
	key := BuildRawIDKey(NamespaceSession, " Session ABC ", "")
	if key.ID != "session-abc" {
		t.Fatalf("expected normalized raw id, got %#v", key)
	}
	if key.Version != DefaultSchemaVersion {
		t.Fatalf("expected default schema version, got %#v", key)
	}
}

func TestTagsForKeyIncludesNamespaceAndExtraTags(t *testing.T) {
	key := BuildRawIDKey(NamespaceUser, "user-1", "")
	tags := TagsForKey(key, "user:user-1", " ")

	expected := []string{"namespace:user", "user:user-1"}
	if len(tags) != len(expected) {
		t.Fatalf("expected tags %v, got %v", expected, tags)
	}
	for i := range expected {
		if tags[i] != expected[i] {
			t.Fatalf("expected tags %v, got %v", expected, tags)
		}
	}
}
