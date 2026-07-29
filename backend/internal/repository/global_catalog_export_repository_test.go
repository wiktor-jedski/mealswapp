package repository

// Implements DESIGN-009 AdminController read-only consistent catalog snapshot verification.

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// Implements DESIGN-009 AdminController global catalog export integration fixture.
//
//go:embed sql/testdata/global_catalog_export_fixture_create.sql
var globalCatalogExportFixtureCreateSQL string

func TestGlobalCatalogExportRepositoryReadOnlySnapshotAndProjection(t *testing.T) {
	id := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	created := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	weight := "12.3400"
	tx := &fakeTx{}
	tx.rows = &fakeRows{next: true, values: []any{
		id, "Żurek", PhysicalStateSolid, 8, &weight, (*string)(nil), (*string)(nil),
		(*string)(nil), (*string)(nil), (*string)(nil), "1.0000", "2.0000", "3.0000",
		[]byte(`{"VitaminC":4.50,"Iron":1.25}`), (*string)(nil), (*string)(nil),
		(*string)(nil), (*string)(nil), (*time.Time)(nil), created, created,
		[]byte(`[{"id":"00000000-0000-4000-8000-000000000002","name":"Category"}]`),
		[]byte(`[]`), []byte(`["dairy"]`), []byte(`[]`),
	}}
	db := &fakeSQLExecutor{tx: tx}
	repository := NewPostgresGlobalCatalogExportRepository(db)
	var items []CatalogExportItem
	count, err := repository.Stream(context.Background(), true, func(item CatalogExportItem) error {
		items = append(items, item)
		return nil
	})
	if err != nil || count != 1 || len(items) != 1 {
		t.Fatalf("Stream() = count %d items %d err %v", count, len(items), err)
	}
	if !tx.committed || tx.rolledBack || tx.execCalls != 1 || tx.execSQL[0] != globalCatalogReadOnlyTransactionSQL {
		t.Fatalf("transaction = committed %t rolledBack %t exec %v", tx.committed, tx.rolledBack, tx.execSQL)
	}
	if len(tx.queryArgs) != 1 || len(tx.queryArgs[0]) != 1 || tx.queryArgs[0][0] != true {
		t.Fatalf("query args = %#v", tx.queryArgs)
	}
	if items[0].ID != id || items[0].AverageUnitWeightGrams == nil || *items[0].AverageUnitWeightGrams != weight ||
		items[0].Micros["Iron"] != json.Number("1.25") || len(items[0].FoodCategories) != 1 ||
		len(items[0].AllergenKeys) != 1 || items[0].AllergenKeys[0] != "dairy" {
		t.Fatalf("item = %#v", items[0])
	}
	if !containsAll(globalCatalogExportSQL, "FROM food_items f", "ORDER BY f.id", "food_item_classifications", "food_item_allergens", "curated_imports") ||
		containsAll(globalCatalogExportSQL, "custom_food_items") {
		t.Fatalf("unsafe or incomplete export SQL: %s", globalCatalogExportSQL)
	}
	for _, forbidden := range []string{" users ", "custom_food", "encrypted_", "login_methods", "saved_", "consent", "subscription", "email"} {
		if strings.Contains(strings.ToLower(globalCatalogExportSQL), forbidden) {
			t.Fatalf("export SQL references private/account data %q", forbidden)
		}
	}
}

func TestGlobalCatalogExportRepositoryCancellationAndFailuresRollback(t *testing.T) {
	sentinel := errors.New("failure")
	cases := []struct {
		name string
		db   *fakeSQLExecutor
	}{
		{"begin", &fakeSQLExecutor{beginErr: sentinel}},
		{"configure", &fakeSQLExecutor{tx: &fakeTx{fakeSQLExecutor: fakeSQLExecutor{execErr: sentinel}}}},
		{"query", &fakeSQLExecutor{tx: &fakeTx{fakeSQLExecutor: fakeSQLExecutor{queryErr: sentinel}}}},
		{"rows", &fakeSQLExecutor{tx: &fakeTx{fakeSQLExecutor: fakeSQLExecutor{rows: &fakeRows{err: sentinel}}}}},
		{"commit", &fakeSQLExecutor{tx: &fakeTx{fakeSQLExecutor: fakeSQLExecutor{rows: &fakeRows{}}, commitErr: sentinel}}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewPostgresGlobalCatalogExportRepository(test.db).Stream(context.Background(), false, func(CatalogExportItem) error { return nil }); err == nil {
				t.Fatal("failure path succeeded")
			}
		})
	}
	tx := &fakeTx{fakeSQLExecutor: fakeSQLExecutor{rows: &fakeRows{next: true, scanErr: sentinel}}}
	if _, err := NewPostgresGlobalCatalogExportRepository(&fakeSQLExecutor{tx: tx}).Stream(context.Background(), false, func(CatalogExportItem) error { return nil }); err == nil || !tx.rolledBack {
		t.Fatalf("scan failure = %v rolledBack=%t", err, tx.rolledBack)
	}
}

func TestPostgresGlobalCatalogExportRepositorySelectionOrderingAndPrivateIsolation(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	userID := createRepositoryUser(t, ctx, db, "catalog-export-owner@example.test")
	activeID := uuid.MustParse("00000000-0000-4000-8000-000000000010")
	deletedID := uuid.MustParse("00000000-0000-4000-8000-000000000020")
	curatedID := uuid.MustParse("00000000-0000-4000-8000-000000000060")
	classifications := NewPostgresClassificationRepository(db)
	categoryZulu, err := classifications.Upsert(ctx, ClassificationEntity{Name: "Zulu Category", Kind: ClassificationKindFoodCategory})
	if err != nil {
		t.Fatal(err)
	}
	categoryAlpha, err := classifications.Upsert(ctx, ClassificationEntity{Name: "Alpha Category", Kind: ClassificationKindFoodCategory})
	if err != nil {
		t.Fatal(err)
	}
	roleID, err := classifications.Upsert(ctx, ClassificationEntity{Name: "Main Role", Kind: ClassificationKindCulinaryRole})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(ctx, globalCatalogExportFixtureCreateSQL, activeID, deletedID, categoryZulu, categoryAlpha, roleID, curatedID); err != nil {
		t.Fatal(err)
	}
	if _, err := NewPostgresCustomFoodItemRepository(db).Create(ctx, CustomFoodItemEntity{
		OwnerID: userID,
		FoodItemEntity: FoodItemEntity{
			Name: "Private secret item", PhysicalState: PhysicalStateSolid,
			MacrosPer100: MacroValues{Protein: 1, Carbohydrates: 2, Fat: 3}, Micros: MicroValues{},
		},
	}); err != nil {
		t.Fatal(err)
	}
	exporter := NewPostgresGlobalCatalogExportRepository(db)
	var active []CatalogExportItem
	if count, err := exporter.Stream(ctx, false, func(item CatalogExportItem) error {
		active = append(active, item)
		return nil
	}); err != nil || count != 1 {
		t.Fatalf("active Stream() = count %d err %v", count, err)
	}
	item := active[0]
	if item.ID != activeID || item.Name != "Żółty płyn" || item.PhysicalState != PhysicalStateLiquid ||
		item.AverageServingVolumeMilliliters == nil || *item.AverageServingVolumeMilliliters != "250.5000" ||
		item.DensityGramsPerMilliliter == nil || *item.DensityGramsPerMilliliter != "1.030000" ||
		len(item.FoodCategories) != 2 || item.FoodCategories[0].Name != "Alpha Category" ||
		len(item.CulinaryRoles) != 1 || len(item.AllergenKeys) != 2 ||
		item.AllergenKeys[0] != "dairy" || item.AllergenKeys[1] != "gluten" ||
		len(item.CuratedSources) != 1 || item.ImageAlt == nil || *item.ImageAlt != "Żółty obraz" {
		t.Fatalf("active export = %#v", item)
	}
	var complete []CatalogExportItem
	if count, err := exporter.Stream(ctx, true, func(item CatalogExportItem) error {
		complete = append(complete, item)
		return nil
	}); err != nil || count != 2 {
		t.Fatalf("complete Stream() = count %d err %v", count, err)
	}
	if complete[0].ID != activeID || complete[1].ID != deletedID || complete[1].DeletedAt == nil {
		t.Fatalf("complete ordering/deletion = %#v", complete)
	}
	for _, exported := range complete {
		if exported.Name == "Private secret item" {
			t.Fatal("private custom item was exported")
		}
	}
}

func containsAll(value string, needles ...string) bool {
	for _, needle := range needles {
		if !strings.Contains(value, needle) {
			return false
		}
	}
	return true
}
