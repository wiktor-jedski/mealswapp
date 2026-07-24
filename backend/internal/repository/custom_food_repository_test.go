package repository

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

// TestPostgresCustomFoodItemRepositoryAtomicCreateClaim verifies one durable side effect across concurrent retries and rollback failures.
func TestPostgresCustomFoodItemRepositoryAtomicCreateClaim(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	ownerID := createRepositoryUser(t, ctx, db, "custom-claim@example.test")
	repo := NewPostgresCustomFoodItemRepository(db)
	item := CustomFoodItemEntity{OwnerID: ownerID, FoodItemEntity: FoodItemEntity{
		Name: "Atomic tofu", PhysicalState: PhysicalStateSolid, MacrosPer100: MacroValues{Protein: 10}, Micros: MicroValues{},
	}}
	encode := func(entity CustomFoodItemEntity) ([]byte, error) {
		return json.Marshal(map[string]any{"id": entity.ID, "name": entity.Name})
	}
	claim := CustomFoodItemCreateClaim{UserID: ownerID, Key: "atomic-custom-key", BodyHash: strings.Repeat("a", 64), Item: item}

	results := make(chan CustomFoodItemCreateClaimResult, 2)
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := repo.ClaimCreate(ctx, claim, encode)
			results <- result
			errs <- err
		}()
	}
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	var first []byte
	for result := range results {
		if first == nil {
			first = result.ResponseBody
		} else if !bytes.Equal(first, result.ResponseBody) {
			t.Fatalf("concurrent responses differ: %s != %s", first, result.ResponseBody)
		}
	}
	items, err := repo.List(ctx, ownerID, RepositoryContext{})
	if err != nil || len(items) != 1 {
		t.Fatalf("concurrent items = %#v err=%v", items, err)
	}
	changed := claim
	changed.BodyHash = strings.Repeat("c", 64)
	changed.Item.Name = "Unexpected duplicate"
	if _, err := repo.ClaimCreate(ctx, changed, encode); !IsKind(err, ErrorKindIdempotencyConflict) {
		t.Fatalf("changed-body claim error = %v", err)
	}
	items, err = repo.List(ctx, ownerID, RepositoryContext{})
	if err != nil || len(items) != 1 {
		t.Fatalf("changed-body claim side effects = %#v err=%v", items, err)
	}
	duplicateName := claim
	duplicateName.Key = "duplicate-name-key"
	duplicateName.BodyHash = strings.Repeat("e", 64)
	duplicateName.Item.Name = "  aToMiC ToFu  "
	if _, err := repo.ClaimCreate(ctx, duplicateName, encode); !IsKind(err, ErrorKindConflict) || IsKind(err, ErrorKindIdempotencyConflict) {
		t.Fatalf("duplicate-name claim error = %v, want resource conflict", err)
	}
	if _, err := NewPostgresCheckoutIdempotencyRepository(db).GetCheckoutIdempotency(ctx, ownerID, "POST", "/custom-items", duplicateName.Key); !IsKind(err, ErrorKindNotFound) {
		t.Fatalf("duplicate-name conflict left idempotency claim: %v", err)
	}
	items, err = repo.List(ctx, ownerID, RepositoryContext{})
	if err != nil || len(items) != 1 {
		t.Fatalf("duplicate-name claim side effects = %#v err=%v", items, err)
	}

	rollbackOwner := createRepositoryUser(t, ctx, db, "custom-claim-rollback@example.test")
	rollbackItem := item
	rollbackItem.OwnerID = rollbackOwner
	rollbackItem.Name = "Rollback tofu"
	rollbackClaim := CustomFoodItemCreateClaim{UserID: rollbackOwner, Key: "rollback-custom-key", BodyHash: strings.Repeat("b", 64), Item: rollbackItem}
	if _, err := repo.ClaimCreate(ctx, rollbackClaim, func(CustomFoodItemEntity) ([]byte, error) { return nil, errors.New("response failure") }); !IsKind(err, ErrorKindInternal) {
		t.Fatalf("response failure error = %v", err)
	}
	items, err = repo.List(ctx, rollbackOwner, RepositoryContext{})
	if err != nil || len(items) != 0 {
		t.Fatalf("rollback left items = %#v err=%v", items, err)
	}
	if result, err := repo.ClaimCreate(ctx, rollbackClaim, encode); err != nil || result.Replayed {
		t.Fatalf("retry after rollback = %+v err=%v", result, err)
	}

	invalidOwner := createRepositoryUser(t, ctx, db, "custom-claim-invalid-micro@example.test")
	invalidItem := item
	invalidItem.OwnerID = invalidOwner
	invalidItem.Name = "Invalid micro"
	invalidItem.Micros = MicroValues{"Na": 1}
	invalidClaim := CustomFoodItemCreateClaim{UserID: invalidOwner, Key: "invalid-micro-key", BodyHash: strings.Repeat("d", 64), Item: invalidItem}
	if _, err := repo.ClaimCreate(ctx, invalidClaim, encode); !IsKind(err, ErrorKindInvalidMicronutrientKey) {
		t.Fatalf("invalid micronutrient claim error = %v", err)
	}
	if _, err := NewPostgresCheckoutIdempotencyRepository(db).GetCheckoutIdempotency(ctx, invalidOwner, "POST", "/custom-items", invalidClaim.Key); !IsKind(err, ErrorKindNotFound) {
		t.Fatalf("invalid micronutrient left idempotency claim: %v", err)
	}
}

// Implements DESIGN-005 FoodItemEntity ownerless-row rejection fixture.
//
//go:embed sql/testdata/custom_food_ownerless_create.sql
var testCustomFoodOwnerlessCreateSQL string

// TestPostgresCustomFoodItemRepositoryOwnerScopedCRUD verifies DESIGN-005 FoodItemEntity private-item isolation and invariants.
func TestPostgresCustomFoodItemRepositoryOwnerScopedCRUD(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	ownerA := createRepositoryUser(t, ctx, db, "custom-a@example.test")
	ownerB := createRepositoryUser(t, ctx, db, "custom-b@example.test")
	customRepo := NewPostgresCustomFoodItemRepository(db)
	globalRepo := NewPostgresFoodItemRepository(db)
	classificationRepo := NewPostgresClassificationRepository(db)

	categoryRootID, err := classificationRepo.Upsert(ctx, ClassificationEntity{Name: "Custom Protein", Kind: ClassificationKindFoodCategory})
	if err != nil {
		t.Fatalf("create food category: %v", err)
	}
	categoryID, err := classificationRepo.Upsert(ctx, ClassificationEntity{Name: "Custom Plant Protein", Kind: ClassificationKindFoodCategory, ParentID: &categoryRootID})
	if err != nil {
		t.Fatalf("create child food category: %v", err)
	}
	roleID, err := classificationRepo.Upsert(ctx, ClassificationEntity{Name: "Custom Quick", Kind: ClassificationKindCulinaryRole})
	if err != nil {
		t.Fatalf("create culinary role: %v", err)
	}

	globalID, err := globalRepo.Create(ctx, FoodItemEntity{Name: "My Tofu", PhysicalState: PhysicalStateSolid, MacrosPer100: MacroValues{Protein: 8}})
	if err != nil {
		t.Fatalf("create same-named global food: %v", err)
	}
	item := CustomFoodItemEntity{
		OwnerID: ownerA,
		FoodItemEntity: FoodItemEntity{
			Name:                   "My Tofu",
			PhysicalState:          PhysicalStateSolid,
			PrepTimeMinutes:        4,
			AverageUnitWeightGrams: 28.3495,
			MacrosPer100:           MacroValues{Protein: 9, Carbohydrates: 2, Fat: 4},
			Micros:                 MicroValues{"Sodium": 7},
			FoodCategories:         []ClassificationEntity{{ID: categoryID, Kind: ClassificationKindFoodCategory}},
			CulinaryRoles:          []ClassificationEntity{{ID: roleID, Kind: ClassificationKindCulinaryRole}},
		},
	}
	customID, err := customRepo.Create(ctx, item)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if customID == globalID {
		t.Fatal("custom item unexpectedly shares global identity")
	}

	stored, err := customRepo.GetByID(ctx, ownerA, customID, RepositoryContext{UnitSystem: UnitSystemMetric})
	if err != nil {
		t.Fatalf("GetByID() same owner error = %v", err)
	}
	if stored.OwnerID != ownerA || stored.Name != "My Tofu" || stored.MacrosPer100.Protein != 9 || stored.Micros["Sodium"] != 7 {
		t.Fatalf("stored custom item = %#v", stored)
	}
	if len(stored.FoodCategories) != 1 || stored.FoodCategories[0].ID != categoryID || stored.FoodCategories[0].ParentID == nil || *stored.FoodCategories[0].ParentID != categoryRootID || len(stored.CulinaryRoles) != 1 || stored.CulinaryRoles[0].ID != roleID {
		t.Fatalf("stored custom classifications = categories %#v roles %#v", stored.FoodCategories, stored.CulinaryRoles)
	}
	imperial, err := customRepo.GetByID(ctx, ownerA, customID, RepositoryContext{UnitSystem: UnitSystemImperial})
	if err != nil {
		t.Fatalf("GetByID() imperial error = %v", err)
	}
	if imperial.AverageUnitWeightGrams != 1 {
		t.Fatalf("imperial unit weight = %v, want 1 oz", imperial.AverageUnitWeightGrams)
	}

	if _, err := customRepo.GetByID(ctx, ownerB, customID, RepositoryContext{}); !IsKind(err, ErrorKindNotFound) {
		t.Fatalf("GetByID() cross-owner error = %v, want not found", err)
	}
	if _, err := globalRepo.GetByID(ctx, customID, RepositoryContext{}); !IsKind(err, ErrorKindNotFound) {
		t.Fatalf("global GetByID(customID) error = %v, want not found", err)
	}
	if _, err := customRepo.GetByID(ctx, ownerA, globalID, RepositoryContext{}); !IsKind(err, ErrorKindNotFound) {
		t.Fatalf("custom GetByID(globalID) error = %v, want not found", err)
	}

	normalizedDuplicate := item
	normalizedDuplicate.Name = "  mY tOfU  "
	if _, err := customRepo.Create(ctx, normalizedDuplicate); !IsKind(err, ErrorKindConflict) {
		t.Fatalf("Create() normalized same-owner duplicate error = %v, want conflict", err)
	}
	item.OwnerID = ownerB
	otherOwnerID, err := customRepo.Create(ctx, item)
	if err != nil {
		t.Fatalf("Create() different-owner duplicate error = %v", err)
	}
	if otherOwnerID == customID {
		t.Fatal("different-owner item reused custom item ID")
	}
	ownerAItems, err := customRepo.List(ctx, ownerA, RepositoryContext{UnitSystem: UnitSystemMetric})
	if err != nil {
		t.Fatalf("List() owner A error = %v", err)
	}
	if len(ownerAItems) != 1 || ownerAItems[0].ID != customID || ownerAItems[0].OwnerID != ownerA {
		t.Fatalf("List() owner A items = %#v", ownerAItems)
	}
	ownerBItems, err := customRepo.List(ctx, ownerB, RepositoryContext{UnitSystem: UnitSystemMetric})
	if err != nil {
		t.Fatalf("List() owner B error = %v", err)
	}
	if len(ownerBItems) != 1 || ownerBItems[0].ID != otherOwnerID || ownerBItems[0].ID == globalID {
		t.Fatalf("List() owner B items = %#v", ownerBItems)
	}

	stored.OwnerID = ownerB
	stored.Name = "Cross-owner overwrite"
	if err := customRepo.Update(ctx, stored); !IsKind(err, ErrorKindNotFound) {
		t.Fatalf("Update() cross-owner error = %v, want not found", err)
	}
	if err := customRepo.Delete(ctx, ownerB, customID); !IsKind(err, ErrorKindNotFound) {
		t.Fatalf("Delete() cross-owner error = %v, want not found", err)
	}

	stored.OwnerID = ownerA
	stored.Name = "Updated Tofu"
	stored.MacrosPer100.Protein = 10
	stored.FoodCategories = nil
	if err := customRepo.Update(ctx, stored); err != nil {
		t.Fatalf("Update() same owner error = %v", err)
	}
	updated, err := customRepo.GetByID(ctx, ownerA, customID, RepositoryContext{})
	if err != nil {
		t.Fatalf("GetByID() updated error = %v", err)
	}
	if updated.Name != "Updated Tofu" || updated.MacrosPer100.Protein != 10 || len(updated.FoodCategories) != 0 || len(updated.CulinaryRoles) != 1 {
		t.Fatalf("updated custom item = %#v", updated)
	}

	inUse, err := classificationRepo.IsInUse(ctx, roleID)
	if err != nil || !inUse {
		t.Fatalf("IsInUse() custom classification = %v, %v; want true, nil", inUse, err)
	}
	if err := customRepo.Delete(ctx, ownerA, customID); err != nil {
		t.Fatalf("Delete() same owner error = %v", err)
	}
	if _, err := customRepo.GetByID(ctx, ownerA, customID, RepositoryContext{}); !IsKind(err, ErrorKindNotFound) {
		t.Fatalf("GetByID() deleted error = %v, want not found", err)
	}
	deleted, err := customRepo.GetByID(ctx, ownerA, customID, RepositoryContext{IncludeDeleted: true})
	if err != nil || deleted.DeletedAt == nil {
		t.Fatalf("GetByID() include deleted = %#v, %v", deleted, err)
	}
	item.OwnerID = ownerA
	item.Name = "Updated Tofu"
	if _, err := customRepo.Create(ctx, item); err != nil {
		t.Fatalf("Create() after soft delete error = %v", err)
	}

	if _, err := customRepo.Create(ctx, CustomFoodItemEntity{FoodItemEntity: item.FoodItemEntity}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("Create() ownerless error = %v, want validation", err)
	}
	if _, err := customRepo.GetByID(ctx, uuid.Nil, customID, RepositoryContext{}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("GetByID() ownerless error = %v, want validation", err)
	}
	if _, err := db.Exec(ctx, testCustomFoodOwnerlessCreateSQL); err == nil {
		t.Fatal("direct ownerless insert error = nil, want not-null constraint violation")
	}
}

// TestPostgresCustomFoodItemRepositoryErrorBranches verifies DESIGN-005 owner-scoped error and cleanup paths.
func TestPostgresCustomFoodItemRepositoryErrorBranches(t *testing.T) {
	ctx := context.Background()
	wantErr := errors.New("database failure")
	ownerID := uuid.New()
	itemID := uuid.New()
	values := customFoodFixtureValues(itemID)

	if _, err := NewPostgresCustomFoodItemRepository(nil).GetByID(ctx, ownerID, uuid.Nil, RepositoryContext{}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("GetByID() missing item id error = %v, want validation", err)
	}
	invalidMicros := append([]any(nil), values...)
	invalidMicros[13] = []byte(`[`)
	repo := NewPostgresCustomFoodItemRepository(&fakeSQLExecutor{row: fakeRow{values: invalidMicros}})
	if _, err := repo.GetByID(ctx, ownerID, itemID, RepositoryContext{}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("GetByID() invalid micros error = %v, want validation", err)
	}
	repo = NewPostgresCustomFoodItemRepository(&fakeSQLExecutor{row: fakeRow{err: wantErr}})
	if _, err := repo.GetByID(ctx, ownerID, itemID, RepositoryContext{}); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("GetByID() scan error = %v, want connection", err)
	}
	repo = NewPostgresCustomFoodItemRepository(&fakeSQLExecutor{row: fakeRow{values: values}, queryErr: wantErr})
	if _, err := repo.GetByID(ctx, ownerID, itemID, RepositoryContext{}); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("GetByID() hydrate error = %v, want connection", err)
	}

	valid := CustomFoodItemEntity{OwnerID: ownerID, FoodItemEntity: FoodItemEntity{ID: itemID, Name: "Custom", PhysicalState: PhysicalStateSolid}}
	missingID := valid
	missingID.ID = uuid.Nil
	if err := NewPostgresCustomFoodItemRepository(nil).Update(ctx, missingID); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("Update() missing item id error = %v, want validation", err)
	}
	invalid := valid
	invalid.Name = ""
	if err := NewPostgresCustomFoodItemRepository(nil).Update(ctx, invalid); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("Update() invalid item error = %v, want validation", err)
	}
	repo = NewPostgresCustomFoodItemRepository(&fakeSQLExecutor{rows: &fakeRows{}, tx: &fakeTx{fakeSQLExecutor: fakeSQLExecutor{execErr: wantErr}}})
	if err := repo.Update(ctx, valid); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("Update() exec error = %v, want connection", err)
	}
	rollbackTx := &fakeTx{fakeSQLExecutor: fakeSQLExecutor{
		execErrs: []error{nil, wantErr},
		execTags: []pgconn.CommandTag{pgconn.NewCommandTag("UPDATE 1")},
	}}
	repo = NewPostgresCustomFoodItemRepository(&fakeSQLExecutor{rows: &fakeRows{}, tx: rollbackTx})
	if err := repo.Update(ctx, valid); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("Update() classification replacement error = %v, want connection", err)
	}
	if !rollbackTx.rolledBack {
		t.Fatal("Update() classification replacement error did not roll back")
	}

	if err := NewPostgresCustomFoodItemRepository(nil).Delete(ctx, ownerID, uuid.Nil); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("Delete() missing item id error = %v, want validation", err)
	}
	repo = NewPostgresCustomFoodItemRepository(&fakeSQLExecutor{execErr: wantErr})
	if err := repo.Delete(ctx, ownerID, itemID); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("Delete() exec error = %v, want connection", err)
	}

	item := FoodItemEntity{ID: itemID}
	repo = NewPostgresCustomFoodItemRepository(&fakeSQLExecutor{queryErr: wantErr})
	if err := repo.hydrateClassifications(ctx, &item); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("hydrateClassifications() query error = %v, want connection", err)
	}
	repo = NewPostgresCustomFoodItemRepository(&fakeSQLExecutor{rows: &fakeRows{next: true, scanErr: wantErr}})
	if err := repo.hydrateClassifications(ctx, &item); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("hydrateClassifications() scan error = %v, want connection", err)
	}
	repo = NewPostgresCustomFoodItemRepository(&fakeSQLExecutor{rows: &fakeRows{err: wantErr}})
	if err := repo.hydrateClassifications(ctx, &item); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("hydrateClassifications() rows error = %v, want connection", err)
	}
	repo = NewPostgresCustomFoodItemRepository(&fakeSQLExecutor{rows: &fakeRows{next: true, values: []any{uuid.New(), "Unknown", ClassificationKind("unknown"), (*uuid.UUID)(nil)}}})
	if err := repo.hydrateClassifications(ctx, &item); err != nil {
		t.Fatalf("hydrateClassifications() unknown kind error = %v", err)
	}
	if len(item.FoodCategories) != 0 || len(item.CulinaryRoles) != 0 {
		t.Fatalf("hydrateClassifications() retained unknown kind: %#v", item)
	}

	repo = NewPostgresCustomFoodItemRepository(&fakeSQLExecutor{execErr: wantErr})
	if err := repo.replaceClassifications(ctx, itemID, nil, nil); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("replaceClassifications() clear error = %v, want connection", err)
	}
	repo = NewPostgresCustomFoodItemRepository(&fakeSQLExecutor{execErrs: []error{nil, wantErr}})
	if err := repo.replaceClassifications(ctx, itemID, []ClassificationEntity{{ID: uuid.New()}}, nil); !IsKind(err, ErrorKindConnection) {
		t.Fatalf("replaceClassifications() attach error = %v, want connection", err)
	}
}

// TestReplaceCustomFoodClassificationsPreservesInputs verifies DESIGN-005 classification input immutability.
func TestReplaceCustomFoodClassificationsPreservesInputs(t *testing.T) {
	categoryID := uuid.New()
	sentinelID := uuid.New()
	roleID := uuid.New()
	backing := make([]ClassificationEntity, 2)
	backing[0] = ClassificationEntity{ID: categoryID}
	backing[1] = ClassificationEntity{ID: sentinelID}
	foodCategories := backing[:1]
	culinaryRoles := []ClassificationEntity{{ID: roleID}}

	repo := NewPostgresCustomFoodItemRepository(&fakeSQLExecutor{})
	if err := repo.replaceClassifications(context.Background(), uuid.New(), foodCategories, culinaryRoles); err != nil {
		t.Fatalf("replaceClassifications() error = %v", err)
	}
	if backing[1].ID != sentinelID {
		t.Fatalf("replaceClassifications() mutated food-category backing array: got %s, want %s", backing[1].ID, sentinelID)
	}
	if culinaryRoles[0].ID != roleID {
		t.Fatalf("replaceClassifications() mutated culinary roles: got %s, want %s", culinaryRoles[0].ID, roleID)
	}
}

func customFoodFixtureValues(id uuid.UUID) []any {
	now := time.Now()
	return []any{
		id, "Custom", PhysicalStateSolid, 0,
		floatPtr(100), (*float64)(nil), (*float64)(nil),
		(*string)(nil), (*string)(nil), (*string)(nil),
		1.0, 2.0, 3.0, []byte(`{}`), (*string)(nil),
		(*time.Time)(nil), now, now,
	}
}

// TestPostgresCustomFoodItemRepositoryValidation verifies DESIGN-005 macro, micronutrient, and liquid-density invariants.
func TestPostgresCustomFoodItemRepositoryValidation(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	ownerID := createRepositoryUser(t, ctx, db, "custom-validation@example.test")
	repo := NewPostgresCustomFoodItemRepository(db)

	validLiquid := CustomFoodItemEntity{OwnerID: ownerID, FoodItemEntity: FoodItemEntity{
		Name:                            "Custom Oat Drink",
		PhysicalState:                   PhysicalStateLiquid,
		AverageServingVolumeMilliliters: 250,
		DensityGramsPerMilliliter:       1.03,
		DensitySourceProvider:           "manual-entry",
		DensitySourceKind:               "manual",
		MacrosPer100:                    MacroValues{Protein: 1, Carbohydrates: 8, Fat: 2},
		Micros:                          MicroValues{"Sodium": 4},
	}}
	liquidID, err := repo.Create(ctx, validLiquid)
	if err != nil {
		t.Fatalf("Create() valid liquid error = %v", err)
	}
	liquid, err := repo.GetByID(ctx, ownerID, liquidID, RepositoryContext{UnitSystem: UnitSystemImperial})
	if err != nil {
		t.Fatalf("GetByID() liquid error = %v", err)
	}
	if liquid.AverageServingVolumeMilliliters != 8.4535 || liquid.DensityGramsPerMilliliter != 1.03 || liquid.DensitySourceKind != "manual" {
		t.Fatalf("imperial liquid fields = %#v", liquid.FoodItemEntity)
	}

	invalidCases := []struct {
		name string
		item CustomFoodItemEntity
		kind ErrorKind
	}{
		{name: "negative macro", item: CustomFoodItemEntity{OwnerID: ownerID, FoodItemEntity: FoodItemEntity{Name: "Negative", PhysicalState: PhysicalStateSolid, MacrosPer100: MacroValues{Protein: -1}}}, kind: ErrorKindValidation},
		{name: "invalid micronutrient", item: CustomFoodItemEntity{OwnerID: ownerID, FoodItemEntity: FoodItemEntity{Name: "Alias", PhysicalState: PhysicalStateSolid, Micros: MicroValues{"Na": 1}}}, kind: ErrorKindInvalidMicronutrientKey},
		{name: "missing liquid density", item: CustomFoodItemEntity{OwnerID: ownerID, FoodItemEntity: FoodItemEntity{Name: "No Density", PhysicalState: PhysicalStateLiquid}}, kind: ErrorKindValidation},
		{name: "missing liquid provenance", item: CustomFoodItemEntity{OwnerID: ownerID, FoodItemEntity: FoodItemEntity{Name: "No Provenance", PhysicalState: PhysicalStateLiquid, DensityGramsPerMilliliter: 1}}, kind: ErrorKindValidation},
	}
	nulProvenance := validLiquid
	nulProvenance.Name = "NUL provenance"
	nulProvenance.DensitySourceProvider = "invalid\x00provider"
	invalidCases = append(invalidCases, struct {
		name string
		item CustomFoodItemEntity
		kind ErrorKind
	}{name: "postgres text NUL", item: nulProvenance, kind: ErrorKindValidation})
	if _, err := db.Exec(ctx, testInactiveVocabularyUpsertSQL); err != nil {
		t.Fatalf("insert inactive vocabulary: %v", err)
	}
	invalidCases = append(invalidCases, struct {
		name string
		item CustomFoodItemEntity
		kind ErrorKind
	}{name: "inactive micronutrient", item: CustomFoodItemEntity{OwnerID: ownerID, FoodItemEntity: FoodItemEntity{Name: "Inactive", PhysicalState: PhysicalStateSolid, Micros: MicroValues{"Legacy": 1}}}, kind: ErrorKindInvalidMicronutrientKey})
	for _, test := range invalidCases {
		t.Run(test.name, func(t *testing.T) {
			if _, err := repo.Create(ctx, test.item); !IsKind(err, test.kind) {
				t.Fatalf("Create() error = %v, want %s", err, test.kind)
			}
		})
	}
}
