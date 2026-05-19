package postgres

import (
	"context"
	"math"
	"os"
	"testing"

	"mealswapp/backend/internal/domain/meal"
	"mealswapp/backend/internal/domain/recipe"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRecipeRepositoryCalculatesAndPersistsTotals(t *testing.T) {
	databaseURL := os.Getenv("MEALSWAPP_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("MEALSWAPP_TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	applyMigration(t, ctx, pool, "0003_recipes.down.sql")
	applyMigration(t, ctx, pool, "0002_meals.down.sql")
	applyMigration(t, ctx, pool, "0001_food_items.down.sql")
	applyMigration(t, ctx, pool, "0001_food_items.up.sql")
	applyMigration(t, ctx, pool, "0003_recipes.up.sql")
	defer applyMigration(t, ctx, pool, "0001_food_items.down.sql")
	defer applyMigration(t, ctx, pool, "0003_recipes.down.sql")

	oatsID := insertTestFood(t, ctx, pool, "Oats", 389, 16.9, 66.3, 6.9)
	milkID := insertTestFood(t, ctx, pool, "Milk", 42, 3.4, 5.0, 1.0)

	repo := NewRecipeRepository(pool)
	createdID, err := repo.Create(ctx, recipe.RecipeEntity{
		UserID: uuid.New(),
		Name:   "Porridge",
		Ingredients: []recipe.RecipeIngredientEntity{
			{FoodItemID: oatsID, Quantity: 80, Unit: meal.IngredientUnitGram, Position: 0},
			{FoodItemID: milkID, Quantity: 200, Unit: meal.IngredientUnitMilliliter, Position: 1},
		},
		SourceProvider: "test",
		SourceID:       "porridge",
	})
	if err != nil {
		t.Fatal(err)
	}

	created, err := repo.GetByID(ctx, createdID)
	if err != nil {
		t.Fatal(err)
	}

	assertClose(t, created.CaloriesTotal, 395.2)
	assertClose(t, created.MacrosTotal.ProteinGrams, 20.32)
	assertClose(t, created.MacrosTotal.CarbsGrams, 63.04)
	assertClose(t, created.MacrosTotal.FatGrams, 7.52)

	created.Name = "Smaller porridge"
	created.Ingredients = []recipe.RecipeIngredientEntity{
		{FoodItemID: oatsID, Quantity: 60, Unit: meal.IngredientUnitGram, Position: 0},
		{FoodItemID: milkID, Quantity: 100, Unit: meal.IngredientUnitMilliliter, Position: 1},
	}
	if err := repo.Update(ctx, created); err != nil {
		t.Fatal(err)
	}

	updated, err := repo.GetByID(ctx, createdID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Smaller porridge" || len(updated.Ingredients) != 2 {
		t.Fatalf("unexpected updated recipe: %#v", updated)
	}
	assertClose(t, updated.CaloriesTotal, 275.4)
	assertClose(t, updated.MacrosTotal.ProteinGrams, 13.54)
	assertClose(t, updated.MacrosTotal.CarbsGrams, 44.78)
	assertClose(t, updated.MacrosTotal.FatGrams, 5.14)

	if err := repo.Delete(ctx, createdID); err != nil {
		t.Fatal(err)
	}

	if _, err := repo.GetByID(ctx, createdID); err != pgx.ErrNoRows {
		t.Fatalf("expected deleted recipe to be missing, got %v", err)
	}
}

func assertClose(t *testing.T, got float64, want float64) {
	t.Helper()

	if math.Abs(got-want) > 0.001 {
		t.Fatalf("expected %.3f, got %.3f", want, got)
	}
}
