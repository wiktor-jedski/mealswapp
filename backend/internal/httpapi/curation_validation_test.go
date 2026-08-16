package httpapi

// Implements DESIGN-010 RequestValidator and DESIGN-013 InputNormalizer curation HTTP verification.

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/wiktor-jedski/mealswapp/backend/internal/curation"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
)

type task260BlockedCurationJSONWriter struct {
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func (w *task260BlockedCurationJSONWriter) Write(payload []byte) (int, error) {
	w.once.Do(func() { close(w.started) })
	<-w.release
	return len(payload), nil
}

func TestTask260CurationRejectionCannotBlockRequestOnJSONWriter(t *testing.T) {
	writer := &task260BlockedCurationJSONWriter{started: make(chan struct{}), release: make(chan struct{})}
	t.Cleanup(func() { close(writer.release) })
	telemetry := observability.NewAdminExternalTelemetry(nil, observability.JSONSink{Writer: writer})
	validator := NewCurationRequestValidator(telemetry)
	app := mustNewRouter(t, Dependencies{Config: testConfig(), Routes: []RouteDefinition{{
		Method: fiber.MethodGet, Path: "/admin/external-search", Validate: validator.ValidateExternalSearchQuery,
		Handler: func(ctx *fiber.Ctx) error { return ctx.SendStatus(fiber.StatusNoContent) },
	}}})

	completed := make(chan struct {
		response *http.Response
		err      error
	}, 1)
	go func() {
		response, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/admin/external-search?query=private&provider=usda&page=invalid", nil))
		completed <- struct {
			response *http.Response
			err      error
		}{response, err}
	}()
	select {
	case <-writer.started:
	case <-time.After(time.Second):
		t.Fatal("curation rejection did not reach JSON writer")
	}
	select {
	case result := <-completed:
		if result.err != nil {
			t.Fatal(result.err)
		}
		assertStructuredCuration400(t, result.response)
	case <-time.After(time.Second):
		t.Fatal("blocked JSON writer held curation rejection response")
	}
}

func TestCurationHTTPValidationStopsBeforeProviderOrRepositoryDispatch(t *testing.T) {
	logs := &observability.MemorySink{}
	validator := NewCurationRequestValidator(logs)
	providerCalls, repositoryCalls := 0, 0
	var dispatchedSearch curation.ExternalSearchRequest
	var dispatchedItem curation.ItemRequest
	var dispatchedClassification curation.ClassificationRequest
	app := mustNewRouter(t, Dependencies{Config: testConfig(), Logs: logs, Metrics: logs, Routes: []RouteDefinition{
		{Method: fiber.MethodGet, Path: "/admin/external-search", Validate: validator.ValidateExternalSearchQuery, Handler: func(ctx *fiber.Ctx) error {
			providerCalls++
			dispatchedSearch, _ = NormalizedExternalSearchRequest(ctx)
			return ctx.SendStatus(fiber.StatusNoContent)
		}},
		{Method: fiber.MethodPost, Path: "/admin/items/validate", ExemptCSRF: true, Validate: validator.ValidateItemBody, Handler: func(ctx *fiber.Ctx) error {
			repositoryCalls++
			dispatchedItem, _ = NormalizedCurationItemRequest(ctx)
			return ctx.SendStatus(fiber.StatusNoContent)
		}},
		{Method: fiber.MethodPost, Path: "/admin/classifications/validate", ExemptCSRF: true, Validate: validator.ValidateClassificationBody, Handler: func(ctx *fiber.Ctx) error {
			repositoryCalls++
			dispatchedClassification, _ = NormalizedCurationClassificationRequest(ctx)
			return ctx.SendStatus(fiber.StatusNoContent)
		}},
	}})

	validQuery, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/admin/external-search?query=%20apple%20&provider=USDA&page=1", nil))
	if err != nil {
		t.Fatal(err)
	}
	validQuery.Body.Close()
	if validQuery.StatusCode != fiber.StatusNoContent || providerCalls != 1 {
		t.Fatalf("valid external query status=%d provider calls=%d", validQuery.StatusCode, providerCalls)
	}
	validItem := `{"name":"  Cafe\u0301 au lait ","physicalState":"liquid","imageUrl":"https://images.example.com/cafe.jpg","servingUnit":"millilitres","servingQuantity":250,"sourceProvider":"OpenFoodFacts","externalId":"product:123","providerText":" Cafe drink ","macrosPer100":{"protein":3,"carbohydrates":5,"fat":2},"micronutrients":{"calcium":100}}`
	assertCurationStatus(t, app, fiber.MethodPost, "/api/v1/admin/items/validate", validItem, fiber.StatusNoContent)
	assertCurationStatus(t, app, fiber.MethodPost, "/api/v1/admin/classifications/validate", `{"name":" Fresh   foods "}`, fiber.StatusNoContent)
	if dispatchedSearch.Query != "apple" || dispatchedSearch.Provider != "usda" || dispatchedSearch.Page != 1 {
		t.Fatalf("raw search dispatched: %+v", dispatchedSearch)
	}
	if dispatchedItem.Name != "Café au lait" || dispatchedItem.ServingUnit != "ml" || dispatchedItem.SourceProvider != "openfoodfacts" || dispatchedItem.ProviderText != "Cafe drink" {
		t.Fatalf("raw item dispatched: %+v", dispatchedItem)
	}
	if dispatchedClassification.Name != "Fresh foods" {
		t.Fatalf("raw classification dispatched: %+v", dispatchedClassification)
	}

	for name, path := range map[string]string{
		"empty query":          "/api/v1/admin/external-search?query=&provider=usda&page=1",
		"unsupported provider": "/api/v1/admin/external-search?query=apple&provider=other&page=1",
		"invalid page":         "/api/v1/admin/external-search?query=apple&provider=usda&page=0",
		"non-numeric page":     "/api/v1/admin/external-search?query=apple&provider=usda&page=abc",
		"extra parameter":      "/api/v1/admin/external-search?query=apple&provider=usda&page=1&raw=SECRET",
		"duplicate query":      "/api/v1/admin/external-search?query=apple&query=SECRET&provider=usda&page=1",
	} {
		t.Run(name, func(t *testing.T) {
			resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, path, nil))
			if err != nil {
				t.Fatal(err)
			}
			assertStructuredCuration400(t, resp)
		})
	}

	invalidBodies := map[string]string{
		"control name":           `{"name":"SECRET\nRAW","physicalState":"solid","macrosPer100":{"protein":1,"carbohydrates":1,"fat":1}}`,
		"unsafe image URL":       `{"name":"Item","physicalState":"solid","imageUrl":"http://127.0.0.1/a.jpg","macrosPer100":{"protein":1,"carbohydrates":1,"fat":1}}`,
		"string macro":           `{"name":"Item","physicalState":"solid","macrosPer100":{"protein":"1","carbohydrates":1,"fat":1}}`,
		"negative macro":         `{"name":"Item","physicalState":"solid","macrosPer100":{"protein":-1,"carbohydrates":1,"fat":1}}`,
		"macro total":            `{"name":"Item","physicalState":"solid","macrosPer100":{"protein":40,"carbohydrates":40,"fat":21}}`,
		"string micronutrient":   `{"name":"Item","physicalState":"solid","macrosPer100":{"protein":1,"carbohydrates":1,"fat":1},"micronutrients":{"calcium":"100"}}`,
		"negative micronutrient": `{"name":"Item","physicalState":"solid","macrosPer100":{"protein":1,"carbohydrates":1,"fat":1},"micronutrients":{"calcium":-1}}`,
		"serving mismatch":       `{"name":"Item","physicalState":"solid","servingUnit":"g","macrosPer100":{"protein":1,"carbohydrates":1,"fat":1}}`,
		"provider mismatch":      `{"name":"Item","physicalState":"solid","sourceProvider":"usda","macrosPer100":{"protein":1,"carbohydrates":1,"fat":1}}`,
		"unknown field":          `{"name":"Item","physicalState":"solid","rawPayload":"SECRET","macrosPer100":{"protein":1,"carbohydrates":1,"fat":1}}`,
		"null macro object":      `{"name":"Item","physicalState":"solid","macrosPer100":null}`,
		"null macro field":       `{"name":"Item","physicalState":"solid","macrosPer100":{"protein":null,"carbohydrates":1,"fat":1}}`,
		"duplicate body field":   `{"name":"Item","name":"SECRET","physicalState":"solid","macrosPer100":{"protein":1,"carbohydrates":1,"fat":1}}`,
		"duplicate macro field":  `{"name":"Item","physicalState":"solid","macrosPer100":{"protein":1,"protein":2,"carbohydrates":1,"fat":1}}`,
		"extreme liquid macro":   `{"name":"Item","physicalState":"liquid","macrosPer100":{"protein":1e100,"carbohydrates":1,"fat":1}}`,
		"extreme serving":        `{"name":"Item","physicalState":"liquid","servingUnit":"ml","servingQuantity":1000001,"macrosPer100":{"protein":1,"carbohydrates":1,"fat":1}}`,
		"extreme micronutrient":  `{"name":"Item","physicalState":"liquid","macrosPer100":{"protein":1,"carbohydrates":1,"fat":1},"micronutrients":{"calcium":1e100}}`,
	}
	for name, body := range invalidBodies {
		t.Run(name, func(t *testing.T) {
			resp := curationRequest(t, app, fiber.MethodPost, "/api/v1/admin/items/validate", body)
			assertStructuredCuration400(t, resp)
		})
	}
	for name, body := range map[string]string{
		"empty classification":           `{"name":"   "}`,
		"control classification":         `{"name":"SECRET\tRAW"}`,
		"unknown classification field":   `{"name":"Fruit","raw":"SECRET"}`,
		"duplicate classification field": `{"name":"Fruit","name":"SECRET"}`,
		"nested array field":             `{"name":"Fruit","raw":[{"key":"value"}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			resp := curationRequest(t, app, fiber.MethodPost, "/api/v1/admin/classifications/validate", body)
			assertStructuredCuration400(t, resp)
		})
	}
	for name, body := range map[string][]byte{
		"empty body":      {},
		"non-object":      []byte(`[]`),
		"trailing object": []byte(`{"name":"Fruit"} {}`),
		"malformed UTF-8": {'{', '"', 'n', 'a', 'm', 'e', '"', ':', '"', 0xff, '"', '}'},
	} {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(fiber.MethodPost, "/api/v1/admin/classifications/validate", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			assertStructuredCuration400(t, resp)
		})
	}

	if providerCalls != 1 || repositoryCalls != 2 {
		t.Fatalf("rejected requests dispatched provider=%d repository=%d", providerCalls, repositoryCalls)
	}
	_, captured := logs.Snapshot()
	encoded, err := json.Marshal(captured)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "SECRET") || strings.Contains(string(encoded), "Cafe drink") || strings.Contains(string(encoded), "product:123") {
		t.Fatalf("raw curation input leaked to logs: %s", encoded)
	}
}

func assertCurationStatus(t *testing.T, app *fiber.App, method string, path string, body string, want int) {
	t.Helper()
	resp := curationRequest(t, app, method, path, body)
	defer resp.Body.Close()
	if resp.StatusCode != want {
		t.Fatalf("%s %s = %d, want %d", method, path, resp.StatusCode, want)
	}
}

func curationRequest(t *testing.T, app *fiber.App, method string, path string, body string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func assertStructuredCuration400(t *testing.T, resp *http.Response) {
	t.Helper()
	defer resp.Body.Close()
	envelope := decodeEnvelope(t, resp.Body)
	if resp.StatusCode != fiber.StatusBadRequest || envelope.Error == nil || envelope.Error.Category != "validation" || envelope.Error.Code != "validation_failed" {
		t.Fatalf("curation response = %d %+v", resp.StatusCode, envelope)
	}
}
