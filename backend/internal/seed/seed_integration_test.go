package seed

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestApplyIsIdempotentAndLoadsKnownFixtureIDs(t *testing.T) {
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

	resetSeedSchema(t, ctx, pool)
	defer resetSeedSchema(t, ctx, pool)

	if err := Apply(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if err := Apply(ctx, pool); err != nil {
		t.Fatal(err)
	}

	assertCount(t, ctx, pool, `SELECT count(*) FROM users WHERE id = $1 AND role = 'admin'`, AdminUserID, 1)
	assertCount(t, ctx, pool, `SELECT count(*) FROM food_items WHERE id IN ($1, $2, $3)`, OatsFoodID, MilkFoodID, TofuFoodID, 3)
	assertCount(t, ctx, pool, `SELECT count(*) FROM tags WHERE id IN ($1, $2, $3)`, VeganTagID, DairyTagID, HighProteinTagID, 3)
	assertCount(t, ctx, pool, `SELECT count(*) FROM recipes WHERE id = $1`, PorridgeRecipeID, 1)
	assertCount(t, ctx, pool, `SELECT count(*) FROM recipe_ingredients WHERE recipe_id = $1`, PorridgeRecipeID, 2)
	assertCount(t, ctx, pool, `SELECT count(*) FROM micronutrient_vocabulary WHERE key = 'Sodium' AND active = true`, 1)
}

func resetSeedSchema(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	names := map[int]string{
		1: "food_items",
		2: "meals",
		3: "recipes",
		4: "tags",
		5: "micronutrient_vocabulary",
		6: "repository_foundation",
	}
	for i := 6; i >= 1; i-- {
		applySeedMigration(t, ctx, pool, fmt.Sprintf("%04d_%s.down.sql", i, names[i]))
	}
	for i := 1; i <= 6; i++ {
		applySeedMigration(t, ctx, pool, fmt.Sprintf("%04d_%s.up.sql", i, names[i]))
	}
}

func applySeedMigration(t *testing.T, ctx context.Context, pool *pgxpool.Pool, name string) {
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

	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", ".."))
}

func assertCount(t *testing.T, ctx context.Context, pool *pgxpool.Pool, query string, args ...any) {
	t.Helper()

	want := args[len(args)-1].(int)
	args = args[:len(args)-1]

	var got int
	if err := pool.QueryRow(ctx, query, args...).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("expected count %d for query %q, got %d", want, query, got)
	}
}
