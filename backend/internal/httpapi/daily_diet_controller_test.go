package httpapi

// Implements DESIGN-008 ProfileController verification.

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/dailydiet"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

type fakeDailyDietService struct {
	diet        dailydiet.DailyDiet
	createErr   error
	getErr      error
	listErr     error
	replaceErr  error
	deleteErr   error
	createReq   dailydiet.CreateRequest
	createUser  uuid.UUID
	getUser     uuid.UUID
	replaceUser uuid.UUID
	deleteUser  uuid.UUID
}

func (s *fakeDailyDietService) Create(_ context.Context, userID uuid.UUID, req dailydiet.CreateRequest) (dailydiet.CreateResult, error) {
	s.createUser, s.createReq = userID, req
	return dailydiet.CreateResult{Diet: s.diet, Status: fiber.StatusCreated}, s.createErr
}

func (s *fakeDailyDietService) Get(_ context.Context, userID, _ uuid.UUID) (dailydiet.DailyDiet, error) {
	s.getUser = userID
	return s.diet, s.getErr
}

func (s *fakeDailyDietService) List(_ context.Context, userID uuid.UUID) ([]dailydiet.DailyDiet, error) {
	s.getUser = userID
	return []dailydiet.DailyDiet{s.diet}, s.listErr
}

func (s *fakeDailyDietService) Replace(_ context.Context, userID, _ uuid.UUID, _ dailydiet.ReplaceRequest) (dailydiet.DailyDiet, error) {
	s.replaceUser = userID
	return s.diet, s.replaceErr
}

func (s *fakeDailyDietService) Delete(_ context.Context, userID, _ uuid.UUID) error {
	s.deleteUser = userID
	return s.deleteErr
}

func TestProfileControllerDailyDietCRUDUsesJWTUserAndCSRF(t *testing.T) {
	cfg := testConfig()
	userID := uuid.New()
	authenticator, authCookies := testJWTAuth(t, cfg, userID, nil)
	mealA, mealB, dietID := uuid.New(), uuid.New(), uuid.New()
	service := &fakeDailyDietService{diet: dailydiet.DailyDiet{
		ID: dietID, Name: "Training Day", CreatedAt: time.Unix(1, 0).UTC(), UpdatedAt: time.Unix(2, 0).UTC(),
		Entries:         []dailydiet.DailyDietEntry{{ID: uuid.New(), MealID: mealA, Quantity: 100, Unit: "g", Position: 0}, {ID: uuid.New(), MealID: mealB, Quantity: 200, Unit: "g", Position: 1}},
		AggregateMacros: dailydiet.MacroProjection{Protein: 20, Carbohydrates: 30, Fat: 9, Calories: 281},
	}}
	controller := NewProfileController(&fakeProfileService{}, service)
	app := mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, CSRF: NewCSRFManager(cfg, nil), Routes: controller.Routes()})

	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/daily-diets", nil))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("anonymous list status = %d, want 401", resp.StatusCode)
	}

	postBody := `{"name":"Training Day","entries":[{"mealId":"` + mealA.String() + `","quantity":100,"unit":"g","position":0},{"mealId":"` + mealB.String() + `","quantity":200,"unit":"g","position":1}]}`
	request := httptest.NewRequest(fiber.MethodPost, "/api/v1/daily-diets", strings.NewReader(postBody))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "daily-key-1")
	addCookies(request, authCookies)
	resp, err = app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusForbidden || service.createUser != uuid.Nil {
		t.Fatalf("missing csrf create status=%d createUser=%s", resp.StatusCode, service.createUser)
	}

	csrfToken, csrfCookies := fetchCSRFToken(t, app)
	request = httptest.NewRequest(fiber.MethodPost, "/api/v1/daily-diets", strings.NewReader(postBody))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "daily-key-1")
	request.Header.Set("X-CSRF-Token", csrfToken)
	addCookies(request, csrfCookies)
	addCookies(request, authCookies)
	resp, err = app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	body := decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusCreated || service.createUser != userID || service.createReq.IdempotencyKey != "daily-key-1" {
		t.Fatalf("create status=%d body=%+v user=%s req=%+v", resp.StatusCode, body, service.createUser, service.createReq)
	}
	if _, exposed := body.Data["userId"]; exposed {
		t.Fatalf("create response exposed userId: %+v", body.Data)
	}
	macros, ok := body.Data["aggregateMacros"].(map[string]any)
	if !ok || macros["calories"] != float64(281) {
		t.Fatalf("aggregate response = %#v", body.Data["aggregateMacros"])
	}

	request = httptest.NewRequest(fiber.MethodGet, "/api/v1/daily-diets", nil)
	addCookies(request, authCookies)
	resp, err = app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	body = decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	diets, ok := body.Data["diets"].([]any)
	if resp.StatusCode != fiber.StatusOK || !ok || len(diets) != 1 || service.getUser != userID {
		t.Fatalf("list status=%d body=%+v user=%s", resp.StatusCode, body, service.getUser)
	}

	request = httptest.NewRequest(fiber.MethodGet, "/api/v1/daily-diets/"+dietID.String(), nil)
	addCookies(request, authCookies)
	resp, err = app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	body = decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK || service.getUser != userID || body.Data["id"] != dietID.String() {
		t.Fatalf("get status=%d body=%+v user=%s", resp.StatusCode, body, service.getUser)
	}

	request = httptest.NewRequest(fiber.MethodPut, "/api/v1/daily-diets/"+dietID.String(), strings.NewReader(postBody))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-CSRF-Token", csrfToken)
	addCookies(request, csrfCookies)
	addCookies(request, authCookies)
	resp, err = app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK || service.replaceUser != userID {
		t.Fatalf("replace status=%d user=%s", resp.StatusCode, service.replaceUser)
	}

	request = httptest.NewRequest(fiber.MethodPut, "/api/v1/daily-diets/"+dietID.String(), strings.NewReader(postBody))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-CSRF-Token", csrfToken)
	addCookies(request, csrfCookies)
	addCookies(request, authCookies)
	resp, err = app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("repeated replace status=%d", resp.StatusCode)
	}

	request = httptest.NewRequest(fiber.MethodDelete, "/api/v1/daily-diets/"+dietID.String(), nil)
	request.Header.Set("X-CSRF-Token", csrfToken)
	addCookies(request, csrfCookies)
	addCookies(request, authCookies)
	resp, err = app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusNoContent || service.deleteUser != userID {
		t.Fatalf("delete status=%d user=%s", resp.StatusCode, service.deleteUser)
	}

	request = httptest.NewRequest(fiber.MethodDelete, "/api/v1/daily-diets/"+dietID.String(), nil)
	request.Header.Set("X-CSRF-Token", csrfToken)
	addCookies(request, csrfCookies)
	addCookies(request, authCookies)
	resp, err = app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("repeated delete status=%d", resp.StatusCode)
	}
}

func TestProfileControllerDailyDietRejectsClientOwnershipInvalidBodyAndStableErrors(t *testing.T) {
	cfg := testConfig()
	userID := uuid.New()
	authenticator, authCookies := testJWTAuth(t, cfg, userID, nil)
	service := &fakeDailyDietService{createErr: dailydiet.ErrIdempotencyConflict}
	controller := NewProfileController(&fakeProfileService{}, service)
	app := mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, CSRF: NewCSRFManager(cfg, nil), Routes: controller.Routes()})
	csrfToken, csrfCookies := fetchCSRFToken(t, app)
	validBody := `{"name":"Training Day","entries":[{"mealId":"` + uuid.NewString() + `","quantity":100,"unit":"g","position":0}]}`

	request := httptest.NewRequest(fiber.MethodPost, "/api/v1/daily-diets", strings.NewReader(validBody))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "daily-key-2")
	request.Header.Set("X-CSRF-Token", csrfToken)
	addCookies(request, csrfCookies)
	addCookies(request, authCookies)
	resp, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	body := decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusConflict || body.Error == nil || body.Error.Code != "idempotency_key_conflict" {
		t.Fatalf("conflict response = %d %+v", resp.StatusCode, body)
	}

	invalidBody := `{"name":"Training Day","userId":"` + uuid.NewString() + `","entries":[{"mealId":"` + uuid.NewString() + `","quantity":100,"unit":"g","position":0}]}`
	request = httptest.NewRequest(fiber.MethodPost, "/api/v1/daily-diets", strings.NewReader(invalidBody))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "daily-key-3")
	request.Header.Set("X-CSRF-Token", csrfToken)
	addCookies(request, csrfCookies)
	addCookies(request, authCookies)
	resp, err = app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusBadRequest || service.createReq.IdempotencyKey != "daily-key-2" {
		t.Fatalf("invalid ownership body status=%d createReq=%+v", resp.StatusCode, service.createReq)
	}

	service.createErr = repository.NewError(repository.ErrorKindNotFound, "meal not found", nil)
	request = httptest.NewRequest(fiber.MethodPost, "/api/v1/daily-diets", strings.NewReader(validBody))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "daily-key-4")
	request.Header.Set("X-CSRF-Token", csrfToken)
	addCookies(request, csrfCookies)
	addCookies(request, authCookies)
	resp, err = app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	body = decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusNotFound || body.Error == nil || body.Error.Code != "not_found" {
		t.Fatalf("missing-meal response = %d %+v", resp.StatusCode, body)
	}

	service.createErr = errors.New("unexpected service failure")
	request = httptest.NewRequest(fiber.MethodPost, "/api/v1/daily-diets", strings.NewReader(validBody))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "short")
	request.Header.Set("X-CSRF-Token", csrfToken)
	addCookies(request, csrfCookies)
	addCookies(request, authCookies)
	resp, err = app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	body = decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusBadRequest || body.Error == nil || body.Error.Code != "idempotency_key_required" {
		t.Fatalf("missing-key response = %d %+v", resp.StatusCode, body)
	}
}
