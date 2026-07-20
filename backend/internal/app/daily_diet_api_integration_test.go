package app

// Implements DESIGN-008 ProfileController live PostgreSQL API integration verification.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wiktor-jedski/mealswapp/backend/internal/config"
	"github.com/wiktor-jedski/mealswapp/backend/internal/httpapi"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/testdatabase"
)

// TestDailyDietProductionAPIWithLivePostgres verifies the complete authenticated CRUD path against PostgreSQL.
// Implements DESIGN-008 ProfileController, DESIGN-006 JWTManager, and DESIGN-010 CSRFValidator.
func TestDailyDietProductionAPIWithLivePostgres(t *testing.T) {
	db := openDailyDietAPIIntegrationDB(t)
	ctx := context.Background()
	foodRepo := repository.NewPostgresFoodItemRepository(db)
	foodA, err := foodRepo.Create(ctx, repository.FoodItemEntity{
		Name: "Live Diet Food A", PhysicalState: repository.PhysicalStateLiquid,
		DensityGramsPerMilliliter: 1, DensitySourceKind: "manual",
		MacrosPer100: repository.MacroValues{Protein: 3.4, Carbohydrates: 5, Fat: 1},
	})
	if err != nil {
		t.Fatalf("create Food Item A: %v", err)
	}
	mealRepo := repository.NewPostgresMealRepository(db)
	mealA, err := mealRepo.Create(ctx, repository.MealEntity{
		Type: repository.MealTypeSingle, Name: "Live Diet Meal A", PhysicalState: repository.PhysicalStateSolid,
		MacrosPer100: repository.MacroValues{Protein: 10, Carbohydrates: 20, Fat: 5},
	})
	if err != nil {
		t.Fatalf("create meal A: %v", err)
	}
	mealB, err := mealRepo.Create(ctx, repository.MealEntity{
		Type: repository.MealTypeSingle, Name: "Live Diet Meal B", PhysicalState: repository.PhysicalStateSolid,
		MacrosPer100: repository.MacroValues{Protein: 5, Carbohydrates: 5, Fat: 2},
	})
	if err != nil {
		t.Fatalf("create meal B: %v", err)
	}

	cfg := liveDailyDietAPIConfig()
	server, err := NewProduction(cfg, db, nil, observability.JSONSink{Writer: io.Discard})
	if err != nil {
		t.Fatalf("NewProduction() error = %v", err)
	}
	userCookies, userID := registerLiveDailyDietUser(t, server, cfg, "daily-diet-live-a@example.test")
	otherCookies, otherUserID := registerLiveDailyDietUser(t, server, cfg, "daily-diet-live-b@example.test")
	if userID == otherUserID {
		t.Fatal("live registration returned duplicate user IDs")
	}
	csrfToken, userCookies := fetchLiveDailyDietCSRF(t, server, userCookies)
	otherCSRF, otherCookies := fetchLiveDailyDietCSRF(t, server, otherCookies)
	body := liveMixedDailyDietBody("Training Day", foodA, mealB)

	resp := liveDailyDietRequest(t, server, fiber.MethodPost, "/api/v1/daily-diets", body, userCookies, "live-csrf-failure", "")
	assertLiveDailyDietStatus(t, resp, fiber.StatusForbidden)
	if got := countLiveSavedDiets(t, db, userID); got != 0 {
		t.Fatalf("CSRF failure wrote %d diets", got)
	}

	missingMealBody := liveMixedDailyDietBody("Missing Food Object", uuid.New(), mealB)
	resp = liveDailyDietRequest(t, server, fiber.MethodPost, "/api/v1/daily-diets", missingMealBody, userCookies, "live-missing-meal", csrfToken)
	assertLiveDailyDietStatus(t, resp, fiber.StatusNotFound)
	if got := countLiveSavedDiets(t, db, userID); got != 0 {
		t.Fatalf("missing meal wrote %d diets", got)
	}

	resp = liveDailyDietRequest(t, server, fiber.MethodPost, "/api/v1/daily-diets", body, userCookies, "live-create-key", csrfToken)
	created := decodeLiveDailyDietEnvelope(t, resp)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("create status = %d body = %+v", resp.StatusCode, created)
	}
	dietID := liveUUIDFromData(t, created.Data, "id")
	assertLiveAggregate(t, created.Data, 13.4, 15, 5, 158.6)

	resp = liveDailyDietRequest(t, server, fiber.MethodPost, "/api/v1/daily-diets", body, userCookies, "live-create-key", csrfToken)
	replayed := decodeLiveDailyDietEnvelope(t, resp)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusCreated || liveUUIDFromData(t, replayed.Data, "id") != dietID {
		t.Fatalf("replay status=%d body=%+v", resp.StatusCode, replayed)
	}
	if got := countLiveSavedDiets(t, db, userID); got != 1 {
		t.Fatalf("idempotent create count = %d, want 1", got)
	}

	resp = liveDailyDietRequest(t, server, fiber.MethodPost, "/api/v1/daily-diets", liveDailyDietBody("Different Body", mealA, mealB), userCookies, "live-create-key", csrfToken)
	assertLiveDailyDietStatus(t, resp, fiber.StatusConflict)
	if got := countLiveSavedDiets(t, db, userID); got != 1 {
		t.Fatalf("conflicting create count = %d, want 1", got)
	}

	resp = liveDailyDietRequest(t, server, fiber.MethodGet, "/api/v1/daily-diets/"+dietID.String(), "", userCookies, "", "")
	read := decodeLiveDailyDietEnvelope(t, resp)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK || liveUUIDFromData(t, read.Data, "id") != dietID {
		t.Fatalf("read status=%d body=%+v", resp.StatusCode, read)
	}
	assertLiveAggregate(t, read.Data, 13.4, 15, 5, 158.6)

	resp = liveDailyDietRequest(t, server, fiber.MethodGet, "/api/v1/daily-diets", "", userCookies, "", "")
	list := decodeLiveDailyDietEnvelope(t, resp)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK || len(list.Data["diets"].([]any)) != 1 {
		t.Fatalf("list status=%d body=%+v", resp.StatusCode, list)
	}

	resp = liveDailyDietRequest(t, server, fiber.MethodGet, "/api/v1/daily-diets/"+dietID.String(), "", otherCookies, "", "")
	assertLiveDailyDietStatus(t, resp, fiber.StatusNotFound)
	resp = liveDailyDietRequest(t, server, fiber.MethodPut, "/api/v1/daily-diets/"+dietID.String(), liveDailyDietBody("Cross User", mealA, mealB), otherCookies, "live-cross-user", otherCSRF)
	assertLiveDailyDietStatus(t, resp, fiber.StatusNotFound)
	resp = liveDailyDietRequest(t, server, fiber.MethodDelete, "/api/v1/daily-diets/"+dietID.String(), "", otherCookies, "", otherCSRF)
	assertLiveDailyDietStatus(t, resp, fiber.StatusNotFound)
	if got := countLiveSavedDiets(t, db, userID); got != 1 {
		t.Fatalf("cross-user access changed owner count = %d", got)
	}

	replacement := liveDailyDietBody("Rest Day", mealB, mealB)
	resp = liveDailyDietRequest(t, server, fiber.MethodPut, "/api/v1/daily-diets/"+dietID.String(), replacement, userCookies, "", csrfToken)
	updated := decodeLiveDailyDietEnvelope(t, resp)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK || updated.Data["name"] != "Rest Day" {
		t.Fatalf("replace status=%d body=%+v", resp.StatusCode, updated)
	}
	assertLiveAggregate(t, updated.Data, 15, 15, 6, 174)
	resp = liveDailyDietRequest(t, server, fiber.MethodPut, "/api/v1/daily-diets/"+dietID.String(), replacement, userCookies, "", csrfToken)
	assertLiveDailyDietStatus(t, resp, fiber.StatusOK)

	resp = liveDailyDietRequest(t, server, fiber.MethodDelete, "/api/v1/daily-diets/"+dietID.String(), "", userCookies, "", "")
	assertLiveDailyDietStatus(t, resp, fiber.StatusForbidden)
	resp = liveDailyDietRequest(t, server, fiber.MethodDelete, "/api/v1/daily-diets/"+dietID.String(), "", userCookies, "", csrfToken)
	assertLiveDailyDietStatus(t, resp, fiber.StatusNoContent)
	resp = liveDailyDietRequest(t, server, fiber.MethodDelete, "/api/v1/daily-diets/"+dietID.String(), "", userCookies, "", csrfToken)
	assertLiveDailyDietStatus(t, resp, fiber.StatusNoContent)
	if got := countLiveSavedDiets(t, db, userID); got != 0 {
		t.Fatalf("delete count = %d, want 0", got)
	}
	if _, err := db.Exec(ctx, `UPDATE food_items SET protein_per_100 = 999, carbohydrates_per_100 = 999, fat_per_100 = 999 WHERE id = $1`, foodA); err != nil {
		t.Fatalf("change Food Item macros before replay: %v", err)
	}
	resp = liveDailyDietRequest(t, server, fiber.MethodPost, "/api/v1/daily-diets", body, userCookies, "live-create-key", csrfToken)
	afterDeletionReplay := decodeLiveDailyDietEnvelope(t, resp)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusCreated || !reflect.DeepEqual(afterDeletionReplay.Data, created.Data) {
		t.Fatalf("immutable replay status=%d body=%+v want=%+v", resp.StatusCode, afterDeletionReplay, created)
	}
	if got := countLiveSavedDiets(t, db, userID); got != 0 {
		t.Fatalf("replay after deletion recreated %d diets", got)
	}

	serverTwo, err := NewProduction(cfg, db, nil, observability.JSONSink{Writer: io.Discard})
	if err != nil {
		t.Fatalf("second NewProduction() error = %v", err)
	}
	csrfOne, cookiesOne := fetchLiveDailyDietCSRF(t, server, userCookies)
	csrfTwo, cookiesTwo := fetchLiveDailyDietCSRF(t, serverTwo, userCookies)
	concurrentBody := liveDailyDietBody("Concurrent Day", mealA, mealB)
	start := make(chan struct{})
	responses := make(chan *http.Response, 2)
	errors := make(chan error, 2)
	var wait sync.WaitGroup
	wait.Add(2)
	go liveConcurrentDailyDietCreate(&wait, start, responses, errors, server, concurrentBody, cookiesOne, csrfOne)
	go liveConcurrentDailyDietCreate(&wait, start, responses, errors, serverTwo, concurrentBody, cookiesTwo, csrfTwo)
	close(start)
	wait.Wait()
	close(responses)
	close(errors)
	var concurrentID uuid.UUID
	for response := range responses {
		result := decodeLiveDailyDietEnvelope(t, response)
		response.Body.Close()
		if response.StatusCode != fiber.StatusCreated {
			t.Fatalf("concurrent create status=%d body=%+v", response.StatusCode, result)
		}
		id := liveUUIDFromData(t, result.Data, "id")
		if concurrentID == uuid.Nil {
			concurrentID = id
		} else if concurrentID != id {
			t.Fatalf("concurrent create IDs differ: %s and %s", concurrentID, id)
		}
	}
	for err := range errors {
		if err != nil {
			t.Fatal(err)
		}
	}
	if got := countLiveSavedDiets(t, db, userID); got != 1 {
		t.Fatalf("concurrent idempotent create count = %d, want 1", got)
	}
}

func liveConcurrentDailyDietCreate(wait *sync.WaitGroup, start <-chan struct{}, responses chan<- *http.Response, errors chan<- error, server *fiber.App, body string, cookies []*http.Cookie, csrfToken string) {
	defer wait.Done()
	<-start
	request := httptest.NewRequest(fiber.MethodPost, "/api/v1/daily-diets", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "live-concurrent-key")
	request.Header.Set("X-CSRF-Token", csrfToken)
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	response, err := server.Test(request)
	if err != nil {
		errors <- err
		return
	}
	responses <- response
}

func openDailyDietAPIIntegrationDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	migrationDir, err := filepath.Abs("../../../database/migrations")
	if err != nil {
		t.Fatalf("resolve migration directory: %v", err)
	}
	return testdatabase.Reset(t, migrationDir)
}

func liveDailyDietAPIConfig() config.Config {
	return config.Config{
		APITimeout: time.Second, AllowedOrigins: []string{"http://localhost:5173"}, FrontendOrigin: "http://localhost:5173", Environment: "development",
		Account: config.AccountConfig{AccessTokenTTL: 15 * time.Minute, RefreshTokenTTL: 7 * 24 * time.Hour, AccessCookieName: "__Host-live_access", RefreshCookieName: "__Host-live_refresh", CurrentPrivacyPolicyVersion: "privacy-v1", CurrentTermsVersion: "terms-v1"},
	}
}

func registerLiveDailyDietUser(t *testing.T, server *fiber.App, cfg config.Config, email string) ([]*http.Cookie, uuid.UUID) {
	response := liveDailyDietRequest(t, server, fiber.MethodPost, "/api/v1/auth/register", fmt.Sprintf(`{"email":%q,"password":"StrongerPassword1!","privacyPolicyVersion":"privacy-v1","termsVersion":"terms-v1"}`, email), nil, "", "")
	envelope := decodeLiveDailyDietEnvelope(t, response)
	cookies := response.Cookies()
	response.Body.Close()
	if response.StatusCode != fiber.StatusCreated {
		t.Fatalf("register %s status=%d body=%+v", email, response.StatusCode, envelope)
	}
	return cookies, liveUUIDFromData(t, envelope.Data, "userId")
}

func fetchLiveDailyDietCSRF(t *testing.T, server *fiber.App, cookies []*http.Cookie) (string, []*http.Cookie) {
	response := liveDailyDietRequest(t, server, fiber.MethodGet, "/api/v1/auth/csrf-token", "", cookies, "", "")
	envelope := decodeLiveDailyDietEnvelope(t, response)
	updates := response.Cookies()
	response.Body.Close()
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("csrf status=%d body=%+v", response.StatusCode, envelope)
	}
	token, ok := envelope.Data["csrfToken"].(string)
	if !ok || token == "" {
		t.Fatalf("csrf token response=%+v", envelope)
	}
	return token, mergeLiveDailyDietCookies(cookies, updates)
}

func liveDailyDietRequest(t *testing.T, server *fiber.App, method, path, body string, cookies []*http.Cookie, idempotencyKey, csrfToken string) *http.Response {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	if idempotencyKey != "" {
		request.Header.Set("Idempotency-Key", idempotencyKey)
	}
	if csrfToken != "" {
		request.Header.Set("X-CSRF-Token", csrfToken)
	}
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	response, err := server.Test(request)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	return response
}

func decodeLiveDailyDietEnvelope(t *testing.T, response *http.Response) httpapi.Envelope {
	t.Helper()
	var envelope httpapi.Envelope
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode %s response: %v", response.Status, err)
	}
	return envelope
}

func assertLiveDailyDietStatus(t *testing.T, response *http.Response, want int) {
	t.Helper()
	if response.StatusCode != want {
		body, _ := io.ReadAll(response.Body)
		response.Body.Close()
		t.Fatalf("status=%d want=%d body=%s", response.StatusCode, want, body)
	}
	response.Body.Close()
}

func liveUUIDFromData(t *testing.T, data map[string]any, field string) uuid.UUID {
	t.Helper()
	value, ok := data[field].(string)
	if !ok {
		t.Fatalf("missing %s in response data: %+v", field, data)
	}
	id, err := uuid.Parse(value)
	if err != nil {
		t.Fatalf("invalid %s %q: %v", field, value, err)
	}
	return id
}

func assertLiveAggregate(t *testing.T, data map[string]any, protein, carbohydrates, fat, calories float64) {
	t.Helper()
	macros, ok := data["aggregateMacros"].(map[string]any)
	if !ok || macros["protein"] != protein || macros["carbohydrates"] != carbohydrates || macros["fat"] != fat || macros["calories"] != calories {
		t.Fatalf("aggregate=%+v want protein=%v carbohydrates=%v fat=%v calories=%v", macros, protein, carbohydrates, fat, calories)
	}
}

func countLiveSavedDiets(t *testing.T, db *pgxpool.Pool, userID uuid.UUID) int {
	t.Helper()
	var count int
	if err := db.QueryRow(context.Background(), `SELECT count(*) FROM saved_diets WHERE user_id = $1`, userID).Scan(&count); err != nil {
		t.Fatalf("count saved diets: %v", err)
	}
	return count
}

func liveDailyDietBody(name string, mealA, mealB uuid.UUID) string {
	return fmt.Sprintf(`{"name":%q,"entries":[{"foodObjectId":%q,"foodObjectType":"meal","quantity":100,"unit":"g","position":0},{"foodObjectId":%q,"foodObjectType":"meal","quantity":200,"unit":"g","position":1}]}`, name, mealA.String(), mealB.String())
}

func liveMixedDailyDietBody(name string, foodItem, meal uuid.UUID) string {
	return fmt.Sprintf(`{"name":%q,"entries":[{"foodObjectId":%q,"foodObjectType":"food_item","quantity":100,"unit":"ml","position":0},{"foodObjectId":%q,"foodObjectType":"meal","quantity":200,"unit":"g","position":1}]}`, name, foodItem.String(), meal.String())
}

func mergeLiveDailyDietCookies(existing, updates []*http.Cookie) []*http.Cookie {
	byName := map[string]*http.Cookie{}
	for _, cookie := range existing {
		byName[cookie.Name] = cookie
	}
	for _, cookie := range updates {
		byName[cookie.Name] = cookie
	}
	result := make([]*http.Cookie, 0, len(byName))
	for _, cookie := range byName {
		result = append(result, cookie)
	}
	return result
}
