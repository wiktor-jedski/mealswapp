package repository

// Implements DESIGN-009 TagManager PostgreSQL and atomic-audit verification.

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestClassificationAdminRepositoryCRUDHierarchyConflictsAndSearchRename(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	repo := NewPostgresClassificationRepository(db)

	fruit, err := repo.Create(ctx, ClassificationEntity{Name: "Fruit", Kind: ClassificationKindFoodCategory})
	if err != nil {
		t.Fatalf("Create() root error = %v", err)
	}
	citrus, err := repo.Create(ctx, ClassificationEntity{Name: "Citrus", Kind: ClassificationKindFoodCategory, ParentID: &fruit.ID})
	if err != nil {
		t.Fatalf("Create() child error = %v", err)
	}
	vegetable, err := repo.Create(ctx, ClassificationEntity{Name: "Vegetable", Kind: ClassificationKindFoodCategory})
	if err != nil {
		t.Fatalf("Create() second root error = %v", err)
	}
	role, err := repo.Create(ctx, ClassificationEntity{Name: "Snack", Kind: ClassificationKindCulinaryRole})
	if err != nil {
		t.Fatalf("Create() role error = %v", err)
	}
	if _, err := repo.Create(ctx, ClassificationEntity{Name: " fruit ", Kind: ClassificationKindFoodCategory}); !IsKind(err, ErrorKindConflict) {
		t.Fatalf("duplicate Create() error = %v", err)
	}
	if _, err := repo.Create(ctx, ClassificationEntity{Name: "Wrong parent", Kind: ClassificationKindFoodCategory, ParentID: &role.ID}); !IsKind(err, ErrorKindValidation) {
		t.Fatalf("cross-kind Create() error = %v", err)
	}

	listed, err := repo.List(ctx, ClassificationKindFoodCategory)
	if err != nil || len(listed) != 3 || listed[0].ID != fruit.ID || listed[1].ID != citrus.ID || listed[2].ID != vegetable.ID {
		t.Fatalf("List() = %#v, %v", listed, err)
	}
	loaded, err := repo.GetByID(ctx, citrus.ID)
	if err != nil || loaded.Name != "Citrus" {
		t.Fatalf("GetByID() = %#v, %v", loaded, err)
	}
	renamed, err := repo.Update(ctx, ClassificationEntity{ID: citrus.ID, Name: "Sweet citrus", Kind: citrus.Kind, ParentID: citrus.ParentID})
	if err != nil || renamed.Name != "Sweet citrus" {
		t.Fatalf("Update() = %#v, %v", renamed, err)
	}

	foodRepo := NewPostgresFoodItemRepository(db)
	_, err = foodRepo.Create(ctx, FoodItemEntity{Name: "Orange", PhysicalState: PhysicalStateSolid, MacrosPer100: MacroValues{Protein: 1, Carbohydrates: 12}, FoodCategories: []ClassificationEntity{{ID: citrus.ID}}})
	if err != nil {
		t.Fatalf("create food fixture: %v", err)
	}
	items, total, err := foodRepo.Search(ctx, RepositoryQuery{Name: "Orange", Limit: 10})
	if err != nil || total != 1 || len(items) != 1 || len(items[0].FoodCategories) != 1 || items[0].FoodCategories[0].Name != "Sweet citrus" {
		t.Fatalf("renamed search projection total=%d items=%#v err=%v", total, items, err)
	}
	if err := repo.SoftDelete(ctx, citrus.ID); !IsKind(err, ErrorKindConflict) {
		t.Fatalf("in-use SoftDelete() error = %v", err)
	}
	if err := repo.SoftDelete(ctx, fruit.ID); !IsKind(err, ErrorKindConflict) {
		t.Fatalf("parent SoftDelete() error = %v", err)
	}
	if err := repo.SoftDelete(ctx, vegetable.ID); err != nil {
		t.Fatalf("unused SoftDelete() error = %v", err)
	}
	if _, err := repo.GetByID(ctx, vegetable.ID); !IsKind(err, ErrorKindNotFound) {
		t.Fatalf("deleted GetByID() error = %v", err)
	}
}

func TestClassificationAdminMutationRollsBackWhenAuditFails(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	audit := NewPostgresAdminImportAuditRepository(db)
	name := "Rollback category " + uuid.NewString()
	err := audit.WithMutationAudit(ctx, AdminAuditEntry{AdminUserID: uuid.New(), Action: "classification.create", EntityType: "classification", RequestID: uuid.NewString()}, func(tx AdminMutationExecutor) (AdminAuditChanges, error) {
		created, err := NewPostgresClassificationRepository(tx).Create(ctx, ClassificationEntity{Name: name, Kind: ClassificationKindFoodCategory})
		if err != nil {
			return AdminAuditChanges{}, err
		}
		return AdminAuditChanges{EntityID: &created.ID, After: []byte(`{"name":"must-not-persist"}`)}, nil
	})
	if err == nil {
		t.Fatal("WithMutationAudit() error = nil")
	}
	items, listErr := NewPostgresClassificationRepository(db).List(ctx, ClassificationKindFoodCategory)
	if listErr != nil {
		t.Fatalf("List() error = %v", listErr)
	}
	for _, item := range items {
		if strings.EqualFold(item.Name, name) {
			t.Fatalf("audit failure committed classification %#v", item)
		}
	}
}
