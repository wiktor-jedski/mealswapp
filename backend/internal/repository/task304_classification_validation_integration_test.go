package repository

// Implements DESIGN-005 MicronutrientVocabulary and DESIGN-009 ItemCurator acceptance boundaries.

import (
	"context"
	_ "embed"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"
)

var (
	//go:embed sql/task304_count_food_items.sql
	task304CountFoodItemsSQL string
	//go:embed sql/task304_count_audit_entries.sql
	task304CountAuditEntriesSQL string
	//go:embed sql/task304_count_idempotency_keys.sql
	task304CountIdempotencyKeysSQL string
)

// TestTask304CanonicalValuesPersistAndInvalidValuesRollback proves the real
// repository accepts active canonical values and rejects inactive, unknown, or
// wrong-kind values before an item, audit row, or idempotency claim can commit.
func TestTask304CanonicalValuesPersistAndInvalidValuesRollback(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	adminID := createRepositoryUser(t, ctx, db, "task304-admin-"+uuid.NewString()+"@example.test")
	classificationRepo := NewPostgresClassificationRepository(db)
	categoryID, err := classificationRepo.Upsert(ctx, ClassificationEntity{Name: "Task 304 category", Kind: ClassificationKindFoodCategory})
	if err != nil {
		t.Fatal(err)
	}
	roleID, err := classificationRepo.Upsert(ctx, ClassificationEntity{Name: "Task 304 role", Kind: ClassificationKindCulinaryRole})
	if err != nil {
		t.Fatal(err)
	}
	vocabulary := NewPostgresMicronutrientVocabularyRepository(db)
	canonicalKey := "Task304" + uuid.NewString()[:8]
	if _, err := vocabulary.Create(ctx, MicronutrientVocabularyEntry{Key: canonicalKey, DisplayName: "Task 304 nutrient", Unit: "mg"}); err != nil {
		t.Fatal(err)
	}
	inactiveKey := "Task304Inactive" + uuid.NewString()[:8]
	if _, err := vocabulary.Create(ctx, MicronutrientVocabularyEntry{Key: inactiveKey, DisplayName: "Task 304 inactive", Unit: "mg"}); err != nil {
		t.Fatal(err)
	}
	manual := NewPostgresManualFoodItemRepository(db)
	audit := NewPostgresAdminImportAuditRepository(db)
	canonical := FoodItemEntity{
		Name: "Task 304 canonical values", PhysicalState: PhysicalStateSolid,
		MacrosPer100:   MacroValues{Protein: 10, Carbohydrates: 20, Fat: 3},
		Micros:         MicroValues{"Sodium": 125.5, canonicalKey: 7.25},
		FoodCategories: []ClassificationEntity{{ID: categoryID, Kind: ClassificationKindFoodCategory}},
		CulinaryRoles:  []ClassificationEntity{{ID: roleID, Kind: ClassificationKindCulinaryRole}},
		AllergenKeys:   []string{"peanut", "dairy"},
	}
	encode := func(entity FoodItemEntity) ([]byte, error) { return json.Marshal(entity) }
	claim := ManualFoodItemCreateClaim{AdminUserID: adminID, Key: "task304-canonical-" + uuid.NewString(), BodyHash: strings.Repeat("a", 64), Item: canonical}
	requestID := uuid.NewString()
	var itemID uuid.UUID
	err = audit.WithMutationAudit(ctx, AdminAuditEntry{ActorKind: AdminAuditActorAdministrator, AdminUserID: &adminID, Action: "manual_create", EntityType: "food_item", RequestID: requestID}, func(tx AdminMutationExecutor) (AdminAuditChanges, error) {
		created, mutationErr := manual.ClaimCreate(ctx, tx, claim, encode)
		if mutationErr != nil {
			return AdminAuditChanges{}, mutationErr
		}
		var response struct {
			ID uuid.UUID `json:"id"`
		}
		if err := json.Unmarshal(created.ResponseBody, &response); err != nil {
			return AdminAuditChanges{}, err
		}
		itemID = response.ID
		return AdminAuditChanges{EntityID: &itemID, After: []byte(`{"active":true,"physicalState":"solid"}`)}, nil
	})
	if err != nil {
		t.Fatalf("canonical create: %v", err)
	}
	stored, err := manual.GetByID(ctx, itemID, false)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(stored.Micros, canonical.Micros) || len(stored.FoodCategories) != 1 || stored.FoodCategories[0].ID != categoryID || len(stored.CulinaryRoles) != 1 || stored.CulinaryRoles[0].ID != roleID || !reflect.DeepEqual(stored.AllergenKeys, []string{"dairy", "peanut"}) {
		t.Fatalf("canonical persistence = %+v", stored)
	}
	audits, err := audit.ListAuditForEntity(ctx, "food_item", itemID)
	if err != nil || len(audits) != 1 || audits[0].Action != "manual_create" {
		t.Fatalf("canonical audit rows = %+v err=%v", audits, err)
	}

	if err := audit.WithMutationAudit(ctx, AdminAuditEntry{ActorKind: AdminAuditActorAdministrator, AdminUserID: &adminID, Action: "micronutrient.deactivate", EntityType: "micronutrient_vocabulary", RequestID: uuid.NewString()}, func(tx AdminMutationExecutor) (AdminAuditChanges, error) {
		if _, err := NewPostgresMicronutrientVocabularyRepository(tx).SetActive(ctx, inactiveKey, false); err != nil {
			return AdminAuditChanges{}, err
		}
		return AdminAuditChanges{After: []byte(`{"active":false,"keyDigest":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","unit":"mg"}`)}, nil
	}); err != nil {
		t.Fatal(err)
	}
	invalid := []struct {
		name string
		item FoodItemEntity
	}{
		{name: "alias", item: FoodItemEntity{Name: "Task 304 invalid alias", PhysicalState: PhysicalStateSolid, Micros: MicroValues{"Na": 1}}},
		{name: "unknown", item: FoodItemEntity{Name: "Task 304 invalid unknown", PhysicalState: PhysicalStateSolid, Micros: MicroValues{"NotCanonical": 1}}},
		{name: "inactive", item: FoodItemEntity{Name: "Task 304 invalid inactive", PhysicalState: PhysicalStateSolid, Micros: MicroValues{inactiveKey: 1}}},
		{name: "allergen", item: FoodItemEntity{Name: "Task 304 invalid allergen", PhysicalState: PhysicalStateSolid, AllergenKeys: []string{"not_in_vocabulary"}}},
		{name: "classification", item: FoodItemEntity{Name: "Task 304 invalid classification", PhysicalState: PhysicalStateSolid, FoodCategories: []ClassificationEntity{{ID: roleID, Kind: ClassificationKindFoodCategory}}}},
	}
	for _, testCase := range invalid {
		t.Run(testCase.name, func(t *testing.T) {
			requestID := uuid.NewString()
			invalidClaim := ManualFoodItemCreateClaim{AdminUserID: adminID, Key: "task304-invalid-" + uuid.NewString(), BodyHash: strings.Repeat("b", 64), Item: testCase.item}
			err := audit.WithMutationAudit(ctx, AdminAuditEntry{ActorKind: AdminAuditActorAdministrator, AdminUserID: &adminID, Action: "manual_create", EntityType: "food_item", RequestID: requestID}, func(tx AdminMutationExecutor) (AdminAuditChanges, error) {
				_, mutationErr := manual.ClaimCreate(ctx, tx, invalidClaim, encode)
				return AdminAuditChanges{}, mutationErr
			})
			if !IsKind(err, ErrorKindValidation) && !IsKind(err, ErrorKindInvalidMicronutrientKey) {
				t.Fatalf("invalid value error = %v", err)
			}
			var rowCount, auditCount, claimCount int
			if err := db.QueryRow(ctx, task304CountFoodItemsSQL, testCase.item.Name).Scan(&rowCount); err != nil {
				t.Fatal(err)
			}
			if err := db.QueryRow(ctx, task304CountAuditEntriesSQL, requestID).Scan(&auditCount); err != nil {
				t.Fatal(err)
			}
			if err := db.QueryRow(ctx, task304CountIdempotencyKeysSQL, invalidClaim.Key).Scan(&claimCount); err != nil {
				t.Fatal(err)
			}
			if rowCount != 0 || auditCount != 0 || claimCount != 0 {
				t.Fatalf("invalid value mutated row=%d audit=%d claim=%d", rowCount, auditCount, claimCount)
			}
		})
	}

	rollbackClaim := claim
	rollbackClaim.Key = "task304-audit-rollback-" + uuid.NewString()
	rollbackClaim.Item.Name = "Task 304 audit rollback"
	rollbackRequestID := uuid.NewString()
	err = audit.WithMutationAudit(ctx, AdminAuditEntry{ActorKind: AdminAuditActorAdministrator, AdminUserID: &adminID, Action: "manual_create", EntityType: "food_item", RequestID: rollbackRequestID}, func(tx AdminMutationExecutor) (AdminAuditChanges, error) {
		_, mutationErr := manual.ClaimCreate(ctx, tx, rollbackClaim, encode)
		if mutationErr != nil {
			return AdminAuditChanges{}, mutationErr
		}
		return AdminAuditChanges{After: []byte(`{"forbidden":"field"}`)}, nil
	})
	if err == nil {
		t.Fatalf("audit rollback error = %v", err)
	}
	var rollbackRows, rollbackAudits, rollbackClaims int
	if err := db.QueryRow(ctx, task304CountFoodItemsSQL, rollbackClaim.Item.Name).Scan(&rollbackRows); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(ctx, task304CountAuditEntriesSQL, rollbackRequestID).Scan(&rollbackAudits); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(ctx, task304CountIdempotencyKeysSQL, rollbackClaim.Key).Scan(&rollbackClaims); err != nil {
		t.Fatal(err)
	}
	if rollbackRows != 0 || rollbackAudits != 0 || rollbackClaims != 0 {
		t.Fatalf("audit rollback mutated row=%d audit=%d claim=%d", rollbackRows, rollbackAudits, rollbackClaims)
	}
}
