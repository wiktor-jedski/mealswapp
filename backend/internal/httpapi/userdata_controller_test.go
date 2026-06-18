package httpapi

// Implements DESIGN-008 SavedDataRepository and SearchHistoryRepository verification.

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/userdata"
)

type fakeUserDataService struct {
	items        []repository.SavedItem
	history      []userdata.SearchHistoryEntry
	gotUser      uuid.UUID
	gotItemID    uuid.UUID
	gotKind      repository.SavedItemKind
	clearCalled  bool
	listSavedErr error
	deleteErr    error
	historyErr   error
	clearErr     error
}

func (s *fakeUserDataService) ListSaved(_ context.Context, userID uuid.UUID, kind *repository.SavedItemKind) ([]repository.SavedItem, error) {
	s.gotUser = userID
	if s.listSavedErr != nil {
		return nil, s.listSavedErr
	}
	if kind == nil {
		return s.items, nil
	}
	filtered := []repository.SavedItem{}
	for _, item := range s.items {
		if item.Kind == *kind {
			filtered = append(filtered, item)
		}
	}
	return filtered, nil
}

func (s *fakeUserDataService) DeleteSaved(_ context.Context, userID uuid.UUID, itemID uuid.UUID, kind repository.SavedItemKind) error {
	s.gotUser = userID
	s.gotItemID = itemID
	s.gotKind = kind
	return s.deleteErr
}

func (s *fakeUserDataService) ListHistory(_ context.Context, userID uuid.UUID, _ int) ([]userdata.SearchHistoryEntry, error) {
	s.gotUser = userID
	if s.historyErr != nil {
		return nil, s.historyErr
	}
	return s.history, nil
}

func (s *fakeUserDataService) ClearHistory(_ context.Context, userID uuid.UUID) error {
	s.gotUser = userID
	s.clearCalled = true
	return s.clearErr
}

func TestUserDataControllerServiceFailures(t *testing.T) {
	cfg := testConfig()
	authenticator, authCookies := testJWTAuth(t, cfg, uuid.New(), nil)
	csrf := NewCSRFManager(cfg, nil)
	service := &fakeUserDataService{}
	app := mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, CSRF: csrf, Routes: NewUserDataController(service).Routes()})
	wantErr := errors.New("repository failed")

	service.listSavedErr = wantErr
	req := httptest.NewRequest(fiber.MethodGet, "/api/v1/saved-items", nil)
	addCookies(req, authCookies)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("list saved failure = %d", resp.StatusCode)
	}
	service.historyErr = wantErr
	req = httptest.NewRequest(fiber.MethodGet, "/api/v1/search-history", nil)
	addCookies(req, authCookies)
	resp, _ = app.Test(req)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("list history failure = %d", resp.StatusCode)
	}

	token, csrfCookies := fetchCSRFToken(t, app)
	service.deleteErr = wantErr
	req = httptest.NewRequest(fiber.MethodDelete, "/api/v1/saved-items/favorite/"+uuid.NewString(), nil)
	req.Header.Set("X-CSRF-Token", token)
	addCookies(req, csrfCookies)
	addCookies(req, authCookies)
	resp, _ = app.Test(req)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("delete saved failure = %d", resp.StatusCode)
	}
	token, csrfCookies = fetchCSRFToken(t, app, csrfCookies...)
	service.clearErr = wantErr
	req = httptest.NewRequest(fiber.MethodDelete, "/api/v1/search-history", nil)
	req.Header.Set("X-CSRF-Token", token)
	addCookies(req, csrfCookies)
	addCookies(req, authCookies)
	resp, _ = app.Test(req)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("clear history failure = %d", resp.StatusCode)
	}
}

// TestUserDataControllerSavedItemsAndHistory verifies DESIGN-008 authenticated routes.
func TestUserDataControllerSavedItemsAndHistory(t *testing.T) {
	cfg := testConfig()
	userID := uuid.New()
	authenticator, authCookies := testJWTAuth(t, cfg, userID, nil)
	csrf := NewCSRFManager(cfg, nil)
	itemID := uuid.New()
	service := &fakeUserDataService{
		items:   []repository.SavedItem{{ID: uuid.New(), UserID: userID, ItemID: itemID, Kind: repository.SavedItemKindFavorite}},
		history: []userdata.SearchHistoryEntry{{ID: uuid.New(), Query: "tomato", Mode: "search", FiltersHash: "hash"}},
	}
	controller := NewUserDataController(service)
	app := mustNewRouter(t, Dependencies{Config: cfg, Auth: authenticator, CSRF: csrf, Routes: controller.Routes()})

	req := httptest.NewRequest(fiber.MethodGet, "/api/v1/saved-items?kind=favorite&userId="+uuid.NewString(), nil)
	addCookies(req, authCookies)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	body := decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK || service.gotUser != userID || len(body.Data["items"].([]any)) != 1 {
		t.Fatalf("saved response = %d body=%+v user=%s", resp.StatusCode, body, service.gotUser)
	}

	req = httptest.NewRequest(fiber.MethodGet, "/api/v1/search-history", nil)
	addCookies(req, authCookies)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	body = decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK || len(body.Data["history"].([]any)) != 1 {
		t.Fatalf("history response = %d body=%+v", resp.StatusCode, body)
	}

	req = httptest.NewRequest(fiber.MethodDelete, "/api/v1/saved-items/favorite/"+itemID.String(), nil)
	addCookies(req, authCookies)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusForbidden {
		t.Fatalf("delete without csrf = %d", resp.StatusCode)
	}

	token, csrfCookies := fetchCSRFToken(t, app)
	req = httptest.NewRequest(fiber.MethodDelete, "/api/v1/saved-items/favorite/"+itemID.String(), nil)
	req.Header.Set("X-CSRF-Token", token)
	addCookies(req, csrfCookies)
	addCookies(req, authCookies)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusNoContent || service.gotItemID != itemID || service.gotKind != repository.SavedItemKindFavorite {
		t.Fatalf("delete response = %d item=%s kind=%s", resp.StatusCode, service.gotItemID, service.gotKind)
	}

	token, csrfCookies = fetchCSRFToken(t, app, csrfCookies...)
	req = httptest.NewRequest(fiber.MethodDelete, "/api/v1/search-history", nil)
	req.Header.Set("X-CSRF-Token", token)
	addCookies(req, csrfCookies)
	addCookies(req, authCookies)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusNoContent || !service.clearCalled {
		t.Fatalf("clear response = %d called=%v", resp.StatusCode, service.clearCalled)
	}
}
