package cache

// Implements DESIGN-009 ItemCurator shared food-data generation integration verification.

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/search"
)

type manualItemSearchRepository struct {
	mu    sync.RWMutex
	items []repository.FoodItemEntity
}

func (r *manualItemSearchRepository) GetByID(_ context.Context, id uuid.UUID, _ repository.RepositoryContext) (repository.FoodItemEntity, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, item := range r.items {
		if item.ID == id {
			return item, nil
		}
	}
	return repository.FoodItemEntity{}, repository.NewError(repository.ErrorKindNotFound, "food item not found", nil)
}

func (r *manualItemSearchRepository) Search(_ context.Context, query repository.RepositoryQuery) ([]repository.FoodItemEntity, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]repository.FoodItemEntity, 0, len(r.items))
	for _, item := range r.items {
		if query.Name == "" || strings.Contains(strings.ToLower(item.Name), strings.ToLower(query.Name)) {
			items = append(items, item)
		}
	}
	return items, len(items), nil
}

func (r *manualItemSearchRepository) replace(items ...repository.FoodItemEntity) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items = append([]repository.FoodItemEntity(nil), items...)
}

// TestManualItemGenerationLiveRedisRefreshesPeerCatalogAndSubstitutionSearch
// verifies IT-ARCH-009-008, ARCH-009, DESIGN-009 ItemCurator,
// DESIGN-011 RedisCache, and SW-REQ-056.
func TestManualItemGenerationLiveRedisRefreshesPeerCatalogAndSubstitutionSearch(t *testing.T) {
	redisURL := os.Getenv("MEALSWAPP_REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379/13"
	}
	client, err := Open(redisURL)
	if err != nil {
		t.Fatalf("open Redis: %v", err)
	}
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis unavailable: %v", err)
	}

	query := "task264-" + uuid.NewString()
	source := repository.FoodItemEntity{
		ID: uuid.New(), Name: "Source protein", PhysicalState: repository.PhysicalStateSolid,
		MacrosPer100: repository.MacroValues{Protein: 10, Carbohydrates: 2, Fat: 1},
	}
	manual := repository.FoodItemEntity{
		ID: uuid.New(), Name: query + " tofu", PhysicalState: repository.PhysicalStateSolid,
		MacrosPer100: repository.MacroValues{Protein: 10, Carbohydrates: 2, Fat: 1},
	}
	repo := &manualItemSearchRepository{}
	repo.replace(source)
	generation := NewClassificationGeneration(client)
	store := SearchResponseStore{Store: GoRedisStore{Client: client}, Generation: generation}
	firstCatalog := search.NewCatalogService(repo, store)
	secondCatalog := search.NewCatalogService(repo, store)
	firstSubstitution := search.NewSubstitutionService(repo, store, store)
	secondSubstitution := search.NewSubstitutionService(repo, store, store)
	catalogRequest := search.SearchRequest{Query: query, Mode: search.SearchModeCatalog, Page: 1}
	substitutionRequest := search.SearchRequest{
		Mode: search.SearchModeSubstitution, Page: 1,
		SubstitutionInputs: []search.SubstitutionInput{{FoodObjectID: source.ID, Quantity: 100, Unit: "g"}},
	}
	cleanup := func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cleanupCancel()
		_ = client.Del(cleanupCtx, classificationGenerationKey).Err()
		for _, pattern := range []string{
			string(RedisNamespaceSearch) + ":*:" + BuildSearchCacheKey(catalogRequest).ID,
			string(RedisNamespaceSearch) + ":*:" + BuildSearchCacheKey(substitutionRequest).ID,
			string(RedisNamespaceSimilarity) + ":*:" + BuildSimilarityCacheKey(substitutionRequest.SubstitutionInputs).ID,
		} {
			keys, _, scanErr := client.Scan(cleanupCtx, 0, pattern, 100).Result()
			if scanErr == nil && len(keys) > 0 {
				_ = client.Del(cleanupCtx, keys...).Err()
			}
		}
	}
	cleanup()
	t.Cleanup(cleanup)

	assertSearchItemName(t, firstCatalog, ctx, catalogRequest, "")
	assertSearchItemName(t, firstSubstitution, ctx, substitutionRequest, "")
	_, _, staleSearchToken, err := store.GetSearchResponse(ctx, catalogRequest)
	if err != nil {
		t.Fatalf("capture stale search token: %v", err)
	}
	_, _, staleSimilarityToken, err := store.GetSimilarityCalculation(ctx, substitutionRequest.SubstitutionInputs)
	if err != nil {
		t.Fatalf("capture stale similarity token: %v", err)
	}

	repo.replace(source, manual)
	NewClassificationInvalidator(nil, client).Invalidate()
	assertSearchItemName(t, secondCatalog, ctx, catalogRequest, query+" tofu")
	assertSearchItemName(t, secondSubstitution, ctx, substitutionRequest, query+" tofu")
	if stored, err := store.SetSearchResponse(ctx, catalogRequest, search.SearchResponse{Items: []repository.FoodItemEntity{{Name: "stale catalog"}}}, staleSearchToken); err != nil || stored {
		t.Fatalf("stale catalog write stored=%t err=%v", stored, err)
	}
	if stored, err := store.SetSimilarityCalculation(ctx, substitutionRequest.SubstitutionInputs, search.SimilarityCalculation{}, staleSimilarityToken); err != nil || stored {
		t.Fatalf("stale similarity write stored=%t err=%v", stored, err)
	}

	manual.Name = query + " tempeh"
	manual.MacrosPer100.Protein = 20
	repo.replace(source, manual)
	NewClassificationInvalidator(nil, client).Invalidate()
	assertSearchItemName(t, firstCatalog, ctx, catalogRequest, query+" tempeh")
	assertSearchItemName(t, firstSubstitution, ctx, substitutionRequest, query+" tempeh")

	repo.replace(source)
	NewClassificationInvalidator(nil, client).Invalidate()
	assertSearchItemName(t, secondCatalog, ctx, catalogRequest, "")
	assertSearchItemName(t, secondSubstitution, ctx, substitutionRequest, "")
}

type searchRunner interface {
	Search(context.Context, search.SearchRequest) (search.SearchResponse, error)
}

func assertSearchItemName(t *testing.T, service searchRunner, ctx context.Context, request search.SearchRequest, want string) {
	t.Helper()
	response, err := service.Search(ctx, request)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if want == "" {
		if len(response.Items) != 0 {
			t.Fatalf("search items=%+v, want empty", response.Items)
		}
		return
	}
	if len(response.Items) != 1 || response.Items[0].Name != want {
		t.Fatalf("search items=%+v, want %q", response.Items, want)
	}
}
