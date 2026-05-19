package postgres

import (
	"context"
	"os"
	"testing"

	"mealswapp/backend/internal/domain/tag"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestTagRepositoryAttachRemoveAndFilterFoodItems(t *testing.T) {
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

	applyMigration(t, ctx, pool, "0004_tags.down.sql")
	applyMigration(t, ctx, pool, "0003_recipes.down.sql")
	applyMigration(t, ctx, pool, "0002_meals.down.sql")
	applyMigration(t, ctx, pool, "0001_food_items.down.sql")
	applyMigration(t, ctx, pool, "0001_food_items.up.sql")
	applyMigration(t, ctx, pool, "0004_tags.up.sql")
	defer applyMigration(t, ctx, pool, "0001_food_items.down.sql")
	defer applyMigration(t, ctx, pool, "0004_tags.down.sql")

	oatsID := insertTestFood(t, ctx, pool, "Oats", 389, 16.9, 66.3, 6.9)
	milkID := insertTestFood(t, ctx, pool, "Milk", 42, 3.4, 5.0, 1.0)

	repo := NewTagRepository(pool)
	veganID, err := repo.Upsert(ctx, tag.TagEntity{Name: "Vegan", Kind: tag.KindDiet})
	if err != nil {
		t.Fatal(err)
	}
	dairyID, err := repo.Upsert(ctx, tag.TagEntity{Name: "Dairy", Kind: tag.KindAllergen})
	if err != nil {
		t.Fatal(err)
	}

	if err := repo.AttachToFoodItem(ctx, oatsID, veganID); err != nil {
		t.Fatal(err)
	}
	if err := repo.AttachToFoodItem(ctx, milkID, dairyID); err != nil {
		t.Fatal(err)
	}

	ids, err := repo.QueryFoodItemIDs(ctx, FoodItemTagFilter{IncludeTagIDs: []uuid.UUID{veganID}, ExcludeTagIDs: []uuid.UUID{dairyID}})
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != oatsID {
		t.Fatalf("expected only oats for vegan include and dairy exclude, got %#v", ids)
	}

	if err := repo.RemoveFromFoodItem(ctx, oatsID, veganID); err != nil {
		t.Fatal(err)
	}

	ids, err = repo.QueryFoodItemIDs(ctx, FoodItemTagFilter{IncludeTagIDs: []uuid.UUID{veganID}})
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 0 {
		t.Fatalf("expected no vegan-tagged foods after removal, got %#v", ids)
	}

	if err := repo.RemoveFromFoodItem(ctx, oatsID, veganID); err != pgx.ErrNoRows {
		t.Fatalf("expected removing an absent link to return no rows, got %v", err)
	}
}
