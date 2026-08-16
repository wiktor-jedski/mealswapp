package cache

// Implements DESIGN-009 TagManager and DESIGN-011 RedisCache shared-generation integration verification.

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/search"
)

type sharedFilterClassificationRepository struct {
	mu      sync.Mutex
	entries []repository.ClassificationEntity
}

func (r *sharedFilterClassificationRepository) List(_ context.Context, kind repository.ClassificationKind) ([]repository.ClassificationEntity, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	entries := make([]repository.ClassificationEntity, 0, len(r.entries))
	for _, entry := range r.entries {
		if entry.Kind == kind {
			entries = append(entries, entry)
		}
	}
	return entries, nil
}

type emptyFilterAllergenRepository struct{}

func (emptyFilterAllergenRepository) ListActive(context.Context) ([]repository.AllergenVocabularyEntry, error) {
	return []repository.AllergenVocabularyEntry{}, nil
}

// TestClassificationGenerationLiveRedisCoordinatesInstancesAndRejectsStaleWrite
// verifies IT-ARCH-009-005, ARCH-009, DESIGN-009 TagManager, and SW-REQ-057.
func TestClassificationGenerationLiveRedisCoordinatesInstancesAndRejectsStaleWrite(t *testing.T) {
	redisURL := os.Getenv("MEALSWAPP_REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379/13"
	}
	client, err := Open(redisURL)
	if err != nil {
		t.Fatalf("open Redis: %v", err)
	}
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis unavailable: %v", err)
	}
	if err := client.Del(ctx, classificationGenerationKey).Err(); err != nil {
		t.Fatalf("reset generation: %v", err)
	}
	t.Cleanup(func() { _ = client.Del(context.Background(), classificationGenerationKey).Err() })

	generation := NewClassificationGeneration(client)
	repo := &sharedFilterClassificationRepository{}
	first := search.NewVersionedFilterOptionService(repo, emptyFilterAllergenRepository{}, generation)
	second := search.NewVersionedFilterOptionService(repo, emptyFilterAllergenRepository{}, generation)
	if _, err := first.Options(ctx, search.SearchModeSubstitution); err != nil {
		t.Fatalf("warm first options: %v", err)
	}
	if _, err := second.Options(ctx, search.SearchModeSubstitution); err != nil {
		t.Fatalf("warm second options: %v", err)
	}
	id := uuid.New()
	repo.mu.Lock()
	repo.entries = []repository.ClassificationEntity{{ID: id, Name: "Created", Kind: repository.ClassificationKindFoodCategory}}
	repo.mu.Unlock()
	NewClassificationInvalidator(first, client).Invalidate()
	created, err := second.Options(ctx, search.SearchModeSubstitution)
	if err != nil || filterOptionLabel(created.Options, id) != "Created" {
		t.Fatalf("peer create options=%#v err=%v", created.Options, err)
	}
	repo.mu.Lock()
	repo.entries[0].Name = "Renamed"
	repo.mu.Unlock()
	NewClassificationInvalidator(first, client).Invalidate()
	renamed, err := second.Options(ctx, search.SearchModeSubstitution)
	if err != nil || filterOptionLabel(renamed.Options, id) != "Renamed" {
		t.Fatalf("peer rename options=%#v err=%v", renamed.Options, err)
	}

	request := search.SearchRequest{Query: "classification generation race", Mode: search.SearchModeCatalog, Page: 1}
	store := SearchResponseStore{Store: GoRedisStore{Client: client}, Generation: generation}
	_, hit, staleToken, err := store.GetSearchResponse(ctx, request)
	if err != nil || hit {
		t.Fatalf("initial search lookup hit=%v err=%v", hit, err)
	}
	NewClassificationInvalidator(first, client).Invalidate()
	stored, err := store.SetSearchResponse(ctx, request, search.SearchResponse{Items: []repository.FoodItemEntity{{Name: "Old label"}}, TotalCount: 1, Page: 1}, staleToken)
	if err != nil || stored {
		t.Fatalf("stale guarded write stored=%v err=%v", stored, err)
	}
	if _, hit, _, err := store.GetSearchResponse(ctx, request); err != nil || hit {
		t.Fatalf("post-invalidation stale lookup hit=%v err=%v", hit, err)
	}
	inputs := []search.SubstitutionInput{{FoodObjectID: uuid.New(), Quantity: 100, Unit: "g"}}
	_, hit, similarityToken, err := store.GetSimilarityCalculation(ctx, inputs)
	if err != nil || hit {
		t.Fatalf("initial similarity lookup hit=%v err=%v", hit, err)
	}
	NewClassificationInvalidator(first, client).Invalidate()
	stored, err = store.SetSimilarityCalculation(ctx, inputs, search.SimilarityCalculation{}, similarityToken)
	if err != nil || stored {
		t.Fatalf("stale similarity write stored=%v err=%v", stored, err)
	}
	if _, hit, _, err := store.GetSimilarityCalculation(ctx, inputs); err != nil || hit {
		t.Fatalf("post-invalidation similarity lookup hit=%v err=%v", hit, err)
	}
	repo.mu.Lock()
	repo.entries = nil
	repo.mu.Unlock()
	NewClassificationInvalidator(first, client).Invalidate()
	deleted, err := second.Options(ctx, search.SearchModeSubstitution)
	if err != nil || filterOptionLabel(deleted.Options, id) != "" {
		t.Fatalf("peer delete options=%#v err=%v", deleted.Options, err)
	}
}

func filterOptionLabel(options []search.FilterOption, id uuid.UUID) string {
	for _, option := range options {
		if option.FilterID == id.String() {
			return option.Label
		}
	}
	return ""
}
