package postgres

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"mealswapp/backend/internal/domain/meal"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMealRepositoryCRUDWithMultipleIngredients(t *testing.T) {
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

	applyMigration(t, ctx, pool, "0002_meals.down.sql")
	applyMigration(t, ctx, pool, "0001_food_items.down.sql")
	applyMigration(t, ctx, pool, "0001_food_items.up.sql")
	applyMigration(t, ctx, pool, "0002_meals.up.sql")
	defer applyMigration(t, ctx, pool, "0001_food_items.down.sql")
	defer applyMigration(t, ctx, pool, "0002_meals.down.sql")

	firstFoodID := insertTestFood(t, ctx, pool, "Oats", 389, 16.9, 66.3, 6.9)
	secondFoodID := insertTestFood(t, ctx, pool, "Milk", 42, 3.4, 5.0, 1.0)

	repo := NewMealRepository(pool)
	userID := uuid.New()
	createdID, err := repo.Create(ctx, meal.MealEntity{
		UserID: userID,
		Name:   "Breakfast",
		Type:   meal.MealTypeRecipe,
		Items: []meal.MealItemEntity{
			{FoodItemID: firstFoodID, Quantity: 80, Unit: meal.IngredientUnitGram, Position: 0},
			{FoodItemID: secondFoodID, Quantity: 200, Unit: meal.IngredientUnitMilliliter, Position: 1},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	created, err := repo.GetByID(ctx, createdID)
	if err != nil {
		t.Fatal(err)
	}
	if created.Name != "Breakfast" || len(created.Items) != 2 {
		t.Fatalf("unexpected created meal: %#v", created)
	}

	created.Name = "Updated breakfast"
	created.Items = []meal.MealItemEntity{
		{FoodItemID: secondFoodID, Quantity: 250, Unit: meal.IngredientUnitMilliliter, Position: 0},
		{FoodItemID: firstFoodID, Quantity: 60, Unit: meal.IngredientUnitGram, Position: 1},
	}
	if err := repo.Update(ctx, created); err != nil {
		t.Fatal(err)
	}

	updated, err := repo.GetByID(ctx, createdID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Updated breakfast" || len(updated.Items) != 2 {
		t.Fatalf("unexpected updated meal: %#v", updated)
	}
	if updated.Items[0].FoodItemID != secondFoodID || updated.Items[0].Quantity != 250 {
		t.Fatalf("meal items were not replaced in position order: %#v", updated.Items)
	}

	if err := repo.Delete(ctx, createdID); err != nil {
		t.Fatal(err)
	}

	if _, err := repo.GetByID(ctx, createdID); err != pgx.ErrNoRows {
		t.Fatalf("expected deleted meal to be missing, got %v", err)
	}
}

func insertTestFood(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	name string,
	calories float64,
	protein float64,
	carbs float64,
	fat float64,
) uuid.UUID {
	t.Helper()

	var id uuid.UUID
	err := pool.QueryRow(ctx, `
		INSERT INTO food_items (
			name,
			physical_state,
			serving_unit,
			serving_size,
			calories_per_100,
			protein_grams_per_100,
			carbs_grams_per_100,
			fat_grams_per_100
		)
		VALUES ($1, 'solid', 'gram', 100, $2, $3, $4, $5)
		RETURNING id
	`, name, calories, protein, carbs, fat).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}

	return id
}

func applyMigration(t *testing.T, ctx context.Context, pool *pgxpool.Pool, name string) {
	t.Helper()

	sql, err := os.ReadFile(filepath.Join(repoRoot(t), "db", "migrations", name))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := pool.Exec(ctx, string(sql)); err != nil {
		t.Fatalf("apply migration %s: %v", name, err)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not resolve current file path")
	}

	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", "..", ".."))
}
