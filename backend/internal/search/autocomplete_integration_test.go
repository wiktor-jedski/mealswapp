package search

// Implements DESIGN-002 AutocompleteRanker repository integration verification.

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wiktor-jedski/mealswapp/backend/internal/migrations"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

const autocompleteTestDatabaseURL = "postgres://mealswapp:mealswapp@localhost:5432/mealswapp?sslmode=disable"

func openAutocompleteTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	databaseURL := os.Getenv("MEALSWAPP_DATABASE_URL")
	if databaseURL == "" {
		databaseURL = autocompleteTestDatabaseURL
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("postgres unavailable: %v", err)
	}
	if _, err := pool.Exec(ctx, "SELECT pg_advisory_lock(9010101)"); err != nil {
		pool.Close()
		t.Fatalf("acquire autocomplete test database lock: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "SELECT pg_advisory_unlock(9010101)")
	})

	migrationDir, err := filepath.Abs("../../../database/migrations")
	if err != nil {
		pool.Close()
		t.Fatalf("resolve migration dir: %v", err)
	}
	if _, err := pool.Exec(ctx, "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"); err != nil {
		pool.Close()
		t.Fatalf("reset database schema: %v", err)
	}
	if err := migrations.Run(ctx, pool, "up", migrationDir); err != nil {
		pool.Close()
		t.Fatalf("apply migrations up: %v", err)
	}

	t.Cleanup(pool.Close)
	return pool
}

func TestAutocompleteServiceUsesRealRepositoriesForRankingAndSafety(t *testing.T) {
	// Verifies IT-ARCH-002-004.
	// Verifies ARCH-002.
	// Verifies ARCH-005.
	// Traces SW-REQ-004, SW-REQ-010, SW-REQ-019.
	db := openAutocompleteTestDB(t)
	ctx := context.Background()
	foodRepo := repository.NewPostgresFoodItemRepository(db)
	mealRepo := repository.NewPostgresMealRepository(db)
	service := NewAutocompleteService(foodRepo, mealRepo)

	activeFoodID := createAutocompleteFood(t, ctx, foodRepo, "App")
	createAutocompleteFood(t, ctx, foodRepo, "Apple")
	createAutocompleteFood(t, ctx, foodRepo, "Application")
	createAutocompleteFood(t, ctx, foodRepo, "Apx")
	createAutocompleteMeal(t, ctx, mealRepo, "Apzzzzzz")
	createAutocompleteMeal(t, ctx, mealRepo, "App")
	createAutocompleteMeal(t, ctx, mealRepo, "App Snack")
	deletedFoodID := createAutocompleteFood(t, ctx, foodRepo, "App Deleted")
	deletedMealID := createAutocompleteMeal(t, ctx, mealRepo, "App Removed")
	if err := foodRepo.Delete(ctx, deletedFoodID); err != nil {
		t.Fatalf("delete food fixture: %v", err)
	}
	if err := mealRepo.Delete(ctx, deletedMealID); err != nil {
		t.Fatalf("delete meal fixture: %v", err)
	}
	for i := 0; i < PageSize+3; i++ {
		createAutocompleteFood(t, ctx, foodRepo, "App Overflow "+string(rune('A'+i)))
	}

	first, err := service.Autocomplete(ctx, " app ", repository.RepositoryContext{IncludeDeleted: true})
	if err != nil {
		t.Fatalf("Autocomplete() error = %v", err)
	}
	second, err := service.Autocomplete(ctx, " app ", repository.RepositoryContext{IncludeDeleted: true})
	if err != nil {
		t.Fatalf("second Autocomplete() error = %v", err)
	}

	if len(first) > PageSize {
		t.Fatalf("Autocomplete() returned %d results, want at most %d", len(first), PageSize)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("repeated autocomplete calls differ:\nfirst=%#v\nsecond=%#v", first, second)
	}
	gotLabels := labels(first)
	if len(gotLabels) < 4 || !reflect.DeepEqual(gotLabels[:4], []string{"App", "App", "Apx", "Apple"}) {
		t.Fatalf("ranked labels prefix = %#v, want exact food and meal first, then distance-ranked candidates", gotLabels)
	}
	for _, item := range first {
		if item.Label == "App Deleted" || item.Label == "App Removed" {
			t.Fatalf("Autocomplete() included deleted row: %#v", first)
		}
	}
	if first[0].ItemID != activeFoodID.String() && first[1].ItemID != activeFoodID.String() {
		t.Fatalf("active food exact match missing from first exact results: %#v", first[:2])
	}

	fuzzy, err := service.Autocomplete(ctx, "apzzzz", repository.RepositoryContext{})
	if err != nil {
		t.Fatalf("Autocomplete() fuzzy query error = %v", err)
	}
	fuzzyLabels := labels(fuzzy)
	if len(fuzzyLabels) < 2 || fuzzyLabels[0] != "Apzzzzzz" {
		t.Fatalf("fuzzy labels = %#v, want lower Levenshtein distance before shorter label", fuzzyLabels)
	}
	if fuzzy[0].Length <= fuzzy[1].Length || fuzzy[0].LevenshteinDistance >= fuzzy[1].LevenshteinDistance {
		t.Fatalf("fuzzy ranking metadata = %#v, want Levenshtein distance before length", fuzzy[:2])
	}

	injected, err := service.Autocomplete(ctx, "x' OR 1=1 --", repository.RepositoryContext{})
	if err != nil {
		t.Fatalf("Autocomplete() special-character query error = %v", err)
	}
	if len(injected) != 0 {
		t.Fatalf("Autocomplete() special-character query returned %#v, want no injected rows", injected)
	}
}

func createAutocompleteFood(t *testing.T, ctx context.Context, repo repository.FoodItemRepository, name string) uuid.UUID {
	t.Helper()
	id, err := repo.Create(ctx, repository.FoodItemEntity{
		Name:                   name,
		PhysicalState:          repository.PhysicalStateSolid,
		AverageUnitWeightGrams: 100,
		MacrosPer100:           repository.MacroValues{Protein: 1, Carbohydrates: 2, Fat: 3},
	})
	if err != nil {
		t.Fatalf("create food %q: %v", name, err)
	}
	return id
}

func createAutocompleteMeal(t *testing.T, ctx context.Context, repo repository.MealRepository, name string) uuid.UUID {
	t.Helper()
	id, err := repo.Create(ctx, repository.MealEntity{
		Type:                   repository.MealTypeSingle,
		Name:                   name,
		PhysicalState:          repository.PhysicalStateSolid,
		AverageUnitWeightGrams: 100,
		MacrosPer100:           repository.MacroValues{Protein: 2, Carbohydrates: 3, Fat: 4},
	})
	if err != nil {
		t.Fatalf("create meal %q: %v", name, err)
	}
	return id
}
