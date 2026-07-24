package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/search"
)

type filterOptionReaderStub struct {
	response search.FilterOptionsResponse
	err      error
	mode     search.SearchMode
	calls    int
}

func (s *filterOptionReaderStub) Options(_ context.Context, mode search.SearchMode) (search.FilterOptionsResponse, error) {
	s.calls++
	s.mode = mode
	return s.response, s.err
}

// Implements DESIGN-009 TagManager public-read and route projection verification.
func TestFilterOptionControllerReturnsServiceOwnedOptionsAnonymously(t *testing.T) {
	service := &filterOptionReaderStub{response: search.FilterOptionsResponse{Mode: search.SearchModeSubstitution, Options: []search.FilterOption{{
		FilterID: "repository-owned-id", Kind: search.SearchFilterKindFoodCategory, Label: "Persisted label", IncludeAllowed: true, ExcludeAllowed: true,
	}, {
		FilterID: "policy-owned-id", Kind: search.SearchFilterKindDietaryPreset, Label: "Policy label", LabelKey: "filter.policy.label", ExcludeAllowed: true,
		Excludes: []search.FilterOptionReference{{FilterID: "persisted-allergen", Kind: search.SearchFilterKindAllergen}},
	}}}}
	controller := NewFilterOptionController(service)
	routes := controller.Routes()
	if len(routes) != 1 || routes[0].RequiresAuth || routes[0].OptionalAuth || routes[0].RequiresCSRF || routes[0].Method != fiber.MethodGet {
		t.Fatalf("public route security = %#v", routes)
	}
	app := mustNewRouter(t, Dependencies{Config: testConfig(), Routes: routes})
	req := httptest.NewRequest(fiber.MethodGet, "/api/v1/search/filter-options?mode=substitution", nil)
	req.Header.Set("X-User-Role", "admin")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusOK || service.calls != 1 || service.mode != search.SearchModeSubstitution {
		t.Fatalf("response status=%d calls=%d mode=%q body=%s", resp.StatusCode, service.calls, service.mode, body)
	}
	var envelope struct {
		Status string `json:"status"`
		Data   struct {
			Mode    string            `json:"mode"`
			Options []filterOptionDTO `json:"options"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Status != "ok" || envelope.Data.Mode != "substitution" || len(envelope.Data.Options) != 2 || envelope.Data.Options[0].FilterID != "repository-owned-id" || envelope.Data.Options[1].FilterID != "policy-owned-id" || len(envelope.Data.Options[1].Excludes) != 1 {
		t.Fatalf("projected envelope = %#v", envelope)
	}
}

// Implements DESIGN-009 TagManager supported-mode and structured-failure verification.
func TestFilterOptionControllerRejectsInvalidModeAndStructuresDependencyFailure(t *testing.T) {
	service := &filterOptionReaderStub{}
	app := mustNewRouter(t, Dependencies{Config: testConfig(), Routes: NewFilterOptionController(service).Routes()})
	for _, path := range []string{
		"/api/v1/search/filter-options",
		"/api/v1/search/filter-options?mode=catalog",
		"/api/v1/search/filter-options?mode=substitution&extra=true",
	} {
		resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, path, nil))
		if err != nil {
			t.Fatal(err)
		}
		body := decodeEnvelope(t, resp.Body)
		resp.Body.Close()
		if resp.StatusCode != fiber.StatusBadRequest || body.Error == nil || body.Error.Code != "validation_failed" {
			t.Fatalf("GET %s = %d %#v", path, resp.StatusCode, body)
		}
	}
	if service.calls != 0 {
		t.Fatalf("invalid modes dispatched %d calls", service.calls)
	}

	service.err = repository.NewError(repository.ErrorKindConnection, "sensitive database host", errors.New("secret socket"))
	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/search/filter-options?mode=substitution", nil))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusServiceUnavailable || !strings.Contains(string(raw), `"category":"dependency"`) || !strings.Contains(string(raw), `"code":"dependency_unavailable"`) || strings.Contains(string(raw), "sensitive") || strings.Contains(string(raw), "secret socket") {
		t.Fatalf("dependency response = %d %s", resp.StatusCode, raw)
	}
}
