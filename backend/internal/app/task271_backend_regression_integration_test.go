package app

// Implements DESIGN-009 AdminController/ItemCurator, DESIGN-010 RequestValidator,
// and DESIGN-011 RedisCache production integration verification.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wiktor-jedski/mealswapp/backend/internal/cache"
	"github.com/wiktor-jedski/mealswapp/backend/internal/config"
	"github.com/wiktor-jedski/mealswapp/backend/internal/httpapi"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// TestTask271ProductionBackendRegressionGate verifies IT-ARCH-009-008 and
// IT-ARCH-009-009, ARCH-009, DESIGN-009 ItemCurator, DESIGN-010
// RequestValidator, DESIGN-011 CacheInvalidator, and
// SW-REQ-032/SW-REQ-056/SW-REQ-090 through production composition,
// authenticated HTTP, PostgreSQL, Redis, audited transactions, Catalog Search,
// and Substitution Search.
func TestTask271ProductionBackendRegressionGate(t *testing.T) {
	db := openDailyDietAPIIntegrationDB(t)
	redisClient := openTask206Redis(t)
	cfg := liveDailyDietAPIConfig()
	var telemetry task271SafeBuffer
	serverOne, err := NewProduction(cfg, db, redisClient, observability.JSONSink{Writer: &telemetry})
	if err != nil {
		t.Fatalf("compose first production app: %v", err)
	}
	serverTwo, err := NewProduction(cfg, db, redisClient, observability.JSONSink{Writer: io.Discard})
	if err != nil {
		t.Fatalf("compose second production app: %v", err)
	}

	adminCookies, adminID := task271RegisterAdmin(t, serverOne, db, cfg)
	adminCSRF, adminCookies := fetchLiveDailyDietCSRF(t, serverOne, adminCookies)
	grantTask206Trial(t, db, adminID)
	userCookies, userID := registerLiveDailyDietUser(t, serverOne, cfg, "task-271-user-"+uuid.NewString()+"@example.test")
	userCSRF, userCookies := fetchLiveDailyDietCSRF(t, serverOne, userCookies)
	otherCookies, _ := registerLiveDailyDietUser(t, serverOne, cfg, "task-271-other-"+uuid.NewString()+"@example.test")

	nonAdmin := liveDailyDietRequest(t, serverOne, fiber.MethodPost, "/api/v1/admin/items", task271ItemBody("Forbidden global item"), userCookies, "task-271-forbidden", userCSRF)
	task271AssertError(t, nonAdmin, fiber.StatusForbidden, "forbidden")
	task271AssertCount(t, db, "SELECT count(*) FROM food_items", 0)

	sourceID, err := repository.NewPostgresFoodItemRepository(db).Create(context.Background(), repository.FoodItemEntity{
		Name: "Task 271 substitution source", PhysicalState: repository.PhysicalStateSolid,
		MacrosPer100: repository.MacroValues{Protein: 10, Carbohydrates: 5, Fat: 2},
	})
	if err != nil {
		t.Fatalf("create substitution source: %v", err)
	}
	query := "gate" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	catalogBody := task271CatalogBody(query)
	substitutionBody := task271SubstitutionBody(sourceID)
	task271AssertSearchNames(t, serverTwo, catalogBody, adminCookies)
	task271AssertCacheHit(t, task271AssertSearchNames(t, serverTwo, catalogBody, adminCookies))
	task271AssertSearchNames(t, serverTwo, substitutionBody, adminCookies)
	task271AssertCacheHit(t, task271AssertSearchNames(t, serverTwo, substitutionBody, adminCookies))

	generation := cache.NewClassificationGeneration(redisClient)
	beforeCreate := task271Generation(t, generation)
	itemName := query + " tofu"
	create := liveDailyDietRequest(t, serverOne, fiber.MethodPost, "/api/v1/admin/items", task271ItemBody(itemName), adminCookies, "task-271-create", adminCSRF)
	createEnvelope := decodeLiveDailyDietEnvelope(t, create)
	create.Body.Close()
	if create.StatusCode != fiber.StatusCreated {
		t.Fatalf("manual create status=%d body=%+v", create.StatusCode, createEnvelope)
	}
	itemID := liveUUIDFromData(t, createEnvelope.Data, "id")
	task271AssertGeneration(t, generation, beforeCreate+1)
	task271AssertAuditCount(t, db, itemID, "manual_create", 1)
	task271AssertSearchNames(t, serverTwo, catalogBody, adminCookies, itemName)
	task271AssertSearchNames(t, serverTwo, substitutionBody, adminCookies, itemName)

	beforeReplay := task271Generation(t, generation)
	replay := liveDailyDietRequest(t, serverOne, fiber.MethodPost, "/api/v1/admin/items", task271ItemBody(itemName), adminCookies, "task-271-create", adminCSRF)
	replayEnvelope := decodeLiveDailyDietEnvelope(t, replay)
	replay.Body.Close()
	if replay.StatusCode != fiber.StatusCreated || liveUUIDFromData(t, replayEnvelope.Data, "id") != itemID {
		t.Fatalf("manual replay status=%d body=%+v", replay.StatusCode, replayEnvelope)
	}
	task271AssertGeneration(t, generation, beforeReplay)
	task271AssertAuditCount(t, db, itemID, "manual_create", 1)

	rollbackQuery := "rollback" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	task271AssertSearchNames(t, serverTwo, task271CatalogBody(rollbackQuery), adminCookies)
	task271InstallAuditFailure(t, db)
	beforeRollback := task271Generation(t, generation)
	rollback := liveDailyDietRequest(t, serverOne, fiber.MethodPost, "/api/v1/admin/items", task271ItemBody(rollbackQuery+" secret-trigger-item"), adminCookies, "task-271-rollback", adminCSRF)
	task271DropAuditFailure(t, db)
	rollbackEnvelope := task271AssertError(t, rollback, fiber.StatusServiceUnavailable, "dependency_unavailable")
	task271AssertGeneration(t, generation, beforeRollback)
	task271AssertCount(t, db, "SELECT count(*) FROM food_items WHERE name LIKE 'rollback%'", 0)
	task271AssertCount(t, db, "SELECT count(*) FROM admin_audit_entries WHERE action='manual_create'", 1)
	encodedRollback, err := json.Marshal(rollbackEnvelope)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"task 271 forced audit failure", "secret-trigger-item"} {
		if strings.Contains(string(encodedRollback), forbidden) || strings.Contains(telemetry.String(), forbidden) {
			t.Fatalf("unsafe rollback detail %q reached envelope or telemetry", forbidden)
		}
	}

	updatedName := query + " tempeh"
	beforeUpdate := task271Generation(t, generation)
	update := liveDailyDietRequest(t, serverOne, fiber.MethodPut, "/api/v1/admin/items/"+itemID.String(), task271ItemBody(updatedName), adminCookies, "", adminCSRF)
	updateEnvelope := decodeLiveDailyDietEnvelope(t, update)
	update.Body.Close()
	if update.StatusCode != fiber.StatusOK || updateEnvelope.Data["name"] != updatedName {
		t.Fatalf("manual update status=%d body=%+v", update.StatusCode, updateEnvelope)
	}
	task271AssertGeneration(t, generation, beforeUpdate+1)
	task271AssertAuditCount(t, db, itemID, "manual_update", 1)
	task271AssertSearchNames(t, serverTwo, catalogBody, adminCookies, updatedName)
	task271AssertSearchNames(t, serverTwo, substitutionBody, adminCookies, updatedName)

	beforeDelete := task271Generation(t, generation)
	deleteResponse := liveDailyDietRequest(t, serverOne, fiber.MethodDelete, "/api/v1/admin/items/"+itemID.String(), "", adminCookies, "", adminCSRF)
	assertLiveDailyDietStatus(t, deleteResponse, fiber.StatusNoContent)
	task271AssertGeneration(t, generation, beforeDelete+1)
	task271AssertAuditCount(t, db, itemID, "manual_delete", 1)
	task271AssertSearchNames(t, serverTwo, catalogBody, adminCookies)
	task271AssertSearchNames(t, serverTwo, substitutionBody, adminCookies)

	privateID := task271CreateCustomItem(t, serverOne, userCookies, userCSRF, query+" private")
	ownerIsolation := liveDailyDietRequest(t, serverOne, fiber.MethodGet, "/api/v1/custom-items/"+privateID.String(), "", otherCookies, "", "")
	task271AssertError(t, ownerIsolation, fiber.StatusNotFound, "not_found")
	customDispatches := strings.Count(telemetry.String(), observability.MetricCustomItemLifecycleOutcomes)
	task271AssertDuplicateCustomJSON(t, serverOne, db, userID, privateID, userCookies, userCSRF)
	if got := strings.Count(telemetry.String(), observability.MetricCustomItemLifecycleOutcomes); got != customDispatches {
		t.Fatalf("duplicate JSON reached custom-item service: lifecycle metrics before=%d after=%d", customDispatches, got)
	}
}

type task271SafeBuffer struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (b *task271SafeBuffer) Write(payload []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.Write(payload)
}

func (b *task271SafeBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.String()
}

func task271RegisterAdmin(t *testing.T, server *fiber.App, db *pgxpool.Pool, cfg config.Config) ([]*http.Cookie, uuid.UUID) {
	t.Helper()
	email := "task-271-admin-" + uuid.NewString() + "@example.test"
	cookies, adminID := registerLiveDailyDietUser(t, server, cfg, email)
	if _, err := db.Exec(context.Background(), `UPDATE users SET role='admin' WHERE id=$1`, adminID); err != nil {
		t.Fatalf("promote administrator: %v", err)
	}
	login := liveDailyDietRequest(t, server, fiber.MethodPost, "/api/v1/auth/login", fmt.Sprintf(`{"email":%q,"password":"StrongerPassword1!"}`, email), cookies, "", "")
	envelope := decodeLiveDailyDietEnvelope(t, login)
	cookies = mergeLiveDailyDietCookies(cookies, login.Cookies())
	login.Body.Close()
	if login.StatusCode != fiber.StatusOK || envelope.Data["role"] != "admin" {
		t.Fatalf("admin login status=%d body=%+v", login.StatusCode, envelope)
	}
	return cookies, adminID
}

func task271ItemBody(name string) string {
	return fmt.Sprintf(`{"name":%q,"physicalState":"solid","prepTimeMinutes":0,"macrosPer100":{"protein":10,"carbohydrates":5,"fat":2},"micros":{},"foodCategoryIds":[],"culinaryRoleIds":[],"allergenKeys":[]}`, name)
}

func task271CustomItemBody(name string) string {
	return fmt.Sprintf(`{"name":%q,"physicalState":"solid","prepTimeMinutes":0,"macrosPer100":{"protein":10,"carbohydrates":5,"fat":2},"micros":{},"foodCategoryIds":[],"culinaryRoleIds":[]}`, name)
}

func task271CatalogBody(query string) string {
	return fmt.Sprintf(`{"query":%q,"mode":"catalog","page":1}`, query)
}

func task271SubstitutionBody(sourceID uuid.UUID) string {
	return fmt.Sprintf(`{"query":"","mode":"substitution","page":1,"substitutionInputs":[{"foodObjectId":%q,"foodObjectType":"food_item","quantity":100,"unit":"g"}]}`, sourceID)
}

func task271AssertSearchNames(t *testing.T, server *fiber.App, body string, cookies []*http.Cookie, want ...string) map[string]any {
	t.Helper()
	response := liveDailyDietRequest(t, server, fiber.MethodPost, "/api/v1/search", body, cookies, "", "")
	envelope := decodeLiveDailyDietEnvelope(t, response)
	response.Body.Close()
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("search status=%d body=%+v", response.StatusCode, envelope)
	}
	raw, ok := envelope.Data["items"].([]any)
	if !ok {
		t.Fatalf("search items missing: %+v", envelope.Data)
	}
	names := make([]string, 0, len(raw))
	for _, value := range raw {
		item, itemOK := value.(map[string]any)
		name, nameOK := item["name"].(string)
		if itemOK && nameOK {
			names = append(names, name)
		}
	}
	if strings.Join(names, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("search names=%q want=%q", names, want)
	}
	return envelope.Data
}

func task271AssertCacheHit(t *testing.T, data map[string]any) {
	t.Helper()
	metadata, ok := data["cache"].(map[string]any)
	if !ok || metadata["status"] != "hit" || metadata["namespace"] != "search" {
		t.Fatalf("search cache was not prewarmed: %+v", data["cache"])
	}
}

func task271Generation(t *testing.T, generation cache.ClassificationGeneration) uint64 {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	value, err := generation.Current(ctx)
	if err != nil {
		t.Fatalf("read shared generation: %v", err)
	}
	return value
}

func task271AssertGeneration(t *testing.T, generation cache.ClassificationGeneration, want uint64) {
	t.Helper()
	if got := task271Generation(t, generation); got != want {
		t.Fatalf("shared generation=%d want=%d", got, want)
	}
}

func task271AssertAuditCount(t *testing.T, db *pgxpool.Pool, itemID uuid.UUID, action string, want int) {
	t.Helper()
	var count int
	if err := db.QueryRow(context.Background(), `SELECT count(*) FROM admin_audit_entries WHERE entity_type='food_item' AND entity_id=$1 AND action=$2`, itemID, action).Scan(&count); err != nil {
		t.Fatalf("count %s audits: %v", action, err)
	}
	if count != want {
		t.Fatalf("%s audit count=%d want=%d", action, count, want)
	}
}

func task271AssertCount(t *testing.T, db *pgxpool.Pool, query string, want int) {
	t.Helper()
	var count int
	if err := db.QueryRow(context.Background(), query).Scan(&count); err != nil {
		t.Fatalf("count query: %v", err)
	}
	if count != want {
		t.Fatalf("count=%d want=%d for %s", count, want, query)
	}
}

func task271InstallAuditFailure(t *testing.T, db *pgxpool.Pool) {
	t.Helper()
	const statement = `
CREATE FUNCTION task271_reject_manual_audit() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF NEW.action = 'manual_create' THEN
    RAISE EXCEPTION 'task 271 forced audit failure';
  END IF;
  RETURN NEW;
END
$$;
CREATE TRIGGER task271_reject_manual_audit
BEFORE INSERT ON admin_audit_entries
FOR EACH ROW EXECUTE FUNCTION task271_reject_manual_audit()`
	if _, err := db.Exec(context.Background(), statement); err != nil {
		t.Fatalf("install audit failure: %v", err)
	}
	t.Cleanup(func() { task271DropAuditFailure(t, db) })
}

func task271DropAuditFailure(t *testing.T, db *pgxpool.Pool) {
	t.Helper()
	if _, err := db.Exec(context.Background(), `DROP TRIGGER IF EXISTS task271_reject_manual_audit ON admin_audit_entries; DROP FUNCTION IF EXISTS task271_reject_manual_audit()`); err != nil {
		t.Fatalf("drop audit failure: %v", err)
	}
}

func task271AssertError(t *testing.T, response *http.Response, status int, code string) httpapi.Envelope {
	t.Helper()
	envelope := decodeLiveDailyDietEnvelope(t, response)
	response.Body.Close()
	if response.StatusCode != status || envelope.Status != "error" || envelope.Error == nil || envelope.Error.Code != code || envelope.RequestID == "" {
		t.Fatalf("error response=%d %+v want=%d %s", response.StatusCode, envelope, status, code)
	}
	return envelope
}

func task271CreateCustomItem(t *testing.T, server *fiber.App, cookies []*http.Cookie, csrf, name string) uuid.UUID {
	t.Helper()
	response := liveDailyDietRequest(t, server, fiber.MethodPost, "/api/v1/custom-items", task271CustomItemBody(name), cookies, "task-271-private", csrf)
	envelope := decodeLiveDailyDietEnvelope(t, response)
	response.Body.Close()
	if response.StatusCode != fiber.StatusCreated {
		t.Fatalf("create private item status=%d body=%+v", response.StatusCode, envelope)
	}
	return liveUUIDFromData(t, envelope.Data, "id")
}

func task271AssertDuplicateCustomJSON(t *testing.T, server *fiber.App, db *pgxpool.Pool, ownerID, itemID uuid.UUID, cookies []*http.Cookie, csrf string) {
	t.Helper()
	bodies := []string{
		`{"name":"first","name":"second","physicalState":"solid","macrosPer100":{"protein":1,"carbohydrates":2,"fat":3},"micros":{}}`,
		`{"name":"macro","physicalState":"solid","macrosPer100":{"protein":1,"protein":2,"carbohydrates":2,"fat":3},"micros":{}}`,
		`{"name":"micro","physicalState":"solid","macrosPer100":{"protein":1,"carbohydrates":2,"fat":3},"micros":{"Sodium":1,"Sodium":2}}`,
	}
	for _, method := range []string{fiber.MethodPost, fiber.MethodPut} {
		for index, body := range bodies {
			path := "/api/v1/custom-items"
			if method == fiber.MethodPut {
				path += "/" + itemID.String()
			}
			response := liveDailyDietRequest(t, server, method, path, body, cookies, fmt.Sprintf("task-271-duplicate-%s-%d", method, index), csrf)
			envelope := task271AssertError(t, response, fiber.StatusBadRequest, "invalid_json")
			encoded, err := json.Marshal(envelope)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(encoded), "Sodium") || strings.Contains(string(encoded), "second") {
				t.Fatalf("duplicate JSON detail leaked: %s", encoded)
			}
		}
	}
	var count int
	if err := db.QueryRow(context.Background(), `SELECT count(*) FROM custom_food_items WHERE owner_id=$1`, ownerID).Scan(&count); err != nil {
		t.Fatalf("count private items: %v", err)
	}
	if count != 1 {
		t.Fatalf("private item count=%d want=1", count)
	}
	var name string
	if err := db.QueryRow(context.Background(), `SELECT name FROM custom_food_items WHERE id=$1 AND owner_id=$2`, itemID, ownerID).Scan(&name); err != nil {
		t.Fatalf("load private item after duplicate updates: %v", err)
	}
	if !strings.HasSuffix(name, " private") {
		t.Fatalf("private item changed after duplicate updates: %q", name)
	}
}
