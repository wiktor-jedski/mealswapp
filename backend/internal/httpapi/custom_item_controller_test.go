package httpapi

// Implements DESIGN-008 ProfileController custom-item HTTP verification.

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/customitem"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

type fakeCustomItemService struct {
	item       customitem.Item
	createUser uuid.UUID
	createReq  customitem.CreateRequest
	getUser    uuid.UUID
	updateUser uuid.UUID
	deleteUser uuid.UUID
	err        error
}

func (s *fakeCustomItemService) Create(_ context.Context, userID uuid.UUID, req customitem.CreateRequest) (customitem.CreateResult, error) {
	s.createUser, s.createReq = userID, req
	return customitem.CreateResult{Item: s.item, Status: fiber.StatusCreated}, s.err
}
func (s *fakeCustomItemService) Get(_ context.Context, userID, _ uuid.UUID) (customitem.Item, error) {
	s.getUser = userID
	return s.item, s.err
}
func (s *fakeCustomItemService) Update(_ context.Context, userID, _ uuid.UUID, _ customitem.Request) (customitem.Item, error) {
	s.updateUser = userID
	return s.item, s.err
}
func (s *fakeCustomItemService) Delete(_ context.Context, userID, _ uuid.UUID) error {
	s.deleteUser = userID
	return s.err
}

func customItemBody(name string) string {
	return `{"name":"` + name + `","physicalState":"solid","prepTimeMinutes":0,"macrosPer100":{"protein":10,"carbohydrates":5,"fat":2},"micros":{},"foodCategoryIds":[],"culinaryRoleIds":[]}`
}

func TestProfileControllerCustomItemRoutesRequireAuthenticationAndCSRF(t *testing.T) {
	cfg := testConfig()
	userID, itemID := uuid.New(), uuid.New()
	authenticator, authCookies := testJWTAuth(t, cfg, userID, nil)
	service := &fakeCustomItemService{item: customitem.Item{ID: itemID, Name: "Tofu", PhysicalState: repository.PhysicalStateSolid}}
	controller := NewProfileController(&fakeProfileService{}).WithCustomItems(service)
	app := mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, CSRF: NewCSRFManager(cfg, nil), Routes: controller.Routes()})

	for _, route := range []struct{ method, path string }{
		{fiber.MethodPost, "/api/v1/custom-items"},
		{fiber.MethodGet, "/api/v1/custom-items/" + itemID.String()},
		{fiber.MethodPut, "/api/v1/custom-items/" + itemID.String()},
		{fiber.MethodDelete, "/api/v1/custom-items/" + itemID.String()},
	} {
		resp, err := app.Test(httptest.NewRequest(route.method, route.path, strings.NewReader(customItemBody("Tofu"))))
		if err != nil {
			t.Fatal(err)
		}
		body := decodeEnvelope(t, resp.Body)
		resp.Body.Close()
		if resp.StatusCode != fiber.StatusUnauthorized || body.Error == nil || body.Error.Code != "unauthorized" {
			t.Fatalf("anonymous %s = %d %+v", route.method, resp.StatusCode, body)
		}
	}
	for _, route := range []struct{ method, path, body string }{
		{fiber.MethodPost, "/api/v1/custom-items", customItemBody("Tofu")},
		{fiber.MethodPut, "/api/v1/custom-items/" + itemID.String(), customItemBody("Tofu")},
		{fiber.MethodDelete, "/api/v1/custom-items/" + itemID.String(), ""},
	} {
		request := httptest.NewRequest(route.method, route.path, strings.NewReader(route.body))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Idempotency-Key", "custom-key-http")
		addCookies(request, authCookies)
		resp, err := app.Test(request)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != fiber.StatusForbidden {
			t.Fatalf("missing CSRF %s = %d", route.method, resp.StatusCode)
		}
	}

	request := httptest.NewRequest(fiber.MethodPost, "/api/v1/custom-items", strings.NewReader(customItemBody("Tofu")))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "custom-key-http")
	addCookies(request, authCookies)
	resp, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusForbidden || service.createUser != uuid.Nil {
		t.Fatalf("missing CSRF create = %d user=%s", resp.StatusCode, service.createUser)
	}

	token, csrfCookies := fetchCSRFToken(t, app)
	request = httptest.NewRequest(fiber.MethodPost, "/api/v1/custom-items", strings.NewReader(customItemBody("Tofu")))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "custom-key-http")
	request.Header.Set("X-CSRF-Token", token)
	addCookies(request, authCookies)
	addCookies(request, csrfCookies)
	resp, err = app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	body := decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusCreated || service.createUser != userID || service.createReq.IdempotencyKey != "custom-key-http" {
		t.Fatalf("create = %d body=%+v user=%s req=%+v", resp.StatusCode, body, service.createUser, service.createReq)
	}
	if _, exposed := body.Data["ownerId"]; exposed {
		t.Fatalf("response exposed owner: %+v", body.Data)
	}

	request = httptest.NewRequest(fiber.MethodPut, "/api/v1/custom-items/"+itemID.String(), strings.NewReader(customItemBody("Updated")))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-CSRF-Token", token)
	addCookies(request, authCookies)
	addCookies(request, csrfCookies)
	resp, err = app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK || service.updateUser != userID {
		t.Fatalf("update = %d user=%s", resp.StatusCode, service.updateUser)
	}

	request = httptest.NewRequest(fiber.MethodDelete, "/api/v1/custom-items/"+itemID.String(), nil)
	request.Header.Set("X-CSRF-Token", token)
	addCookies(request, authCookies)
	addCookies(request, csrfCookies)
	resp, err = app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusNoContent || service.deleteUser != userID {
		t.Fatalf("delete = %d user=%s", resp.StatusCode, service.deleteUser)
	}
}

func TestProfileControllerCustomItemRejectsClientOwnershipAndMapsSafeErrors(t *testing.T) {
	cfg := testConfig()
	userID, itemID := uuid.New(), uuid.New()
	authenticator, authCookies := testJWTAuth(t, cfg, userID, nil)
	service := &fakeCustomItemService{item: customitem.Item{ID: itemID}, err: repository.NewError(repository.ErrorKindNotFound, "private detail", nil)}
	controller := NewProfileController(&fakeProfileService{}).WithCustomItems(service)
	app := mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, CSRF: NewCSRFManager(cfg, nil), Routes: controller.Routes()})

	request := httptest.NewRequest(fiber.MethodGet, "/api/v1/custom-items/"+itemID.String(), nil)
	addCookies(request, authCookies)
	resp, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	body := decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusNotFound || body.Error == nil || body.Error.Code != "not_found" || strings.Contains(body.Error.Message, "private") || service.getUser != userID {
		t.Fatalf("cross-user-safe not found = %d %+v user=%s", resp.StatusCode, body, service.getUser)
	}

	service.err = nil
	token, csrfCookies := fetchCSRFToken(t, app)
	service.err = customitem.ErrIdempotencyConflict
	request = httptest.NewRequest(fiber.MethodPost, "/api/v1/custom-items", strings.NewReader(customItemBody("Tofu")))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "custom-key-conflict")
	request.Header.Set("X-CSRF-Token", token)
	addCookies(request, authCookies)
	addCookies(request, csrfCookies)
	resp, err = app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	body = decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusConflict || body.Error == nil || body.Error.Code != "idempotency_key_conflict" {
		t.Fatalf("idempotency conflict = %d %+v", resp.StatusCode, body)
	}

	service.err = repository.NewError(repository.ErrorKindConflict, "duplicate custom item name", nil)
	request = httptest.NewRequest(fiber.MethodPost, "/api/v1/custom-items", strings.NewReader(customItemBody("Tofu")))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "different-custom-key")
	request.Header.Set("X-CSRF-Token", token)
	addCookies(request, authCookies)
	addCookies(request, csrfCookies)
	resp, err = app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	body = decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusConflict || body.Error == nil || body.Error.Code != "conflict" || strings.Contains(body.Error.Message, "duplicate") {
		t.Fatalf("resource conflict = %d %+v", resp.StatusCode, body)
	}

	service.err = nil
	service.createUser = uuid.Nil
	ownedBody := strings.TrimSuffix(customItemBody("Tofu"), "}") + `,"ownerId":"` + uuid.NewString() + `"}`
	request = httptest.NewRequest(fiber.MethodPost, "/api/v1/custom-items", strings.NewReader(ownedBody))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "custom-key-owner")
	request.Header.Set("X-CSRF-Token", token)
	addCookies(request, authCookies)
	addCookies(request, csrfCookies)
	resp, err = app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	body = decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusBadRequest || body.Error == nil || body.Error.Code != "invalid_json" || service.createUser != uuid.Nil {
		t.Fatalf("client owner body = %d %+v serviceUser=%s", resp.StatusCode, body, service.createUser)
	}
}

func TestProfileControllerCustomItemRejectsEscapedNULProvenanceBeforeService(t *testing.T) {
	cfg := testConfig()
	userID := uuid.New()
	authenticator, authCookies := testJWTAuth(t, cfg, userID, nil)
	service := &fakeCustomItemService{}
	controller := NewProfileController(&fakeProfileService{}).WithCustomItems(service)
	app := mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, CSRF: NewCSRFManager(cfg, nil), Routes: controller.Routes()})
	token, csrfCookies := fetchCSRFToken(t, app)

	for _, field := range []string{"densitySourceProvider", "densitySourceFoodId", "densitySourceKind"} {
		service.createUser = uuid.Nil
		body := `{"name":"Liquid","physicalState":"liquid","densityGramsPerMilliliter":1,"densitySourceKind":"manual","macrosPer100":{"protein":1,"carbohydrates":0,"fat":0},"micros":{},"` + field + `":"invalid\u0000text"}`
		request := httptest.NewRequest(fiber.MethodPost, "/api/v1/custom-items", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Idempotency-Key", "nul-provenance-key")
		request.Header.Set("X-CSRF-Token", token)
		addCookies(request, authCookies)
		addCookies(request, csrfCookies)
		resp, err := app.Test(request)
		if err != nil {
			t.Fatal(err)
		}
		envelope := decodeEnvelope(t, resp.Body)
		resp.Body.Close()
		if resp.StatusCode != fiber.StatusBadRequest || envelope.Error == nil || envelope.Error.Code != "validation_failed" || service.createUser != uuid.Nil {
			t.Fatalf("escaped NUL %s = %d %+v serviceUser=%s", field, resp.StatusCode, envelope, service.createUser)
		}
	}
}

func TestProfileControllerCustomItemRejectsSchemaAndDomainViolationsBeforeService(t *testing.T) {
	cfg := testConfig()
	userID := uuid.New()
	authenticator, authCookies := testJWTAuth(t, cfg, userID, nil)
	service := &fakeCustomItemService{}
	controller := NewProfileController(&fakeProfileService{}).WithCustomItems(service)
	app := mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, CSRF: NewCSRFManager(cfg, nil), Routes: controller.Routes()})
	token, csrfCookies := fetchCSRFToken(t, app)
	categoryID := uuid.NewString()
	tests := []string{
		`{"name":"   ","physicalState":"solid","macrosPer100":{"protein":1,"carbohydrates":0,"fat":0},"micros":{}}`,
		`{"name":"bad\u0000name","physicalState":"solid","macrosPer100":{"protein":1,"carbohydrates":0,"fat":0},"micros":{}}`,
		`{"name":"X","physicalState":"solid","micros":{}}`,
		`{"name":"X","physicalState":"solid","macrosPer100":null,"micros":{}}`,
		`{"name":"X","physicalState":"solid","macrosPer100":{"protein":1,"carbohydrates":2},"micros":{}}`,
		`{"name":"X","physicalState":"solid","macrosPer100":{"protein":-1,"carbohydrates":0,"fat":0},"micros":{}}`,
		`{"name":"X","physicalState":"solid","macrosPer100":{"protein":60,"carbohydrates":41,"fat":0},"micros":{}}`,
		`{"name":"X","physicalState":"solid","averageUnitWeightGrams":0,"macrosPer100":{"protein":1,"carbohydrates":0,"fat":0},"micros":{}}`,
		`{"name":"X","physicalState":"liquid","macrosPer100":{"protein":1,"carbohydrates":0,"fat":0},"micros":{}}`,
		`{"name":"X","physicalState":"solid","macrosPer100":{"protein":1e309,"carbohydrates":0,"fat":0},"micros":{}}`,
		`{"name":"X","physicalState":"solid","macrosPer100":{"protein":1,"carbohydrates":0,"fat":0},"micros":null}`,
		`{"name":"X","physicalState":"solid","macrosPer100":{"protein":1,"carbohydrates":0,"fat":0},"micros":{"Sodium":-1}}`,
		`{"name":"X","physicalState":"solid","macrosPer100":{"protein":1,"carbohydrates":0,"fat":0},"micros":{},"foodCategoryIds":["` + categoryID + `","` + categoryID + `"]}`,
		`{"name":"X","physicalState":"solid","macrosPer100":{"protein":1,"carbohydrates":0,"fat":0},"micros":{},"foodCategoryIds":["00000000-0000-0000-0000-000000000000"]}`,
		`{"name":"X","physicalState":"solid","macrosPer100":{"protein":1,"carbohydrates":0,"fat":0},"micros":{},"imageUrl":"://bad"}`,
		`{"name":"X","physicalState":"solid","macrosPer100":{"protein":1,"carbohydrates":0,"fat":0},"micros":{},"ownerId":"` + uuid.NewString() + `"}`,
	}
	for index, body := range tests {
		service.createUser = uuid.Nil
		request := httptest.NewRequest(fiber.MethodPost, "/api/v1/custom-items", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Idempotency-Key", "custom-invalid-body")
		request.Header.Set("X-CSRF-Token", token)
		addCookies(request, authCookies)
		addCookies(request, csrfCookies)
		resp, err := app.Test(request)
		if err != nil {
			t.Fatal(err)
		}
		envelope := decodeEnvelope(t, resp.Body)
		resp.Body.Close()
		if resp.StatusCode != fiber.StatusBadRequest || envelope.Error == nil || service.createUser != uuid.Nil {
			t.Fatalf("case %d = %d %+v serviceUser=%s", index, resp.StatusCode, envelope, service.createUser)
		}
	}
}

func TestProfileControllerCustomItemClassificationProjectionOmitsParentID(t *testing.T) {
	cfg := testConfig()
	userID, itemID, classificationID := uuid.New(), uuid.New(), uuid.New()
	authenticator, authCookies := testJWTAuth(t, cfg, userID, nil)
	service := &fakeCustomItemService{item: customitem.Item{
		ID: itemID, Name: "Projected", PhysicalState: repository.PhysicalStateSolid,
		FoodCategories: []customitem.ClassificationSummary{{ID: classificationID, Name: "Child", Kind: repository.ClassificationKindFoodCategory}},
		CulinaryRoles:  []customitem.ClassificationSummary{},
	}}
	app := mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, CSRF: NewCSRFManager(cfg, nil), Routes: NewProfileController(&fakeProfileService{}).WithCustomItems(service).Routes()})
	request := httptest.NewRequest(fiber.MethodGet, "/api/v1/custom-items/"+itemID.String(), nil)
	addCookies(request, authCookies)
	resp, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	body := decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	encoded, err := json.Marshal(body.Data)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusOK || !strings.Contains(string(encoded), classificationID.String()) || strings.Contains(string(encoded), "parentId") {
		t.Fatalf("classification HTTP projection = %d %s", resp.StatusCode, encoded)
	}
}

func TestProfileControllerCustomItemMapsInvalidMicronutrientsToStructuredValidation(t *testing.T) {
	cfg := testConfig()
	userID := uuid.New()
	authenticator, authCookies := testJWTAuth(t, cfg, userID, nil)
	service := &fakeCustomItemService{err: repository.NewError(repository.ErrorKindInvalidMicronutrientKey, "inactive internal vocabulary detail", nil)}
	controller := NewProfileController(&fakeProfileService{}).WithCustomItems(service)
	app := mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, CSRF: NewCSRFManager(cfg, nil), Routes: controller.Routes()})
	token, csrfCookies := fetchCSRFToken(t, app)
	body := `{"name":"X","physicalState":"solid","macrosPer100":{"protein":1,"carbohydrates":0,"fat":0},"micros":{"Na":1}}`
	request := httptest.NewRequest(fiber.MethodPost, "/api/v1/custom-items", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "custom-invalid-micro")
	request.Header.Set("X-CSRF-Token", token)
	addCookies(request, authCookies)
	addCookies(request, csrfCookies)
	resp, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	envelope := decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusBadRequest || envelope.Error == nil || envelope.Error.Category != "validation" || envelope.Error.Code != "validation_failed" || strings.Contains(envelope.Error.Message, "internal") {
		t.Fatalf("invalid micronutrient response = %d %+v", resp.StatusCode, envelope)
	}
}
