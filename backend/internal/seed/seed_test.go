package seed

// Implements DESIGN-005 MicronutrientVocabulary.

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wiktor-jedski/mealswapp/backend/internal/migrations"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

type fakeBeginner struct {
	tx  *fakeSeedTx
	err error
}

func (b fakeBeginner) Begin(context.Context) (pgx.Tx, error) { return b.tx, b.err }

type fakeSeedTx struct {
	execErr   error
	commitErr error
}

func (t *fakeSeedTx) Begin(context.Context) (pgx.Tx, error) { return t, nil }
func (t *fakeSeedTx) Commit(context.Context) error          { return t.commitErr }
func (t *fakeSeedTx) Rollback(context.Context) error        { return nil }
func (t *fakeSeedTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, t.execErr
}
func (t *fakeSeedTx) Query(context.Context, string, ...any) (pgx.Rows, error) { return nil, nil }
func (t *fakeSeedTx) QueryRow(context.Context, string, ...any) pgx.Row        { return nil }
func (t *fakeSeedTx) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, nil
}
func (t *fakeSeedTx) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults { return nil }
func (t *fakeSeedTx) LargeObjects() pgx.LargeObjects                         { return pgx.LargeObjects{} }
func (t *fakeSeedTx) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	return nil, nil
}
func (t *fakeSeedTx) Conn() *pgx.Conn { return nil }

func TestRunErrors(t *testing.T) {
	ctx := context.Background()
	testErr := errors.New("failed")
	if err := Run(ctx, fakeBeginner{err: testErr}); !errors.Is(err, testErr) {
		t.Fatalf("Run() begin error = %v", err)
	}
	if err := Run(ctx, fakeBeginner{tx: &fakeSeedTx{execErr: testErr}}); !errors.Is(err, testErr) {
		t.Fatalf("Run() exec error = %v", err)
	}
	if err := Run(ctx, fakeBeginner{tx: &fakeSeedTx{commitErr: testErr}}); !errors.Is(err, testErr) {
		t.Fatalf("Run() commit error = %v", err)
	}
}

const seedTestDatabaseURL = "postgres://mealswapp:mealswapp@localhost:5432/mealswapp?sslmode=disable"

func openSeedTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	databaseURL := os.Getenv("MEALSWAPP_DATABASE_URL")
	if databaseURL == "" {
		databaseURL = seedTestDatabaseURL
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
	if _, err := pool.Exec(ctx, `SELECT pg_advisory_lock(9010101)`); err != nil {
		pool.Close()
		t.Fatalf("acquire seed test database lock: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `SELECT pg_advisory_unlock(9010101)`)
	})
	migrationDir, err := filepath.Abs("../../../database/migrations")
	if err != nil {
		pool.Close()
		t.Fatalf("resolve migration dir: %v", err)
	}
	if err := migrations.Run(ctx, pool, "down", migrationDir); err != nil {
		pool.Close()
		t.Fatalf("reset migrations down: %v", err)
	}
	if err := migrations.Run(ctx, pool, "up", migrationDir); err != nil {
		pool.Close()
		t.Fatalf("apply migrations up: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestRunIsIdempotentAndSeedsRepositoryFixtures(t *testing.T) {
	db := openSeedTestDB(t)
	ctx := context.Background()

	if err := Run(ctx, db); err != nil {
		t.Fatalf("Run() first error = %v", err)
	}
	firstCounts := seedCounts(t, ctx, db)
	if err := Run(ctx, db); err != nil {
		t.Fatalf("Run() second error = %v", err)
	}
	secondCounts := seedCounts(t, ctx, db)
	if firstCounts != secondCounts {
		t.Fatalf("seed counts changed after second run: first=%#v second=%#v", firstCounts, secondCounts)
	}

	foodID := uuid.MustParse("21000000-0000-0000-0000-000000000001")
	recipeID := uuid.MustParse("22000000-0000-0000-0000-000000000002")
	userID := uuid.MustParse("23000000-0000-0000-0000-000000000001")
	foodRepo := repository.NewPostgresFoodItemRepository(db)
	mealRepo := repository.NewPostgresMealRepository(db)
	tagRepo := repository.NewPostgresTagRepository(db)
	entitlementRepo := repository.NewPostgresEntitlementRepository(db)
	savedRepo := repository.NewPostgresSavedDataRepository(db)
	adminRepo := repository.NewPostgresAdminImportAuditRepository(db)

	food, err := foodRepo.GetByID(ctx, foodID, repository.RepositoryContext{})
	if err != nil {
		t.Fatalf("GetByID() seeded food error = %v", err)
	}
	if food.MacrosPer100 != (repository.MacroValues{Protein: 0.3, Carbohydrates: 14, Fat: 0.2}) || len(food.CategoryTags) != 1 {
		t.Fatalf("seeded food = %#v", food)
	}

	macros, err := mealRepo.CalculateMacros(ctx, recipeID)
	if err != nil {
		t.Fatalf("CalculateMacros() seeded composite error = %v", err)
	}
	if macros != (repository.MacroValues{Protein: 4.7091, Carbohydrates: 9.4545, Fat: 1.0182}) {
		t.Fatalf("seeded composite macros = %#v", macros)
	}

	categoryTags, err := tagRepo.List(ctx, repository.TagKindCategory)
	if err != nil {
		t.Fatalf("List() category tags error = %v", err)
	}
	functionalityTags, err := tagRepo.List(ctx, repository.TagKindFunctionality)
	if err != nil {
		t.Fatalf("List() functionality tags error = %v", err)
	}
	if len(categoryTags) < 2 || len(functionalityTags) < 2 {
		t.Fatalf("seeded tags category=%#v functionality=%#v", categoryTags, functionalityTags)
	}

	entitlement, err := entitlementRepo.GetLatest(ctx, userID)
	if err != nil {
		t.Fatalf("GetLatest() seeded entitlement error = %v", err)
	}
	if entitlement.Tier != "free" || entitlement.SearchLimitPer24h != 3 {
		t.Fatalf("seeded entitlement = %#v", entitlement)
	}

	items, err := savedRepo.ListItems(ctx, userID, nil)
	if err != nil {
		t.Fatalf("ListItems() seeded saved data error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("seeded saved items length = %d, want 2: %#v", len(items), items)
	}

	imported, err := adminRepo.FindCuratedImport(ctx, "seed-provider", "seed-external-1")
	if err != nil {
		t.Fatalf("FindCuratedImport() seeded import error = %v", err)
	}
	if imported.FoodItemID == nil || *imported.FoodItemID != foodID {
		t.Fatalf("seeded import = %#v", imported)
	}
}

type seedCountSnapshot struct {
	Foods        int
	Meals        int
	Users        int
	Entitlements int
	SavedItems   int
	AuditEntries int
}

func seedCounts(t *testing.T, ctx context.Context, db *pgxpool.Pool) seedCountSnapshot {
	t.Helper()
	var counts seedCountSnapshot
	queries := []struct {
		sql  string
		dest *int
	}{
		{`SELECT count(*) FROM food_items WHERE id::text LIKE '21000000-%'`, &counts.Foods},
		{`SELECT count(*) FROM meals WHERE id::text LIKE '22000000-%'`, &counts.Meals},
		{`SELECT count(*) FROM users WHERE id::text LIKE '23000000-%'`, &counts.Users},
		{`SELECT count(*) FROM entitlements WHERE user_id = '23000000-0000-0000-0000-000000000001'`, &counts.Entitlements},
		{`SELECT count(*) FROM saved_items WHERE user_id = '23000000-0000-0000-0000-000000000001'`, &counts.SavedItems},
		{`SELECT count(*) FROM admin_audit_entries WHERE request_id = 'seed-request'`, &counts.AuditEntries},
	}
	for _, query := range queries {
		if err := db.QueryRow(ctx, query.sql).Scan(query.dest); err != nil {
			t.Fatalf("count query %q: %v", query.sql, err)
		}
	}
	return counts
}
