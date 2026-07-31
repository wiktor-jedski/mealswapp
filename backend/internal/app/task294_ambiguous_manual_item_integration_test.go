package app

// Implements DESIGN-009 ItemCurator ambiguous manual-item outcome recovery verification.

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wiktor-jedski/mealswapp/backend/internal/cache"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
)

// TestTask294CorruptedCreateResponseRecoversWithoutDuplicateEffects verifies
// SW-REQ-056 against production HTTP composition, PostgreSQL, and Redis.
func TestTask294CorruptedCreateResponseRecoversWithoutDuplicateEffects(t *testing.T) {
	db := openDailyDietAPIIntegrationDB(t)
	redisClient := openTask206Redis(t)
	cfg := liveDailyDietAPIConfig()
	server, err := NewProduction(cfg, db, redisClient, observability.JSONSink{Writer: io.Discard})
	if err != nil {
		t.Fatalf("compose production app: %v", err)
	}
	adminCookies, adminID := task271RegisterAdmin(t, server, db, cfg)
	csrf, adminCookies := fetchLiveDailyDietCSRF(t, server, adminCookies)
	generation := cache.NewClassificationGeneration(redisClient)
	before := task271Generation(t, generation)
	name := "task294-" + strings.ReplaceAll(uuid.NewString(), "-", "")[:16]
	body := task271ItemBody(name)
	key := "task-294-" + uuid.NewString()

	committed := liveDailyDietRequest(t, server, fiber.MethodPost, "/api/v1/admin/items", body, adminCookies, key, csrf)
	if committed.StatusCode != fiber.StatusCreated {
		t.Fatalf("manual create status=%d", committed.StatusCode)
	}
	committed.Body.Close()
	committed.Body = io.NopCloser(strings.NewReader("{"))
	var undecodable map[string]any
	if json.NewDecoder(committed.Body).Decode(&undecodable) == nil {
		t.Fatal("response-corruption fixture remained decodable")
	}
	committed.Body.Close()

	itemID := task294ItemIDByName(t, db, name)
	task294AssertEffects(t, db, generation, adminID, itemID, key, name, before+1)

	recovered := liveDailyDietRequest(t, server, fiber.MethodGet, "/api/v1/admin/items/"+itemID.String(), "", adminCookies, "", "")
	recoveredEnvelope := decodeLiveDailyDietEnvelope(t, recovered)
	recovered.Body.Close()
	if recovered.StatusCode != fiber.StatusOK || recoveredEnvelope.Data["name"] != name {
		t.Fatalf("authoritative recovery status=%d body=%+v", recovered.StatusCode, recoveredEnvelope)
	}
	picker := liveDailyDietRequest(t, server, fiber.MethodGet, "/api/v1/admin/items?query="+name+"&page=1&pageSize=20", "", adminCookies, "", "")
	pickerEnvelope := decodeLiveDailyDietEnvelope(t, picker)
	picker.Body.Close()
	if picker.StatusCode != fiber.StatusOK || pickerEnvelope.Data["total"] != float64(1) {
		t.Fatalf("picker recovery status=%d body=%+v", picker.StatusCode, pickerEnvelope)
	}

	for range 2 {
		replay := liveDailyDietRequest(t, server, fiber.MethodPost, "/api/v1/admin/items", body, adminCookies, key, csrf)
		replayEnvelope := decodeLiveDailyDietEnvelope(t, replay)
		replay.Body.Close()
		if replay.StatusCode != fiber.StatusCreated || liveUUIDFromData(t, replayEnvelope.Data, "id") != itemID {
			t.Fatalf("same-key replay status=%d body=%+v", replay.StatusCode, replayEnvelope)
		}
		task294AssertEffects(t, db, generation, adminID, itemID, key, name, before+1)
	}

	conflict := liveDailyDietRequest(t, server, fiber.MethodPost, "/api/v1/admin/items", task271ItemBody(name+" changed"), adminCookies, key, csrf)
	task271AssertError(t, conflict, fiber.StatusConflict, "idempotency_key_conflict")
	missing := liveDailyDietRequest(t, server, fiber.MethodGet, "/api/v1/admin/items/"+uuid.NewString(), "", adminCookies, "", "")
	task271AssertError(t, missing, fiber.StatusNotFound, "not_found")
	task294AssertEffects(t, db, generation, adminID, itemID, key, name, before+1)
}

func task294ItemIDByName(t *testing.T, db *pgxpool.Pool, name string) uuid.UUID {
	t.Helper()
	var itemID uuid.UUID
	if err := db.QueryRow(context.Background(), `SELECT id FROM food_items WHERE name=$1`, name).Scan(&itemID); err != nil {
		t.Fatalf("find committed manual item: %v", err)
	}
	return itemID
}

func task294AssertEffects(t *testing.T, db *pgxpool.Pool, generation cache.ClassificationGeneration, adminID, itemID uuid.UUID, key, name string, wantGeneration uint64) {
	t.Helper()
	var foods, audits, claims int
	if err := db.QueryRow(context.Background(), `SELECT count(*) FROM food_items WHERE name=$1`, name).Scan(&foods); err != nil {
		t.Fatalf("count committed foods: %v", err)
	}
	if err := db.QueryRow(context.Background(), `SELECT count(*) FROM admin_audit_entries WHERE entity_type='food_item' AND entity_id=$1 AND action='manual_create'`, itemID).Scan(&audits); err != nil {
		t.Fatalf("count committed audits: %v", err)
	}
	if err := db.QueryRow(context.Background(), `SELECT count(*) FROM mutation_idempotency_keys WHERE user_id=$1 AND method='POST' AND route='/admin/items' AND key=$2 AND status_code=201`, adminID, key).Scan(&claims); err != nil {
		t.Fatalf("count committed idempotency records: %v", err)
	}
	if foods != 1 || audits != 1 || claims != 1 {
		t.Fatalf("effects foods=%d audits=%d claims=%d, want exactly one each", foods, audits, claims)
	}
	task271AssertGeneration(t, generation, wantGeneration)
}
