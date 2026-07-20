package httpapi

// Implements DESIGN-010 RequestValidator and DESIGN-002 SearchController verification.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
	"github.com/wiktor-jedski/mealswapp/backend/internal/search"
)

func TestSearchRequestValidationStopsBeforeHandler(t *testing.T) {
	calls := 0
	app := mustNewRouter(t, Dependencies{Config: testConfig(), Routes: []RouteDefinition{{
		Method:     fiber.MethodPost,
		Path:       "/search",
		ExemptCSRF: true,
		Validate:   ValidateJSON(ValidateSearchRequestBody),
		Handler: func(ctx *fiber.Ctx) error {
			calls++
			return ctx.JSON(Envelope{Status: "ok", RequestID: requestID(ctx)})
		},
	}}})
	validBody := `{"query":"  Fresh   TOMATO  ","mode":"catalog","page":1,"filters":[{"filterId":"vegetable","kind":" Food_Category ","include":true}]}`
	resp := postSearchValidation(t, app, validBody)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK || calls != 1 {
		t.Fatalf("valid search = %d calls=%d", resp.StatusCode, calls)
	}
	validSubstitutionBody := `{"query":"","mode":"substitution","page":1,"substitutionInputs":[{"foodObjectId":"2d4a5f20-c55f-4ba7-9751-779e682f7063","quantity":100,"unit":"ml"}]}`
	resp = postSearchValidation(t, app, validSubstitutionBody)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK || calls != 2 {
		t.Fatalf("valid substitution = %d calls=%d", resp.StatusCode, calls)
	}
	validMealSubstitutionBody := `{"query":"","mode":"substitution","page":1,"substitutionInputs":[{"foodObjectId":"2d4a5f20-c55f-4ba7-9751-779e682f7063","foodObjectType":"meal","quantity":100,"unit":"g"}]}`
	resp = postSearchValidation(t, app, validMealSubstitutionBody)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK || calls != 3 {
		t.Fatalf("valid meal substitution = %d calls=%d", resp.StatusCode, calls)
	}
	tooManyFilters := strings.TrimSuffix(strings.Repeat(`{"filterId":"dairy","kind":"allergen","include":true},`, 21), ",")
	tooManyInputs := strings.TrimSuffix(strings.Repeat(`{"foodObjectId":"2d4a5f20-c55f-4ba7-9751-779e682f7063","quantity":12.5,"unit":"g"},`, 21), ",")

	for name, body := range map[string]string{
		"empty query":                  `{"query":"   ","mode":"catalog","page":1}`,
		"maximum query length":         `{"query":"` + strings.Repeat("a", 201) + `","mode":"catalog","page":1}`,
		"invalid mode":                 `{"query":"tomato","mode":"meal_plan","page":1}`,
		"zero page":                    `{"query":"tomato","mode":"catalog","page":0}`,
		"string page":                  `{"query":"tomato","mode":"catalog","page":"1"}`,
		"fractional page":              `{"query":"tomato","mode":"catalog","page":1.5}`,
		"unsupported filter kind":      `{"query":"tomato","mode":"catalog","page":1,"filters":[{"filterId":"x","kind":"brand","include":true}]}`,
		"old physical state kind":      `{"query":"tomato","mode":"catalog","page":1,"filters":[{"filterId":"solid","kind":"food_object_type","include":true}]}`,
		"string substitution quantity": `{"query":"tomato","mode":"substitution","page":1,"substitutionInputs":[{"foodObjectId":"2d4a5f20-c55f-4ba7-9751-779e682f7063","quantity":"12.5","unit":"g"}]}`,
		"invalid food object type":     `{"query":"tomato","mode":"substitution","page":1,"substitutionInputs":[{"foodObjectId":"2d4a5f20-c55f-4ba7-9751-779e682f7063","foodObjectType":"recipe","quantity":12.5,"unit":"g"}]}`,
		"aliased substitution unit":    `{"query":"tomato","mode":"substitution","page":1,"substitutionInputs":[{"foodObjectId":"2d4a5f20-c55f-4ba7-9751-779e682f7063","quantity":12.5,"unit":"gram"}]}`,
		"fluid ounce alias":            `{"query":"tomato","mode":"substitution","page":1,"substitutionInputs":[{"foodObjectId":"2d4a5f20-c55f-4ba7-9751-779e682f7063","quantity":12.5,"unit":"fluid_ounce"}]}`,
		"too many filters":             `{"query":"tomato","mode":"catalog","page":1,"filters":[` + tooManyFilters + `]}`,
		"too many substitution inputs": `{"query":"tomato","mode":"substitution","page":1,"substitutionInputs":[` + tooManyInputs + `]}`,
		"invalid daily diet id":        `{"query":"tomato","mode":"daily_diet_alternative","page":1,"dailyDietId":"not-a-uuid"}`,
		"catalog with substitution":    `{"query":"tomato","mode":"catalog","page":1,"substitutionInputs":[{"foodObjectId":"2d4a5f20-c55f-4ba7-9751-779e682f7063","quantity":12.5,"unit":"g"}]}`,
		"catalog with daily diet id":   `{"query":"tomato","mode":"catalog","page":1,"dailyDietId":"2d4a5f20-c55f-4ba7-9751-779e682f7063"}`,
		"substitution without inputs":  `{"query":"tomato","mode":"substitution","page":1}`,
		"substitution with daily id":   `{"query":"tomato","mode":"substitution","page":1,"dailyDietId":"2d4a5f20-c55f-4ba7-9751-779e682f7063","substitutionInputs":[{"foodObjectId":"2d4a5f20-c55f-4ba7-9751-779e682f7063","quantity":12.5,"unit":"g"}]}`,
		"daily diet with inputs":       `{"query":"tomato","mode":"daily_diet_alternative","page":1,"dailyDietId":"2d4a5f20-c55f-4ba7-9751-779e682f7063","substitutionInputs":[{"foodObjectId":"2d4a5f20-c55f-4ba7-9751-779e682f7063","quantity":12.5,"unit":"g"}]}`,
	} {
		resp := postSearchValidation(t, app, body)
		envelope := decodeEnvelope(t, resp.Body)
		resp.Body.Close()
		if resp.StatusCode != fiber.StatusBadRequest || envelope.Error == nil || envelope.Error.Category != "validation" || envelope.Error.Code != "validation_failed" {
			t.Fatalf("%s response = %d %+v", name, resp.StatusCode, envelope)
		}
	}
	if calls != 3 {
		t.Fatalf("invalid search dispatched handler calls=%d", calls)
	}
}

func TestAutocompleteValidationStopsBeforeHandler(t *testing.T) {
	calls := 0
	app := mustNewRouter(t, Dependencies{Config: testConfig(), Routes: []RouteDefinition{{
		Method:   fiber.MethodGet,
		Path:     "/search/autocomplete",
		Validate: ValidateQuery(ValidateAutocompleteQueryParams),
		Handler: func(ctx *fiber.Ctx) error {
			calls++
			return ctx.SendStatus(fiber.StatusNoContent)
		},
	}}})
	for path, want := range map[string]int{
		"/api/v1/search/autocomplete?query=%20Lentils%20&page=1":        fiber.StatusNoContent,
		"/api/v1/search/autocomplete?q=":                                fiber.StatusBadRequest,
		"/api/v1/search/autocomplete?q=Apple":                           fiber.StatusBadRequest,
		"/api/v1/search/autocomplete?query=" + strings.Repeat("a", 121): fiber.StatusBadRequest,
		"/api/v1/search/autocomplete?query=lentils&page=-1":             fiber.StatusBadRequest,
	} {
		resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, path, nil))
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != want {
			t.Fatalf("GET %s = %d, want %d", path, resp.StatusCode, want)
		}
	}
	if calls != 1 {
		t.Fatalf("autocomplete handler calls=%d", calls)
	}
}

func TestRejectedSearchValidationDoesNotLogRawInput(t *testing.T) {
	raw := "SECRET-RAW-SEARCH-VALUE"
	logs := &observability.MemorySink{}
	audit := &auditSink{}
	app := mustNewRouter(t, Dependencies{Config: testConfig(), Logs: logs, Metrics: logs, Audit: audit, Routes: []RouteDefinition{{
		Method:     fiber.MethodPost,
		Path:       "/search",
		ExemptCSRF: true,
		Validate:   ValidateJSON(ValidateSearchRequestBody),
		Handler:    func(ctx *fiber.Ctx) error { return ctx.SendStatus(fiber.StatusNoContent) },
	}}})
	resp := postSearchValidation(t, app, `{"query":"`+raw+`","mode":"brand_mode","page":1}`)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("raw rejected status = %d", resp.StatusCode)
	}
	encodedLogs, err := json.Marshal(logs.Logs)
	if err != nil {
		t.Fatal(err)
	}
	encodedAudit, err := json.Marshal(audit.entries)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encodedLogs), raw) || strings.Contains(string(encodedAudit), raw) {
		t.Fatalf("raw rejected input leaked logs=%s audit=%s", encodedLogs, encodedAudit)
	}
}

func TestDailyDietAlternativeSearchBoundaryReturns422WithoutSideEffects(t *testing.T) {
	workerSideEffects := 0
	app := mustNewRouter(t, Dependencies{Config: testConfig(), Routes: []RouteDefinition{{
		Method:     fiber.MethodPost,
		Path:       "/search",
		ExemptCSRF: true,
		Validate:   ValidateJSON(ValidateSearchRequestBody),
		Handler: func(ctx *fiber.Ctx) error {
			body := map[string]any{}
			if err := ctx.BodyParser(&body); err != nil {
				return err
			}
			req, err := ParseValidatedSearchRequestBody(body)
			if err != nil {
				return err
			}
			prepared, err := search.PrepareSearchRequest(req, search.DailyDietDataUnavailable)
			if err != nil {
				return AppError{HTTPStatus: fiber.StatusBadRequest, Category: "validation", Code: "validation_failed", Message: "request validation failed", Cause: err}
			}
			if prepared.Rejection != nil {
				return ctx.Status(fiber.StatusUnprocessableEntity).JSON(Envelope{Status: "rejected", RequestID: requestID(ctx), Data: map[string]any{"rejection": prepared.Rejection}})
			}
			workerSideEffects++
			return ctx.JSON(Envelope{Status: "ok", RequestID: requestID(ctx)})
		},
	}}})

	resp := postSearchValidation(t, app, `{"query":"lentil","mode":"daily_diet_alternative","page":1,"dailyDietId":"61e0cae4-0f45-4854-8ac5-b228214cdd1d"}`)
	body := decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusUnprocessableEntity || body.Status != "rejected" {
		t.Fatalf("daily diet unavailable response = %d %+v", resp.StatusCode, body)
	}
	rejection, ok := body.Data["rejection"].(map[string]any)
	if !ok || rejection["code"] != "phase_07_saved_diet_unavailable" || rejection["field"] != "dailyDietId" {
		t.Fatalf("rejection payload = %+v", body.Data["rejection"])
	}
	if workerSideEffects != 0 {
		t.Fatalf("worker side effects = %d", workerSideEffects)
	}

	resp = postSearchValidation(t, app, `{"query":"lentil","mode":"daily_diet_alternative","page":1}`)
	body = decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusBadRequest || body.Error == nil || body.Error.Code != "validation_failed" {
		t.Fatalf("missing dailyDietId response = %d %+v", resp.StatusCode, body)
	}
	if workerSideEffects != 0 {
		t.Fatalf("worker side effects after missing id = %d", workerSideEffects)
	}

	resp = postSearchValidation(t, app, `{"query":"lentil","mode":"daily_diet","page":1}`)
	body = decodeEnvelope(t, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusBadRequest || body.Error == nil || body.Error.Code != "validation_failed" {
		t.Fatalf("missing dailyDietId daily diet response = %d %+v", resp.StatusCode, body)
	}
	if workerSideEffects != 0 {
		t.Fatalf("worker side effects after daily diet missing id = %d", workerSideEffects)
	}
}

func TestParseValidatedSearchRequestBodyPreservesDailyDietAndFilters(t *testing.T) {
	body := map[string]any{
		"query":       "apple",
		"mode":        "daily_diet",
		"page":        float64(2),
		"dailyDietId": "61e0cae4-0f45-4854-8ac5-b228214cdd1d",
		"filters": []any{
			map[string]any{"filterId": "solid", "kind": "physical_state", "include": true},
		},
	}
	req, err := ParseValidatedSearchRequestBody(body)
	if err != nil {
		t.Fatal(err)
	}
	if req.Mode != search.SearchModeDailyDiet || req.Page != 2 || req.DailyDietID == nil {
		t.Fatalf("request core fields = %+v", req)
	}
	if len(req.Filters) != 1 || req.Filters[0].Kind != search.SearchFilterKindPhysicalState || !req.Filters[0].Include {
		t.Fatalf("filters = %+v", req.Filters)
	}
	if len(req.SubstitutionInputs) != 0 {
		t.Fatalf("substitution inputs = %+v", req.SubstitutionInputs)
	}
}

func TestParseValidatedSearchRequestBodyRejectsMissingOrMistypedFields(t *testing.T) {
	for name, body := range map[string]map[string]any{
		"missing query": {"mode": "catalog", "page": float64(1)},
		"mistyped query": {
			"query": 123,
			"mode":  "catalog",
			"page":  float64(1),
		},
		"missing mode": {"query": "apple", "page": float64(1)},
		"mistyped mode": {
			"query": "apple",
			"mode":  123,
			"page":  float64(1),
		},
		"missing page": {"query": "apple", "mode": "catalog"},
		"mistyped page": {
			"query": "apple",
			"mode":  "catalog",
			"page":  map[string]any{},
		},
	} {
		if _, err := ParseValidatedSearchRequestBody(body); err == nil {
			t.Fatalf("%s accepted", name)
		}
	}
}

func postSearchValidation(t *testing.T, app *fiber.App, body string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(fiber.MethodPost, "/api/v1/search", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}
