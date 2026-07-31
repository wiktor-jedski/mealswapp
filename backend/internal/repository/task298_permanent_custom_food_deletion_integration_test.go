package repository

// Implements DESIGN-008 AccountDeleter migration and delayed-replay verification.

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestTask298MigrationMaterializesLegacyRetryMarker proves migration 31 keeps a
// pre-existing completed create key from replaying after a legacy purge.
func TestTask298MigrationMaterializesLegacyRetryMarker(t *testing.T) {
	db := openRepositoryTestDB(t)
	ctx := context.Background()
	ownerID := createRepositoryUser(t, ctx, db, "task-298-legacy@example.test")
	itemID, err := NewPostgresCustomFoodItemRepository(db).Create(ctx, CustomFoodItemEntity{
		OwnerID: ownerID,
		FoodItemEntity: FoodItemEntity{
			Name:          "Legacy private item",
			PhysicalState: PhysicalStateSolid,
			MacrosPer100:  MacroValues{Protein: 10, Carbohydrates: 20, Fat: 5},
			Micros:        MicroValues{},
		},
	})
	if err != nil {
		t.Fatalf("create legacy item: %v", err)
	}
	if _, err := db.Exec(ctx, `UPDATE custom_food_items SET deleted_at = now() WHERE id = $1`, itemID); err != nil {
		t.Fatalf("soft-delete legacy item: %v", err)
	}

	const key = "task-298-legacy-key"
	const bodyHash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	responseBody, err := json.Marshal(map[string]string{"id": itemID.String(), "name": "Legacy private item"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO mutation_idempotency_keys (user_id, method, route, key, body_hash, status_code, response_body)
		VALUES ($1, 'POST', '/custom-items', $2, $3, 201, $4::jsonb)
	`, ownerID, key, bodyHash, responseBody); err != nil {
		t.Fatalf("seed legacy create claim: %v", err)
	}

	task298ExecMigration(t, db, "000031_permanent_custom_food_deletion.down.sql")
	task298ExecMigration(t, db, "000031_permanent_custom_food_deletion.up.sql")

	var markerJSON string
	var expiresAt time.Time
	if err := db.QueryRow(ctx, `
		SELECT to_jsonb(marker)::text, expires_at
		FROM (SELECT user_id, key, expires_at FROM deleted_custom_food_create_keys WHERE user_id = $1 AND key = $2) AS marker
	`, ownerID, key).Scan(&markerJSON, &expiresAt); err != nil {
		t.Fatalf("load materialized retry marker: %v", err)
	}
	if !expiresAt.After(time.Now()) || strings.Contains(markerJSON, itemID.String()) || strings.Contains(markerJSON, "Legacy private item") {
		t.Fatalf("legacy marker=%s expires_at=%s contains deleted payload or is expired", markerJSON, expiresAt)
	}
	var marker map[string]any
	if err := json.Unmarshal([]byte(markerJSON), &marker); err != nil {
		t.Fatalf("decode marker: %v", err)
	}
	if len(marker) != 3 || marker["user_id"] != ownerID.String() || marker["key"] != key {
		t.Fatalf("marker has unexpected payload: %s", markerJSON)
	}

	var itemCount, claimCount int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM custom_food_items WHERE id = $1`, itemID).Scan(&itemCount); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(ctx, `SELECT count(*) FROM mutation_idempotency_keys WHERE user_id = $1 AND key = $2`, ownerID, key).Scan(&claimCount); err != nil {
		t.Fatal(err)
	}
	if itemCount != 0 || claimCount != 0 {
		t.Fatalf("legacy purge retained item=%d or claim=%d", itemCount, claimCount)
	}

	_, err = NewPostgresCustomFoodItemRepository(db).ClaimCreate(ctx, CustomFoodItemCreateClaim{
		UserID:   ownerID,
		Key:      key,
		BodyHash: bodyHash,
		Item: CustomFoodItemEntity{OwnerID: ownerID, FoodItemEntity: FoodItemEntity{
			Name: "Legacy private item", PhysicalState: PhysicalStateSolid, Micros: MicroValues{},
		}},
	}, func(item CustomFoodItemEntity) ([]byte, error) { return json.Marshal(item) })
	if !IsKind(err, ErrorKindConflict) {
		t.Fatalf("delayed legacy create replay error=%v, want conflict", err)
	}
}

func task298ExecMigration(t *testing.T, db *pgxpool.Pool, name string) {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("../../../database/migrations", name))
	if err != nil {
		t.Fatalf("resolve migration %s: %v", name, err)
	}
	sql, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration %s: %v", name, err)
	}
	if _, err := db.Exec(context.Background(), string(sql)); err != nil {
		t.Fatalf("execute migration %s: %v", name, err)
	}
}
