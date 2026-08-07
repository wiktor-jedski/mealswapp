package app

// Implements DESIGN-008 AccountDeleter permanent custom-item deletion integration verification.

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// TestTask298PermanentCustomFoodDeletionPostgresAndExport proves the complete
// owner-scoped conflict, hard-delete, marker, reuse, and export behavior.
func TestTask298PermanentCustomFoodDeletionPostgresAndExport(t *testing.T) {
	db := openDailyDietAPIIntegrationDB(t)
	ctx := context.Background()
	cfg := liveDailyDietAPIConfig()
	server, err := NewProduction(cfg, db, nil, observability.JSONSink{Writer: io.Discard})
	if err != nil {
		t.Fatalf("NewProduction() error = %v", err)
	}

	ownerCookies, ownerID := registerLiveDailyDietUser(t, server, cfg, "task-298-owner-"+uuid.NewString()+"@example.test")
	otherCookies, _ := registerLiveDailyDietUser(t, server, cfg, "task-298-other-"+uuid.NewString()+"@example.test")
	ownerCSRF, ownerCookies := fetchLiveDailyDietCSRF(t, server, ownerCookies)
	otherCSRF, otherCookies := fetchLiveDailyDietCSRF(t, server, otherCookies)
	const createKey = "task-298-create-key"
	const itemName = "Task 298 private lentils"
	itemID, itemBody := task298CreateCustomItem(t, server, ownerCookies, ownerCSRF, createKey, itemName)

	classificationID, err := repository.NewPostgresClassificationRepository(db).Upsert(ctx, repository.ClassificationEntity{
		Name: "Task 298 private category", Kind: repository.ClassificationKindFoodCategory,
	})
	if err != nil {
		t.Fatalf("create deletion classification: %v", err)
	}
	if _, err := db.Exec(ctx, `INSERT INTO custom_food_item_classifications (custom_food_item_id, classification_id) VALUES ($1, $2)`, itemID, classificationID); err != nil {
		t.Fatalf("attach deletion classification: %v", err)
	}

	dietID := uuid.New()
	if _, err := db.Exec(ctx, `INSERT INTO saved_diets (id, user_id, name) VALUES ($1, $2, 'Task 298 owner diet')`, dietID, ownerID); err != nil {
		t.Fatalf("seed saved diet: %v", err)
	}
	if _, err := db.Exec(ctx, `INSERT INTO saved_diet_meal_entries (saved_diet_id, custom_food_item_id, quantity, unit, position) VALUES ($1, $2, 100, 'g', 0)`, dietID, itemID); err != nil {
		t.Fatalf("seed saved diet reference: %v", err)
	}

	beforeJSON := task298Export(t, server, ownerCookies, "json")
	beforeCSV := task298Export(t, server, ownerCookies, "csv")
	for _, payload := range [][]byte{beforeJSON, beforeCSV} {
		if !strings.Contains(string(payload), itemID.String()) || !strings.Contains(string(payload), itemName) {
			t.Fatalf("pre-delete export omitted custom item: %s", payload)
		}
	}

	conflict := liveDailyDietRequest(t, server, fiber.MethodDelete, "/api/v1/custom-items/"+itemID.String(), "", ownerCookies, "", ownerCSRF)
	conflictEnvelope := decodeLiveDailyDietEnvelope(t, conflict)
	conflict.Body.Close()
	if conflict.StatusCode != fiber.StatusConflict || conflictEnvelope.Error == nil || conflictEnvelope.Error.Code != "custom_item_in_use" {
		t.Fatalf("referenced deletion status=%d envelope=%+v", conflict.StatusCode, conflictEnvelope)
	}
	affected, ok := conflictEnvelope.Error.Data["affectedDiets"].([]any)
	if !ok || len(affected) != 1 || affected[0].(map[string]any)["id"] != dietID.String() || affected[0].(map[string]any)["name"] != "Task 298 owner diet" {
		t.Fatalf("bounded affected diets=%#v", conflictEnvelope.Error.Data["affectedDiets"])
	}
	var retainedName string
	if err := db.QueryRow(ctx, `SELECT name FROM custom_food_items WHERE id = $1`, itemID).Scan(&retainedName); err != nil || retainedName != itemName {
		t.Fatalf("conflict mutated item name=%q err=%v", retainedName, err)
	}
	var retainedReferences int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM saved_diet_meal_entries WHERE custom_food_item_id = $1`, itemID).Scan(&retainedReferences); err != nil || retainedReferences != 1 {
		t.Fatalf("conflict mutated references=%d err=%v", retainedReferences, err)
	}

	crossOwner := liveDailyDietRequest(t, server, fiber.MethodDelete, "/api/v1/custom-items/"+itemID.String(), "", otherCookies, "", otherCSRF)
	assertLiveDailyDietStatus(t, crossOwner, fiber.StatusNotFound)

	removeDiet := liveDailyDietRequest(t, server, fiber.MethodDelete, "/api/v1/daily-diets/"+dietID.String(), "", ownerCookies, "", ownerCSRF)
	assertLiveDailyDietStatus(t, removeDiet, fiber.StatusNoContent)
	deleteResponse := liveDailyDietRequest(t, server, fiber.MethodDelete, "/api/v1/custom-items/"+itemID.String(), "", ownerCookies, "", ownerCSRF)
	assertLiveDailyDietStatus(t, deleteResponse, fiber.StatusNoContent)
	repeatedDelete := liveDailyDietRequest(t, server, fiber.MethodDelete, "/api/v1/custom-items/"+itemID.String(), "", ownerCookies, "", ownerCSRF)
	assertLiveDailyDietStatus(t, repeatedDelete, fiber.StatusNotFound)

	var itemCount, classificationCount, referenceCount int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM custom_food_items WHERE id = $1`, itemID).Scan(&itemCount); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(ctx, `SELECT count(*) FROM custom_food_item_classifications WHERE custom_food_item_id = $1`, itemID).Scan(&classificationCount); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(ctx, `SELECT count(*) FROM saved_diet_meal_entries WHERE custom_food_item_id = $1`, itemID).Scan(&referenceCount); err != nil {
		t.Fatal(err)
	}
	if itemCount != 0 || classificationCount != 0 || referenceCount != 0 {
		t.Fatalf("hard deletion retained item=%d classifications=%d references=%d", itemCount, classificationCount, referenceCount)
	}

	var markerJSON string
	var markerExpiry time.Time
	if err := db.QueryRow(ctx, `
		SELECT to_jsonb(marker)::text, expires_at
		FROM (SELECT user_id, key, expires_at FROM deleted_custom_food_create_keys WHERE user_id = $1 AND key = $2) AS marker
	`, ownerID, createKey).Scan(&markerJSON, &markerExpiry); err != nil {
		t.Fatalf("load delete retry marker: %v", err)
	}
	if !markerExpiry.After(time.Now()) || strings.Contains(markerJSON, itemID.String()) || strings.Contains(markerJSON, itemName) {
		t.Fatalf("marker=%s expires_at=%s contains deleted payload or is expired", markerJSON, markerExpiry)
	}
	var markerFields map[string]any
	if err := json.Unmarshal([]byte(markerJSON), &markerFields); err != nil {
		t.Fatalf("decode delete retry marker: %v", err)
	}
	if len(markerFields) != 3 {
		t.Fatalf("marker fields=%v, want only owner/key/expiry", markerFields)
	}

	replay := liveDailyDietRequest(t, server, fiber.MethodPost, "/api/v1/custom-items", itemBody, ownerCookies, createKey, ownerCSRF)
	replayEnvelope := decodeLiveDailyDietEnvelope(t, replay)
	replay.Body.Close()
	if replay.StatusCode != fiber.StatusConflict || replayEnvelope.Error == nil {
		t.Fatalf("delayed create replay status=%d envelope=%+v", replay.StatusCode, replayEnvelope)
	}

	if _, err := db.Exec(ctx, `UPDATE deleted_custom_food_create_keys SET expires_at = now() - interval '1 second' WHERE user_id = $1 AND key = $2`, ownerID, createKey); err != nil {
		t.Fatal(err)
	}
	if err := repository.NewPostgresCustomFoodItemRepository(db).PurgeExpiredDeletedCustomFoodCreateKeys(ctx); err != nil {
		t.Fatalf("purge expired markers: %v", err)
	}
	var expiredMarkers int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM deleted_custom_food_create_keys WHERE user_id = $1 AND key = $2`, ownerID, createKey).Scan(&expiredMarkers); err != nil {
		t.Fatal(err)
	}
	if expiredMarkers != 0 {
		t.Fatalf("expired marker count=%d, want 0", expiredMarkers)
	}

	reusedID, _ := task298CreateCustomItem(t, server, ownerCookies, ownerCSRF, "task-298-reuse-key", itemName)
	if reusedID == itemID {
		t.Fatal("same-name create reused deleted item identity")
	}
	reusedDelete := liveDailyDietRequest(t, server, fiber.MethodDelete, "/api/v1/custom-items/"+reusedID.String(), "", ownerCookies, "", ownerCSRF)
	assertLiveDailyDietStatus(t, reusedDelete, fiber.StatusNoContent)

	afterJSON := task298Export(t, server, ownerCookies, "json")
	afterCSV := task298Export(t, server, ownerCookies, "csv")
	for _, payload := range [][]byte{afterJSON, afterCSV} {
		if strings.Contains(string(payload), itemID.String()) || strings.Contains(string(payload), itemName) {
			t.Fatalf("deleted custom item was recovered by export: %s", payload)
		}
	}
	readDeleted := liveDailyDietRequest(t, server, fiber.MethodGet, "/api/v1/custom-items/"+itemID.String(), "", ownerCookies, "", "")
	assertLiveDailyDietStatus(t, readDeleted, fiber.StatusNotFound)
	listAfterDelete := liveDailyDietRequest(t, server, fiber.MethodGet, "/api/v1/custom-items", "", ownerCookies, "", "")
	listEnvelope := decodeLiveDailyDietEnvelope(t, listAfterDelete)
	listAfterDelete.Body.Close()
	if listAfterDelete.StatusCode != fiber.StatusOK || len(listEnvelope.Data["items"].([]any)) != 0 {
		t.Fatalf("deleted item remained in list: status=%d body=%+v", listAfterDelete.StatusCode, listEnvelope)
	}

	if _, err := db.Exec(ctx, `DELETE FROM users WHERE id = $1`, ownerID); err != nil {
		t.Fatalf("erase owner for marker cascade: %v", err)
	}
	var remainingMarkers int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM deleted_custom_food_create_keys WHERE user_id = $1`, ownerID).Scan(&remainingMarkers); err != nil {
		t.Fatal(err)
	}
	if remainingMarkers != 0 {
		t.Fatalf("account erasure retained %d retry markers", remainingMarkers)
	}
}

func task298CreateCustomItem(t *testing.T, server *fiber.App, cookies []*http.Cookie, csrf, key, name string) (uuid.UUID, string) {
	t.Helper()
	body := task240CustomItemBody(name)
	response := liveDailyDietRequest(t, server, fiber.MethodPost, "/api/v1/custom-items", body, cookies, key, csrf)
	envelope := decodeLiveDailyDietEnvelope(t, response)
	response.Body.Close()
	if response.StatusCode != fiber.StatusCreated {
		t.Fatalf("create custom item %q status=%d body=%+v", name, response.StatusCode, envelope)
	}
	return liveUUIDFromData(t, envelope.Data, "id"), body
}

func task298Export(t *testing.T, server *fiber.App, cookies []*http.Cookie, format string) []byte {
	t.Helper()
	response := liveDailyDietRequest(t, server, fiber.MethodGet, "/api/v1/account/export?format="+format, "", cookies, "", "")
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil || response.StatusCode != http.StatusOK {
		t.Fatalf("export %s status=%d err=%v body=%s", format, response.StatusCode, err, body)
	}
	return body
}
