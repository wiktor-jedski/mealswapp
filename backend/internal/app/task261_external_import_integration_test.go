package app

// Implements DESIGN-009 ExternalSearchProxy/DataImporter and DESIGN-012 provider integration verification.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"sync/atomic"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wiktor-jedski/mealswapp/backend/internal/customitem"
	"github.com/wiktor-jedski/mealswapp/backend/internal/dataimporter"
	"github.com/wiktor-jedski/mealswapp/backend/internal/externaldata"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
)

// TestTask261ProviderHTTPImportPostgresFlow verifies IT-ARCH-009-002,
// IT-ARCH-012-001, and IT-ARCH-012-002 through real provider clients,
// ExternalSearchProxy, authenticated Fiber controllers, DataImporter,
// PostgreSQL transaction/audit repositories, and the catalog HTTP consumer.
func TestTask261ProviderHTTPImportPostgresFlow(t *testing.T) {
	var usdaCalls, openFoodFactsCalls atomic.Int32
	usda := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		usdaCalls.Add(1)
		if request.Method != http.MethodGet || request.URL.Path != "/fdc/v1/foods/search" {
			t.Errorf("USDA request=%s %s", request.Method, request.URL.Path)
		}
		query := request.URL.Query()
		if query.Get("query") != "task 261 tofu" || query.Get("pageNumber") != "2" || query.Get("pageSize") != "20" || query.Get("api_key") != "task-261-key" {
			t.Errorf("USDA query=%s", request.URL.RawQuery)
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(response, task261USDAPayload)
	}))
	defer usda.Close()
	openFoodFacts := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		openFoodFactsCalls.Add(1)
		if request.Header.Get("User-Agent") != "Mealswapp task-261 integration (test@example.test)" {
			t.Errorf("OpenFoodFacts User-Agent=%q", request.Header.Get("User-Agent"))
		}
		response.WriteHeader(http.StatusBadRequest)
	}))
	defer openFoodFacts.Close()

	usdaClient, err := externaldata.NewUSDAClient(externaldata.USDAConfig{APIKey: "task-261-key", Endpoint: usda.URL + "/fdc/v1/foods/search", HTTPClient: usda.Client()})
	if err != nil {
		t.Fatal(err)
	}
	openFoodFactsClient, err := externaldata.NewOpenFoodFactsClient(externaldata.OpenFoodFactsConfig{CallerID: "Mealswapp task-261 integration (test@example.test)", Endpoint: openFoodFacts.URL, HTTPClient: openFoodFacts.Client()})
	if err != nil {
		t.Fatal(err)
	}

	db := openDailyDietAPIIntegrationDB(t)
	cfg := liveDailyDietAPIConfig()
	providers := externaldata.ProviderSet{USDA: usdaClient, OpenFoodFacts: openFoodFactsClient}
	server, err := newProduction(cfg, db, nil, observability.JSONSink{Writer: io.Discard}, &providers)
	if err != nil {
		t.Fatalf("compose provider integration app: %v", err)
	}

	email := "task-261-admin@example.test"
	cookies, adminID := registerLiveDailyDietUser(t, server, cfg, email)
	if _, err := db.Exec(context.Background(), `UPDATE users SET role='admin' WHERE id=$1`, adminID); err != nil {
		t.Fatalf("promote integration administrator: %v", err)
	}
	login := liveDailyDietRequest(t, server, fiber.MethodPost, "/api/v1/auth/login", fmt.Sprintf(`{"email":%q,"password":"StrongerPassword1!"}`, email), cookies, "", "")
	loginEnvelope := decodeLiveDailyDietEnvelope(t, login)
	cookies = mergeLiveDailyDietCookies(cookies, login.Cookies())
	login.Body.Close()
	if login.StatusCode != fiber.StatusOK || loginEnvelope.Data["role"] != "admin" {
		t.Fatalf("admin login status=%d body=%+v", login.StatusCode, loginEnvelope)
	}
	csrfToken, cookies := fetchLiveDailyDietCSRF(t, server, cookies)

	searchResponse := liveDailyDietRequest(t, server, fiber.MethodGet, "/api/v1/admin/external-search?"+url.Values{"query": {"task 261 tofu"}, "provider": {"all"}, "page": {"2"}}.Encode(), "", cookies, "", "")
	searchEnvelope := decodeLiveDailyDietEnvelope(t, searchResponse)
	searchResponse.Body.Close()
	if searchResponse.StatusCode != fiber.StatusOK {
		t.Fatalf("external search status=%d body=%+v", searchResponse.StatusCode, searchEnvelope)
	}
	var externalResult externaldata.ExternalSearchResponse
	decodeTask261Data(t, searchEnvelope.Data, &externalResult)
	if usdaCalls.Load() != 1 || openFoodFactsCalls.Load() != 1 || len(externalResult.Candidates) != 1 || externalResult.Page != 2 {
		t.Fatalf("provider flow calls=(%d,%d) response=%+v", usdaCalls.Load(), openFoodFactsCalls.Load(), externalResult)
	}
	if !slices.ContainsFunc(externalResult.Warnings, func(warning externaldata.ExternalDataWarning) bool {
		return warning.Provider == "openfoodfacts" && warning.Code == externaldata.WarningUnavailable
	}) {
		t.Fatalf("partial-provider warning absent: %+v", externalResult.Warnings)
	}
	assertTask261PersistenceCounts(t, db, 0, 0, 0)

	candidate := externalResult.Candidates[0]
	request := dataimporter.Request{SourceProvider: candidate.Provider, ExternalID: candidate.ExternalID, Request: customitem.Request{
		Name: candidate.Name + " curated", PhysicalState: candidate.PhysicalState, MacrosPer100: candidate.MacrosPer100,
		Micros: candidate.Micronutrients, FoodCategoryIDs: []uuid.UUID{}, CulinaryRoleIDs: []uuid.UUID{},
	}}
	requestBody, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}

	importResponse := liveDailyDietRequest(t, server, fiber.MethodPost, "/api/v1/admin/imports", string(requestBody), cookies, "task-261-provider-import", csrfToken)
	importEnvelope := decodeLiveDailyDietEnvelope(t, importResponse)
	importResponse.Body.Close()
	if importResponse.StatusCode != fiber.StatusCreated {
		t.Fatalf("curated import status=%d body=%+v", importResponse.StatusCode, importEnvelope)
	}
	foodItemID := liveUUIDFromData(t, importEnvelope.Data, "foodItemId")
	assertTask261PersistenceCounts(t, db, 1, 1, 1)

	catalogBody := fmt.Sprintf(`{"query":%q,"mode":"catalog","page":1}`, request.Name)
	catalogResponse := liveDailyDietRequest(t, server, fiber.MethodPost, "/api/v1/search", catalogBody, nil, "", "")
	catalogEnvelope := decodeLiveDailyDietEnvelope(t, catalogResponse)
	catalogResponse.Body.Close()
	if catalogResponse.StatusCode != fiber.StatusOK {
		t.Fatalf("catalog status=%d body=%+v", catalogResponse.StatusCode, catalogEnvelope)
	}
	items, ok := catalogEnvelope.Data["items"].([]any)
	if !ok || !slices.ContainsFunc(items, func(value any) bool {
		item, itemOK := value.(map[string]any)
		return itemOK && item["id"] == foodItemID.String() && item["name"] == request.Name
	}) {
		t.Fatalf("persisted item not visible through catalog HTTP: %+v", catalogEnvelope.Data)
	}
}

func decodeTask261Data(t *testing.T, data map[string]any, target any) {
	t.Helper()
	payload, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(payload, target); err != nil {
		t.Fatalf("decode integration response: %v", err)
	}
}

func assertTask261PersistenceCounts(t *testing.T, db *pgxpool.Pool, foods, imports, audits int) {
	t.Helper()
	var gotFoods, gotImports, gotAudits int
	if err := db.QueryRow(context.Background(), `SELECT (SELECT count(*) FROM food_items), (SELECT count(*) FROM curated_imports), (SELECT count(*) FROM admin_audit_entries WHERE action='import_food')`).Scan(&gotFoods, &gotImports, &gotAudits); err != nil {
		t.Fatal(err)
	}
	if gotFoods != foods || gotImports != imports || gotAudits != audits {
		t.Fatalf("persistence counts foods=%d imports=%d audits=%d want=(%d,%d,%d)", gotFoods, gotImports, gotAudits, foods, imports, audits)
	}
}

const task261USDAPayload = `{
  "totalHits": 1,
  "currentPage": 2,
  "totalPages": 2,
  "foods": [{
    "fdcId": 261002,
    "description": "Task 261 provider tofu",
    "servingSize": 100,
    "servingSizeUnit": "g",
    "foodNutrients": [
      {"nutrientName":"Protein","unitName":"G","value":18},
      {"nutrientName":"Carbohydrate, by difference","unitName":"G","value":4},
      {"nutrientName":"Total lipid (fat)","unitName":"G","value":8}
    ],
    "foodMeasures": []
  }]
}`
