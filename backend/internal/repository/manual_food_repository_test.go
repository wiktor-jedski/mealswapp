package repository

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// TestPostgresManualFoodItemCRUD verifies atomic idempotency, audit, search, rollback, and private isolation.
// Implements DESIGN-009 ItemCurator integration behavior.
// TestPostgresManualFoodItemCRUD verifies IT-ARCH-009-006, ARCH-009,
// DESIGN-009 ItemCurator, and SW-REQ-056 with real PostgreSQL persistence.
func TestPostgresManualFoodItemCRUD(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	adminID := createRepositoryUser(t, ctx, db, "manual-curator@example.test")
	manualRepo := NewPostgresManualFoodItemRepository(db)
	foodRepo := NewPostgresFoodItemRepository(db)
	customRepo := NewPostgresCustomFoodItemRepository(db)
	auditRepo := NewPostgresAdminImportAuditRepository(db)
	classificationRepo := NewPostgresClassificationRepository(db)
	categoryID, err := classificationRepo.Upsert(ctx, ClassificationEntity{Name: "Manual protein", Kind: ClassificationKindFoodCategory})
	if err != nil {
		t.Fatal(err)
	}
	roleID, err := classificationRepo.Upsert(ctx, ClassificationEntity{Name: "Manual staple", Kind: ClassificationKindCulinaryRole})
	if err != nil {
		t.Fatal(err)
	}
	item := FoodItemEntity{
		Name: "Manual global tofu", PhysicalState: PhysicalStateSolid, PrepTimeMinutes: 5, AverageUnitWeightGrams: 100,
		MacrosPer100: MacroValues{Protein: 18, Carbohydrates: 3, Fat: 9}, Micros: MicroValues{}, ImageURL: "https://images.example.test/tofu.png",
		FoodCategories: []ClassificationEntity{{ID: categoryID, Kind: ClassificationKindFoodCategory}},
		CulinaryRoles:  []ClassificationEntity{{ID: roleID, Kind: ClassificationKindCulinaryRole}},
	}
	claim := ManualFoodItemCreateClaim{AdminUserID: adminID, Key: "manual-global-key-0001", BodyHash: strings.Repeat("a", 64), Item: item}
	encode := func(entity FoodItemEntity) ([]byte, error) {
		return json.Marshal(map[string]any{"id": entity.ID, "name": entity.Name, "physicalState": entity.PhysicalState})
	}
	var created ManualFoodItemCreateClaimResult
	var itemID uuid.UUID
	err = auditRepo.WithMutationAudit(ctx, AdminAuditEntry{AdminUserID: adminID, Action: "manual_create", EntityType: "food_item", RequestID: uuid.NewString()}, func(tx AdminMutationExecutor) (AdminAuditChanges, error) {
		var mutationErr error
		created, mutationErr = manualRepo.ClaimCreate(ctx, tx, claim, encode)
		if mutationErr != nil {
			return AdminAuditChanges{}, mutationErr
		}
		var response struct {
			ID uuid.UUID `json:"id"`
		}
		if mutationErr = json.Unmarshal(created.ResponseBody, &response); mutationErr != nil {
			return AdminAuditChanges{}, mutationErr
		}
		itemID = response.ID
		return AdminAuditChanges{EntityID: &itemID, After: []byte(`{"active":true,"physicalState":"solid"}`)}, nil
	})
	if err != nil || itemID == uuid.Nil || created.StatusCode != 201 || created.Replayed {
		t.Fatalf("create result=%+v id=%s err=%v", created, itemID, err)
	}
	stored, err := manualRepo.GetByID(ctx, itemID, false)
	if err != nil || stored.Name != item.Name || stored.ImageURL != item.ImageURL || len(stored.FoodCategories) != 1 || len(stored.CulinaryRoles) != 1 {
		t.Fatalf("stored=%+v err=%v", stored, err)
	}
	assertManualFoodSearch(t, ctx, foodRepo, item.Name, itemID, true)

	var replay ManualFoodItemCreateClaimResult
	err = auditRepo.WithMutationAudit(ctx, AdminAuditEntry{AdminUserID: adminID, Action: "manual_create", EntityType: "food_item", RequestID: uuid.NewString()}, func(tx AdminMutationExecutor) (AdminAuditChanges, error) {
		var replayErr error
		replay, replayErr = manualRepo.ClaimCreate(ctx, tx, claim, encode)
		return AdminAuditChanges{Replayed: replay.Replayed}, replayErr
	})
	if err != nil || !replay.Replayed || string(replay.ResponseBody) != string(created.ResponseBody) {
		t.Fatalf("replay=%+v err=%v", replay, err)
	}
	audits, err := auditRepo.ListAuditForEntity(ctx, "food_item", itemID)
	if err != nil || len(audits) != 1 {
		t.Fatalf("create audits=%+v err=%v", audits, err)
	}

	changedClaim := claim
	changedClaim.BodyHash = strings.Repeat("b", 64)
	changedClaim.Item.Name = "Changed key body"
	err = auditRepo.WithMutationAudit(ctx, AdminAuditEntry{AdminUserID: adminID, Action: "manual_create", EntityType: "food_item", RequestID: uuid.NewString()}, func(tx AdminMutationExecutor) (AdminAuditChanges, error) {
		_, mutationErr := manualRepo.ClaimCreate(ctx, tx, changedClaim, encode)
		return AdminAuditChanges{}, mutationErr
	})
	if !IsKind(err, ErrorKindIdempotencyConflict) {
		t.Fatalf("changed key error=%v", err)
	}
	duplicateClaim := claim
	duplicateClaim.Key = "manual-duplicate-key"
	duplicateClaim.BodyHash = strings.Repeat("c", 64)
	err = auditRepo.WithMutationAudit(ctx, AdminAuditEntry{AdminUserID: adminID, Action: "manual_create", EntityType: "food_item", RequestID: uuid.NewString()}, func(tx AdminMutationExecutor) (AdminAuditChanges, error) {
		_, mutationErr := manualRepo.ClaimCreate(ctx, tx, duplicateClaim, encode)
		return AdminAuditChanges{}, mutationErr
	})
	if !IsKind(err, ErrorKindConflict) || IsKind(err, ErrorKindIdempotencyConflict) {
		t.Fatalf("duplicate name error=%v", err)
	}

	invalidItems := []FoodItemEntity{
		{Name: "Invalid macros", PhysicalState: PhysicalStateSolid, MacrosPer100: MacroValues{Protein: 101}, Micros: MicroValues{}},
		{Name: "Invalid micro", PhysicalState: PhysicalStateSolid, MacrosPer100: MacroValues{}, Micros: MicroValues{"not_allowed": 1}},
		{Name: "Invalid classification", PhysicalState: PhysicalStateSolid, MacrosPer100: MacroValues{}, Micros: MicroValues{}, FoodCategories: []ClassificationEntity{{ID: uuid.New(), Kind: ClassificationKindFoodCategory}}},
		{Name: "Invalid liquid", PhysicalState: PhysicalStateLiquid, MacrosPer100: MacroValues{}, Micros: MicroValues{}},
	}
	for index, invalid := range invalidItems {
		invalidClaim := ManualFoodItemCreateClaim{AdminUserID: adminID, Key: "manual-invalid-key-000" + string(rune('a'+index)), BodyHash: strings.Repeat("d", 64), Item: invalid}
		err = auditRepo.WithMutationAudit(ctx, AdminAuditEntry{AdminUserID: adminID, Action: "manual_create", EntityType: "food_item", RequestID: uuid.NewString()}, func(tx AdminMutationExecutor) (AdminAuditChanges, error) {
			_, mutationErr := manualRepo.ClaimCreate(ctx, tx, invalidClaim, encode)
			return AdminAuditChanges{}, mutationErr
		})
		if !IsKind(err, ErrorKindValidation) && !IsKind(err, ErrorKindInvalidMicronutrientKey) {
			t.Fatalf("invalid item %d error=%v", index, err)
		}
	}

	liquid := FoodItemEntity{Name: "Manual global milk", PhysicalState: PhysicalStateLiquid, AverageServingVolumeMilliliters: 250, DensityGramsPerMilliliter: 1.03, DensitySourceKind: "manual", MacrosPer100: MacroValues{Protein: 3}, Micros: MicroValues{}}
	liquidClaim := ManualFoodItemCreateClaim{AdminUserID: adminID, Key: "manual-liquid-key", BodyHash: strings.Repeat("f", 64), Item: liquid}
	err = auditRepo.WithMutationAudit(ctx, AdminAuditEntry{AdminUserID: adminID, Action: "manual_create", EntityType: "food_item", RequestID: uuid.NewString()}, func(tx AdminMutationExecutor) (AdminAuditChanges, error) {
		result, mutationErr := manualRepo.ClaimCreate(ctx, tx, liquidClaim, encode)
		if mutationErr != nil {
			return AdminAuditChanges{}, mutationErr
		}
		var response struct {
			ID uuid.UUID `json:"id"`
		}
		if mutationErr = json.Unmarshal(result.ResponseBody, &response); mutationErr != nil {
			return AdminAuditChanges{}, mutationErr
		}
		return AdminAuditChanges{EntityID: &response.ID, After: []byte(`{"active":true,"physicalState":"liquid"}`)}, nil
	})
	if err != nil {
		t.Fatalf("valid liquid create: %v", err)
	}

	privateID, err := customRepo.Create(ctx, CustomFoodItemEntity{OwnerID: adminID, FoodItemEntity: FoodItemEntity{Name: item.Name, PhysicalState: PhysicalStateSolid, MacrosPer100: MacroValues{Protein: 7}, Micros: MicroValues{}}})
	if err != nil {
		t.Fatalf("same-name private create: %v", err)
	}
	if _, err := customRepo.GetByID(ctx, adminID, itemID, RepositoryContext{}); !IsKind(err, ErrorKindNotFound) {
		t.Fatalf("private repository exposed global item: %v", err)
	}
	if _, err := manualRepo.GetByID(ctx, privateID, false); !IsKind(err, ErrorKindNotFound) {
		t.Fatalf("global repository exposed private item: %v", err)
	}
	var globalHasOwner bool
	if err := db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'food_items' AND column_name = 'owner_id')`).Scan(&globalHasOwner); err != nil || globalHasOwner {
		t.Fatalf("global owner column exists=%t err=%v", globalHasOwner, err)
	}

	updated := stored
	updated.Name = "Manual global tempeh"
	err = auditRepo.WithMutationAudit(ctx, AdminAuditEntry{AdminUserID: adminID, Action: "manual_update", EntityType: "food_item", RequestID: uuid.NewString()}, func(tx AdminMutationExecutor) (AdminAuditChanges, error) {
		before, mutationErr := manualRepo.GetByIDInMutation(ctx, tx, itemID, false)
		if mutationErr != nil {
			return AdminAuditChanges{}, mutationErr
		}
		if mutationErr = manualRepo.Update(ctx, tx, updated); mutationErr != nil {
			return AdminAuditChanges{}, mutationErr
		}
		after, mutationErr := manualRepo.GetByIDInMutation(ctx, tx, itemID, false)
		if mutationErr != nil || before.Name == after.Name {
			return AdminAuditChanges{}, errors.New("authoritative update snapshots missing")
		}
		return AdminAuditChanges{EntityID: &itemID, Before: []byte(`{"active":true,"physicalState":"solid"}`), After: []byte(`{"active":true,"physicalState":"solid"}`)}, nil
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	assertManualFoodSearch(t, ctx, foodRepo, item.Name, itemID, false)
	assertManualFoodSearch(t, ctx, foodRepo, updated.Name, itemID, true)

	rollbackItem := item
	rollbackItem.Name = "Manual audit rollback"
	rollbackClaim := ManualFoodItemCreateClaim{AdminUserID: adminID, Key: "manual-rollback-key", BodyHash: strings.Repeat("9", 64), Item: rollbackItem}
	err = auditRepo.WithMutationAudit(ctx, AdminAuditEntry{AdminUserID: adminID, Action: "manual_create", EntityType: "food_item", RequestID: uuid.NewString()}, func(tx AdminMutationExecutor) (AdminAuditChanges, error) {
		result, mutationErr := manualRepo.ClaimCreate(ctx, tx, rollbackClaim, encode)
		if mutationErr != nil {
			return AdminAuditChanges{}, mutationErr
		}
		var response struct {
			ID uuid.UUID `json:"id"`
		}
		_ = json.Unmarshal(result.ResponseBody, &response)
		return AdminAuditChanges{EntityID: &response.ID, After: []byte(`{"name":"must roll back"}`)}, nil
	})
	if !errors.Is(err, ErrAdminAuditPersistence) {
		t.Fatalf("audit rollback error=%v", err)
	}
	assertManualFoodSearch(t, ctx, foodRepo, rollbackItem.Name, uuid.Nil, false)

	err = auditRepo.WithMutationAudit(ctx, AdminAuditEntry{AdminUserID: adminID, Action: "manual_delete", EntityType: "food_item", RequestID: uuid.NewString()}, func(tx AdminMutationExecutor) (AdminAuditChanges, error) {
		if mutationErr := manualRepo.Delete(ctx, tx, itemID); mutationErr != nil {
			return AdminAuditChanges{}, mutationErr
		}
		return AdminAuditChanges{EntityID: &itemID, Before: []byte(`{"active":true,"physicalState":"solid"}`), After: []byte(`{"active":false,"deleted":true,"physicalState":"solid"}`)}, nil
	})
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	assertManualFoodSearch(t, ctx, foodRepo, updated.Name, itemID, false)
	if _, err := manualRepo.GetByID(ctx, itemID, false); !IsKind(err, ErrorKindNotFound) {
		t.Fatalf("deleted read error=%v", err)
	}
	audits, err = auditRepo.ListAuditForEntity(ctx, "food_item", itemID)
	if err != nil || len(audits) != 3 || len(audits[1].Before) == 0 || len(audits[1].After) == 0 || len(audits[2].Before) == 0 || len(audits[2].After) == 0 {
		t.Fatalf("final audits=%+v err=%v", audits, err)
	}
}

// assertManualFoodSearch verifies active catalog visibility for one exact name.
// Implements DESIGN-009 ItemCurator catalog propagation test helper.
func assertManualFoodSearch(t *testing.T, ctx context.Context, repo FoodItemRepository, name string, id uuid.UUID, want bool) {
	t.Helper()
	items, total, err := repo.Search(ctx, RepositoryQuery{Name: name, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range items {
		if (id == uuid.Nil || item.ID == id) && item.Name == name {
			found = true
		}
	}
	if found != want || want && total == 0 {
		t.Fatalf("search %q found=%t total=%d want=%t items=%+v", name, found, total, want, items)
	}
}
