package httpapi

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// Implements DESIGN-002 SearchController food-object detail HTTP verification.

type fakeFoodObjectLookup struct {
	item    repository.FoodItemEntity
	err     error
	id      uuid.UUID
	context repository.RepositoryContext
	calls   int
}

func (f *fakeFoodObjectLookup) GetByID(_ context.Context, id uuid.UUID, rc repository.RepositoryContext) (repository.FoodItemEntity, error) {
	f.calls++
	f.id = id
	f.context = rc
	return f.item, f.err
}

func TestFoodObjectControllerRoutesExposeDetailPolicyMetadata(t *testing.T) {
	controller := NewFoodObjectController(&fakeFoodObjectLookup{})
	routes := controller.Routes()
	if len(routes) != 1 {
		t.Fatalf("routes = %+v", routes)
	}
	route := routes[0]
	if route.Method != fiber.MethodGet || route.Path != "/food-objects/:id" || !route.OptionalAuth || route.RequiresCSRF || route.ExemptCSRF || route.RateLimit == nil {
		t.Fatalf("food object route policy = %+v", route)
	}
}

func TestFoodObjectControllerReturnsFoodObjectEnvelope(t *testing.T) {
	// Implements DESIGN-002 SearchController food-object detail hydration contract verification.
	itemID := uuid.MustParse("71000000-0000-4000-8000-000000000001")
	categoryID := uuid.MustParse("71000000-0000-4000-8000-000000000002")
	roleID := uuid.MustParse("71000000-0000-4000-8000-000000000003")
	lookup := &fakeFoodObjectLookup{item: repository.FoodItemEntity{
		ID:             itemID,
		Name:           "Apple",
		PhysicalState:  repository.PhysicalStateSolid,
		ImageURL:       "/assets/apple.webp",
		MacrosPer100:   repository.MacroValues{Protein: 0.5, Carbohydrates: 14, Fat: 0.3},
		FoodCategories: []repository.ClassificationEntity{{ID: categoryID, Name: "Fruits", Kind: repository.ClassificationKindFoodCategory}},
		CulinaryRoles:  []repository.ClassificationEntity{{ID: roleID, Name: "Snack", Kind: repository.ClassificationKindCulinaryRole}},
	}}
	app := mustNewRouter(t, Dependencies{Config: testConfig(), Routes: NewFoodObjectController(lookup).Routes()})

	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/food-objects/"+itemID.String(), nil))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	envelope := decodeEnvelope(t, resp.Body)

	if resp.StatusCode != fiber.StatusOK || lookup.calls != 1 || lookup.id != itemID {
		t.Fatalf("response=%d calls=%d lookupID=%s envelope=%+v", resp.StatusCode, lookup.calls, lookup.id, envelope)
	}
	if envelope.Status != "ok" || envelope.RequestID == "" {
		t.Fatalf("envelope metadata = %+v", envelope)
	}
	if envelope.Data["id"] != itemID.String() || envelope.Data["name"] != "Apple" || envelope.Data["macroBasis"] != "100g" {
		t.Fatalf("food object envelope data = %+v", envelope.Data)
	}
	macros := envelope.Data["macros"].(map[string]any)
	if macros["protein"].(float64) != 0.5 || macros["carbohydrates"].(float64) != 14 || macros["fat"].(float64) != 0.3 {
		t.Fatalf("macros = %+v", macros)
	}
	classifications := envelope.Data["classifications"].([]any)
	if len(classifications) != 2 || envelope.Data["primaryFoodCategory"].(map[string]any)["name"] != "Fruits" {
		t.Fatalf("classifications = %+v primary=%+v", classifications, envelope.Data["primaryFoodCategory"])
	}
}

func TestFoodObjectControllerRejectsInvalidUUID(t *testing.T) {
	lookup := &fakeFoodObjectLookup{}
	app := mustNewRouter(t, Dependencies{Config: testConfig(), Routes: NewFoodObjectController(lookup).Routes()})

	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/food-objects/not-a-uuid", nil))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest || lookup.calls != 0 {
		t.Fatalf("invalid uuid response=%d calls=%d", resp.StatusCode, lookup.calls)
	}
}

func TestFoodObjectControllerMapsMissingFoodObjectTo404(t *testing.T) {
	itemID := uuid.MustParse("71000000-0000-4000-8000-000000000099")
	lookup := &fakeFoodObjectLookup{err: repository.NewError(repository.ErrorKindNotFound, "food object not found", nil)}
	app := mustNewRouter(t, Dependencies{Config: testConfig(), Routes: NewFoodObjectController(lookup).Routes()})

	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/food-objects/"+itemID.String(), nil))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.StatusCode != fiber.StatusNotFound || lookup.calls != 1 {
		t.Fatalf("not found response=%d calls=%d", resp.StatusCode, lookup.calls)
	}
}
