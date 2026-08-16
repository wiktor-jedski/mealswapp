package curation

// Implements DESIGN-009 curation requests and DESIGN-013 InputNormalizer verification.

import (
	"context"
	"encoding/json"
	"math"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

func TestInputNormalizerNormalizesTypedCurationRequests(t *testing.T) {
	logs := &observability.MemorySink{}
	normalizer := NewInputNormalizer(logs)
	search, err := normalizer.NormalizeExternalSearch(context.Background(), ExternalSearchRequest{Query: "  Cafe\u0301  ", Provider: "OpenFoodFacts", Page: 2})
	if err != nil || search.Query != "Café" || search.Provider != "openfoodfacts" || search.Page != 2 {
		t.Fatalf("external search = %+v, %v", search, err)
	}
	item, err := normalizer.NormalizeItem(context.Background(), ItemRequest{
		Name: "  Oat   milk ", PhysicalState: repository.PhysicalStateLiquid,
		ImageURL: " https://images.example.com/oat.jpg ", ServingUnit: "fluid ounces", ServingQuantity: 8,
		SourceProvider: " USDA ", ExternalID: " fdc:123 ", ProviderText: "  Oat   beverage ",
		MacrosPer100:   repository.MacroValues{Protein: 1, Carbohydrates: 7, Fat: 2},
		Micronutrients: repository.MicroValues{"calcium": 120},
	})
	if err != nil || item.Name != "Oat milk" || item.ServingUnit != "fl_oz" || item.SourceProvider != "usda" || item.ExternalID != "fdc:123" || item.ProviderText != "Oat beverage" {
		t.Fatalf("item = %+v, %v", item, err)
	}
	parentID := uuid.New()
	classification, err := normalizer.NormalizeClassification(context.Background(), ClassificationRequest{Name: "  Cafe\u0301   foods ", ParentID: &parentID})
	if err != nil || classification.Name != "Café foods" || classification.ParentID == nil || *classification.ParentID != parentID {
		t.Fatalf("classification = %+v, %v", classification, err)
	}
	_, captured := logs.Snapshot()
	if len(captured) == 0 {
		t.Fatal("normalization metadata was not logged")
	}
	encoded, err := json.Marshal(captured)
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{"Oat milk", "fdc:123", "Oat beverage", "Café"} {
		if strings.Contains(string(encoded), raw) {
			t.Fatalf("raw normalized value %q leaked in %s", raw, encoded)
		}
	}
}

func TestInputNormalizerRejectsMalformedCurationFieldsWithoutRawLogs(t *testing.T) {
	valid := func() ItemRequest {
		return ItemRequest{Name: "Item", PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{Protein: 1}}
	}
	unsafeImage := valid()
	unsafeImage.ImageURL = "http://127.0.0.1/a.jpg"
	badUnit := valid()
	badUnit.ServingUnit, badUnit.ServingQuantity = "cup", 1
	badProvider := valid()
	badProvider.SourceProvider, badProvider.ExternalID = "other", "1"
	badProviderID := valid()
	badProviderID.SourceProvider, badProviderID.ExternalID = "usda", "bad id"
	badProviderText := valid()
	badProviderText.ProviderText = "bad\x00text"
	tests := []struct {
		name string
		req  ItemRequest
	}{
		{name: "unsafe image", req: unsafeImage},
		{name: "bad serving unit", req: badUnit},
		{name: "unsupported item provider", req: badProvider},
		{name: "bad provider id", req: badProviderID},
		{name: "bad provider text", req: badProviderText},
		{name: "invalid physical state", req: ItemRequest{Name: "Item", PhysicalState: "gas"}},
		{name: "negative macro", req: ItemRequest{Name: "Item", PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{Protein: -1}}},
		{name: "nonfinite macro", req: ItemRequest{Name: "Item", PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{Protein: math.NaN()}}},
		{name: "solid macro total", req: ItemRequest{Name: "Item", PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{Protein: 40, Carbohydrates: 40, Fat: 21}}},
		{name: "negative serving", req: ItemRequest{Name: "Item", PhysicalState: repository.PhysicalStateSolid, ServingUnit: "g", ServingQuantity: -1}},
		{name: "unit without quantity", req: ItemRequest{Name: "Item", PhysicalState: repository.PhysicalStateSolid, ServingUnit: "g"}},
		{name: "provider without id", req: ItemRequest{Name: "Item", PhysicalState: repository.PhysicalStateSolid, SourceProvider: "usda"}},
		{name: "negative micronutrient", req: ItemRequest{Name: "Item", PhysicalState: repository.PhysicalStateSolid, Micronutrients: repository.MicroValues{"calcium": -1}}},
		{name: "malformed micronutrient key", req: ItemRequest{Name: "Item", PhysicalState: repository.PhysicalStateSolid, Micronutrients: repository.MicroValues{"Calcium mg": 1}}},
		{name: "control text", req: ItemRequest{Name: "SECRET\nRAW", PhysicalState: repository.PhysicalStateSolid}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			logs := &observability.MemorySink{}
			if _, err := NewInputNormalizer(logs).NormalizeItem(context.Background(), tc.req); err == nil {
				t.Fatalf("accepted %+v", tc.req)
			}
			_, captured := logs.Snapshot()
			encoded, err := json.Marshal(captured)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(encoded), "SECRET") || strings.Contains(string(encoded), "RAW") || strings.Contains(string(encoded), "Calcium mg") {
				t.Fatalf("rejected raw value leaked in %s", encoded)
			}
			if len(captured) == 0 || captured[len(captured)-1].Fields["outcome"] != "rejected" {
				t.Fatalf("missing rejection metadata: %+v", captured)
			}
		})
	}
}

func TestInputNormalizerOptionalAndMetadataBranches(t *testing.T) {
	logs := &observability.MemorySink{}
	normalizer := NewInputNormalizer(logs)
	normalizer.RecordRejection(context.Background(), RejectionFieldItemBody)
	item, err := normalizer.NormalizeItem(context.Background(), ItemRequest{
		Name: "Item", PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{Protein: 1},
	})
	if err != nil || item.Micronutrients == nil {
		t.Fatalf("optional micronutrients = %+v, %v", item, err)
	}
	if _, err := normalizer.NormalizeClassification(context.Background(), ClassificationRequest{Name: "bad\x00name"}); err == nil {
		t.Fatal("invalid classification accepted")
	}
	_, captured := logs.Snapshot()
	if len(captured) < 2 || captured[0].Fields["field"] != "curation_item_body" || captured[0].Fields["outcome"] != "rejected" {
		t.Fatalf("metadata logs = %+v", captured)
	}
}

func TestRecordRejectionDropsUnknownMetadata(t *testing.T) {
	logs := &observability.MemorySink{}
	normalizer := NewInputNormalizer(logs)
	normalizer.RecordRejection(context.Background(), RejectionField("SECRET\nRAW"))
	normalizer.log(context.Background(), "SECRET\nRAW", "rejected", false, 0)
	normalizer.log(context.Background(), "curation_item_body", "SECRET\nRAW", false, 0)
	_, captured := logs.Snapshot()
	if len(captured) != 0 {
		t.Fatalf("unknown metadata was logged: %+v", captured)
	}
}

func TestInputNormalizerRejectsNumericValuesAboveDocumentedBounds(t *testing.T) {
	valid := func() ItemRequest {
		return ItemRequest{Name: "Item", PhysicalState: repository.PhysicalStateLiquid, MacrosPer100: repository.MacroValues{Protein: 1}}
	}
	serving := valid()
	serving.ServingUnit = "ml"
	serving.ServingQuantity = math.Nextafter(MaxCurationServingQuantity, math.Inf(1))
	micro := valid()
	micro.Micronutrients = repository.MicroValues{"calcium": math.Nextafter(MaxCurationNutritionValue, math.Inf(1))}
	requests := map[string]ItemRequest{"serving": serving, "micronutrient": micro}
	for name, macros := range map[string]repository.MacroValues{
		"protein":       {Protein: math.Nextafter(MaxCurationNutritionValue, math.Inf(1))},
		"carbohydrates": {Carbohydrates: math.Nextafter(MaxCurationNutritionValue, math.Inf(1))},
		"fat":           {Fat: math.Nextafter(MaxCurationNutritionValue, math.Inf(1))},
	} {
		req := valid()
		req.MacrosPer100 = macros
		requests[name] = req
	}
	for name, req := range requests {
		t.Run(name, func(t *testing.T) {
			if _, err := NewInputNormalizer(nil).NormalizeItem(context.Background(), req); err == nil {
				t.Fatalf("accepted out-of-range request: %+v", req)
			}
		})
	}

	atMaximum := valid()
	atMaximum.MacrosPer100.Protein = MaxCurationNutritionValue
	atMaximum.ServingUnit = "ml"
	atMaximum.ServingQuantity = MaxCurationServingQuantity
	atMaximum.Micronutrients = repository.MicroValues{"calcium": MaxCurationNutritionValue}
	normalized, err := NewInputNormalizer(nil).NormalizeItem(context.Background(), atMaximum)
	if err != nil {
		t.Fatalf("documented maxima rejected: %v", err)
	}
	scaled := repository.ScaleMacros(normalized.MacrosPer100, normalized.ServingQuantity, 100)
	if math.IsInf(scaled.Protein, 0) || math.IsNaN(scaled.Protein) {
		t.Fatalf("bounded downstream scaling overflowed: %+v", scaled)
	}
}

func TestValidateMicronutrientsBoundaries(t *testing.T) {
	tooMany := repository.MicroValues{}
	for index := range 201 {
		tooMany["nutrient_"+strings.Repeat("x", index/10)+string(rune('a'+index%10))] = 1
	}
	for name, values := range map[string]repository.MicroValues{
		"too many":        tooMany,
		"empty key":       {"": 1},
		"long key":        {strings.Repeat("a", 121): 1},
		"nan":             {"calcium": math.NaN()},
		"infinite":        {"calcium": math.Inf(1)},
		"leading marker":  {"_calcium": 1},
		"trailing marker": {"calcium_": 1},
	} {
		t.Run(name, func(t *testing.T) {
			if err := validateMicronutrients(values); err == nil {
				t.Fatalf("accepted %#v", values)
			}
		})
	}
}

func TestExternalSearchRejectsBeforeProviderUse(t *testing.T) {
	normalizer := NewInputNormalizer(nil)
	for _, req := range []ExternalSearchRequest{
		{Query: "", Provider: "usda", Page: 1},
		{Query: "apple\x00", Provider: "usda", Page: 1},
		{Query: "apple", Provider: "other", Page: 1},
		{Query: "apple", Provider: "usda", Page: 0},
	} {
		if _, err := normalizer.NormalizeExternalSearch(context.Background(), req); err == nil {
			t.Fatalf("accepted %+v", req)
		}
	}
}
