package seed

// Implements DESIGN-005 MicronutrientVocabulary.

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/search"
	"github.com/wiktor-jedski/mealswapp/backend/internal/testdatabase"
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

func openSeedTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	migrationDir, err := filepath.Abs("../../../database/migrations")
	if err != nil {
		t.Fatalf("resolve migration dir: %v", err)
	}
	return testdatabase.Reset(t, migrationDir)
}

func TestRunIsIdempotentAndSeedsRepositoryFixtures(t *testing.T) {
	db := openSeedTestDB(t)
	ctx := context.Background()

	// Regression: earlier local dev seeds created milk fixtures with random UUIDs,
	// which conflicts with the deterministic active-name fixture insert.
	insertLegacyActiveFoodFixture(t, ctx, db, "Oat Milk")
	insertLegacyActiveFoodFixture(t, ctx, db, "Cow Milk")

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
	if firstCounts.Meals != 27 {
		t.Fatalf("seeded meal count = %d, want 27", firstCounts.Meals)
	}
	wantReplacementMeals := []string{
		"Almonds", "Avocado", "Banana", "Boiled Potatoes", "Cheddar Cheese",
		"Chicken Breast", "Chickpeas", "Cooked Brown Rice", "Cooked White Rice",
		"Egg Whites", "Firm Tofu", "Ghee", "Granulated Sugar", "Lean Beef",
		"Lentils", "Peanut Butter", "Protein Isolate", "Rolled Oats", "Salmon Fillet",
		"Seitan", "Sweet Potato", "Tuna in Water", "Turkey Breast", "Whole Eggs",
		"Whole Wheat Bread",
	}
	rows, err := db.Query(ctx, `
		SELECT name
		FROM meals
		WHERE id::text LIKE '22000000-0000-0000-0000-0000000001%'
		ORDER BY name
	`)
	if err != nil {
		t.Fatalf("query seeded replacement meals: %v", err)
	}
	defer rows.Close()
	gotReplacementMeals := make([]string, 0, len(wantReplacementMeals))
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan seeded replacement meal: %v", err)
		}
		gotReplacementMeals = append(gotReplacementMeals, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate seeded replacement meals: %v", err)
	}
	if !slices.Equal(gotReplacementMeals, wantReplacementMeals) {
		t.Fatalf("seeded replacement meals = %#v, want %#v", gotReplacementMeals, wantReplacementMeals)
	}

	foodID := uuid.MustParse("21000000-0000-0000-0000-000000000001")
	oatMilkID := uuid.MustParse("21000000-0000-0000-0000-000000000003")
	cowMilkID := uuid.MustParse("21000000-0000-0000-0000-000000000004")
	recipeID := uuid.MustParse("22000000-0000-0000-0000-000000000002")
	userID := uuid.MustParse("23000000-0000-0000-0000-000000000001")
	foodRepo := repository.NewPostgresFoodItemRepository(db)
	mealRepo := repository.NewPostgresMealRepository(db)
	salmonID := uuid.MustParse("22000000-0000-0000-0000-000000000104")
	substitutions := search.NewSubstitutionService(foodRepo, nil).WithMealRepository(mealRepo)
	salmonResponse, err := substitutions.Search(ctx, search.SearchRequest{
		Query: "", Mode: search.SearchModeSubstitution, Page: 1,
		SubstitutionInputs: []search.SubstitutionInput{{
			FoodObjectID: salmonID, FoodObjectType: repository.FoodObjectTypeMeal, Quantity: 100, Unit: "g",
		}},
	})
	if err != nil {
		t.Fatalf("seeded Salmon substitution error = %v", err)
	}
	if salmonResponse.Rejection != nil || salmonResponse.SourceSummary == nil || salmonResponse.SourceSummary.Macros != (repository.MacroValues{Protein: 20, Fat: 13}) || salmonResponse.SourceSummary.Calories != 197 {
		t.Fatalf("seeded Salmon substitution response = %+v", salmonResponse)
	}
	if !slices.Contains(salmonResponse.ItemTypes, repository.FoodObjectTypeMeal) {
		t.Fatalf("seeded Salmon substitutions contain no Meal candidates: types=%+v items=%+v", salmonResponse.ItemTypes, salmonResponse.Items)
	}
	classificationRepo := repository.NewPostgresClassificationRepository(db)
	entitlementRepo := repository.NewPostgresEntitlementRepository(db)
	savedRepo := repository.NewPostgresSavedDataRepository(db)
	adminRepo := repository.NewPostgresAdminImportAuditRepository(db)

	food, err := foodRepo.GetByID(ctx, foodID, repository.RepositoryContext{})
	if err != nil {
		t.Fatalf("GetByID() seeded food error = %v", err)
	}
	if food.MacrosPer100 != (repository.MacroValues{Protein: 0.3, Carbohydrates: 14, Fat: 0.2}) || len(food.FoodCategories) != 1 {
		t.Fatalf("seeded food = %#v", food)
	}
	milkItems, total, err := foodRepo.Search(ctx, repository.RepositoryQuery{Name: "milk", Limit: 10})
	if err != nil {
		t.Fatalf("Search() seeded milk error = %v", err)
	}
	if total < 2 || !containsFoodID(milkItems, oatMilkID) || !containsFoodID(milkItems, cowMilkID) {
		t.Fatalf("seeded milk search total=%d items=%#v", total, milkItems)
	}
	var legacyActiveCount int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM food_items WHERE id::text NOT LIKE '21000000-%' AND normalized_name IN ('oat milk', 'cow milk') AND deleted_at IS NULL`).Scan(&legacyActiveCount); err != nil {
		t.Fatalf("legacy active count query: %v", err)
	}
	if legacyActiveCount != 0 {
		t.Fatalf("legacy active fixture count = %d, want 0", legacyActiveCount)
	}
	dairyFreeItems, _, err := foodRepo.Search(ctx, repository.RepositoryQuery{Name: "milk", ExcludedAllergenKeys: []string{"dairy"}, Limit: 10})
	if err != nil {
		t.Fatalf("Search() seeded dairy-free milk error = %v", err)
	}
	if !containsFoodID(dairyFreeItems, oatMilkID) || containsFoodID(dairyFreeItems, cowMilkID) {
		t.Fatalf("seeded dairy-free milk items=%#v", dairyFreeItems)
	}

	macros, err := mealRepo.CalculateMacros(ctx, recipeID)
	if err != nil {
		t.Fatalf("CalculateMacros() seeded composite error = %v", err)
	}
	if macros != (repository.MacroValues{Protein: 4.7091, Carbohydrates: 9.4545, Fat: 1.0182}) {
		t.Fatalf("seeded composite macros = %#v", macros)
	}

	foodCategories, err := classificationRepo.List(ctx, repository.ClassificationKindFoodCategory)
	if err != nil {
		t.Fatalf("List() food_category classifications error = %v", err)
	}
	culinaryRoles, err := classificationRepo.List(ctx, repository.ClassificationKindCulinaryRole)
	if err != nil {
		t.Fatalf("List() culinary_role classifications error = %v", err)
	}
	if len(foodCategories) < 2 || len(culinaryRoles) < 2 {
		t.Fatalf("seeded classifications food_category=%#v culinary_role=%#v", foodCategories, culinaryRoles)
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

func insertLegacyActiveFoodFixture(t *testing.T, ctx context.Context, db *pgxpool.Pool, name string) {
	t.Helper()
	_, err := db.Exec(ctx, `
		INSERT INTO food_items (
			name, physical_state, average_serving_volume_milliliters,
			density_grams_per_milliliter, density_source_kind,
			protein_per_100, carbohydrates_per_100, fat_per_100
		)
		VALUES ($1, 'liquid', 240, 1.03, 'estimated', 1, 1, 1)
	`, name)
	if err != nil {
		t.Fatalf("insert legacy active food fixture %q: %v", name, err)
	}
}

func containsFoodID(items []repository.FoodItemEntity, id uuid.UUID) bool {
	for _, item := range items {
		if item.ID == id {
			return true
		}
	}
	return false
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
