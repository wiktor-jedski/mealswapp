package catalogexport

// Implements DESIGN-009 AdminController deterministic bounded catalog export verification.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

type exportStore struct {
	items          []repository.CatalogExportItem
	includeDeleted bool
	err            error
}

func (s *exportStore) Stream(ctx context.Context, includeDeleted bool, consume repository.CatalogExportRowConsumer) (int, error) {
	s.includeDeleted = includeDeleted
	count := 0
	for _, item := range s.items {
		if err := ctx.Err(); err != nil {
			return count, err
		}
		if err := consume(item); err != nil {
			return count, err
		}
		count++
	}
	return count, s.err
}

func TestServiceWritesByteIdenticalCompleteCatalog(t *testing.T) {
	id := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	categoryID := uuid.MustParse("00000000-0000-4000-8000-000000000003")
	roleID := uuid.MustParse("00000000-0000-4000-8000-000000000004")
	curatedID := uuid.MustParse("00000000-0000-4000-8000-000000000005")
	weight, image, alt := "125.5000", "https://example.test/żółć.png", "Żółty ser"
	source, external := "usda", "сыр-42"
	created := time.Date(2026, 7, 1, 2, 3, 4, 500, time.FixedZone("fixture", 3600))
	store := &exportStore{items: []repository.CatalogExportItem{{
		ID: id, Name: "Crème brûlée", PhysicalState: repository.PhysicalStateSolid, PrepTimeMinutes: 7,
		AverageUnitWeightGrams: &weight, ProteinPer100: "10.0000", CarbohydratesPer100: "20.2500", FatPer100: "3.5000",
		Micros:         map[string]json.Number{"VitaminC": "4.5000", "Iron": "1.2500"},
		FoodCategories: []repository.CatalogExportClassification{{ID: categoryID, Name: "Dessert"}},
		CulinaryRoles:  []repository.CatalogExportClassification{{ID: roleID, Name: "Main"}},
		AllergenKeys:   []string{"dairy", "egg"}, ImageURL: &image, ImageAlt: &alt,
		SourceProvider: &source, ExternalID: &external, CreatedAt: created, UpdatedAt: created,
		CuratedSources: []repository.CatalogExportCuratedSource{{ID: curatedID, Provider: source, ExternalID: external, Status: "imported"}},
	}}}
	service := NewService(store)
	var first, second bytes.Buffer
	count, err := service.Write(context.Background(), &first, true)
	if err != nil || count != 1 || !store.includeDeleted {
		t.Fatalf("Write() = count %d err %v includeDeleted %t", count, err, store.includeDeleted)
	}
	if _, err := service.Write(context.Background(), &second, true); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first.Bytes(), second.Bytes()) {
		t.Fatal("unchanged exports are not byte-identical")
	}
	text := first.String()
	for _, expected := range []string{
		`"schema":"mealswapp.global-catalog.v1"`,
		`"idempotencyKey":"global-catalog:` + id.String() + `"`,
		`"name":"Crème brûlée"`, `"averageUnitWeightGrams":125.5`,
		`"averageServingVolumeMilliliters":null`, `"imageAlt":"Żółty ser"`,
		`"sourceProvider":"usda"`, `"externalId":"сыр-42"`,
		`"foodCategoryNames":["Dessert"]`, `"allergenKeys":["dairy","egg"]`,
		`"createdAt":"2026-07-01T01:03:04.0000005Z"`, `"deletedAt":null`,
	} {
		if !strings.Contains(text, expected) {
			t.Errorf("export missing %s: %s", expected, text)
		}
	}
	if strings.Contains(text, "generatedAt") || strings.Contains(text, "ownerId") || strings.Contains(text, "email") {
		t.Fatalf("export contains volatile/private metadata: %s", text)
	}
}

func TestServiceEmptyLargeCancellationAndErrorPropagation(t *testing.T) {
	service := NewService(&exportStore{})
	var output bytes.Buffer
	if count, err := service.Write(context.Background(), &output, false); err != nil || count != 0 || output.String() != `{"schema":"mealswapp.global-catalog.v1","items":[]}`+"\n" {
		t.Fatalf("empty Write() = %d %v %q", count, err, output.String())
	}

	items := make([]repository.CatalogExportItem, 2000)
	for index := range items {
		items[index] = repository.CatalogExportItem{
			ID: uuid.New(), Name: "item", PhysicalState: repository.PhysicalStateSolid,
			ProteinPer100: "0", CarbohydratesPer100: "0", FatPer100: "0",
			CreatedAt: time.Unix(0, 0), UpdatedAt: time.Unix(0, 0),
		}
	}
	if count, err := NewService(&exportStore{items: items}).Write(context.Background(), &output, false); err != nil || count != len(items) {
		t.Fatalf("large Write() = %d %v", count, err)
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := NewService(&exportStore{items: items[:1]}).Write(cancelled, &bytes.Buffer{}, false); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel error = %v", err)
	}
	sentinel := errors.New("snapshot failed")
	if _, err := NewService(&exportStore{err: sentinel}).Write(context.Background(), &bytes.Buffer{}, false); !errors.Is(err, sentinel) {
		t.Fatalf("store error = %v", err)
	}
	if _, err := (*Service)(nil).Write(context.Background(), &bytes.Buffer{}, false); err == nil {
		t.Fatal("nil service succeeded")
	}
}
