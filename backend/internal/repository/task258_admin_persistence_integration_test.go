package repository

// Implements DESIGN-009 TagManager and DESIGN-013 parameterized admin persistence integration gate.

import (
	"context"
	"testing"
)

// TestTask258AdminPersistenceTreatsInputAsData proves that an adversarial
// classification name remains inert data and cannot alter the PostgreSQL schema.
func TestTask258AdminPersistenceTreatsInputAsData(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	repo := NewPostgresClassificationRepository(db)
	name := `Task 258 x'); DROP TABLE users; --`

	created, err := repo.Create(ctx, ClassificationEntity{Name: name, Kind: ClassificationKindFoodCategory})
	if err != nil {
		t.Fatalf("Create() adversarial name error = %v", err)
	}
	loaded, err := repo.GetByID(ctx, created.ID)
	if err != nil || loaded.Name != name {
		t.Fatalf("GetByID() = %+v, %v", loaded, err)
	}
	var usersTable string
	if err := db.QueryRow(ctx, `SELECT to_regclass('public.users')::text`).Scan(&usersTable); err != nil || usersTable != "users" {
		t.Fatalf("users relation after adversarial write = %q, %v", usersTable, err)
	}
}
