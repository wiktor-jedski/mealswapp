package dataimporter

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/customitem"
	"github.com/wiktor-jedski/mealswapp/backend/internal/providerregistry"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// Implements DESIGN-009 DataImporter service validation and conflict verification.

type importStoreStub struct {
	claim  repository.CuratedImportConfirmation
	result repository.CuratedImportConfirmationResult
	err    error
	calls  int
}

func (s *importStoreStub) ConfirmCuratedImport(_ context.Context, _ repository.AdminMutationExecutor, claim repository.CuratedImportConfirmation) (repository.CuratedImportConfirmationResult, error) {
	s.calls++
	s.claim = claim
	return s.result, s.err
}

type importExecutorStub struct {
	repository.AdminMutationExecutor
}

type evidenceResolverStub struct {
	identity providerregistry.Identity
	err      error
}

func (s evidenceResolverStub) Resolve(string) (providerregistry.Identity, error) {
	return s.identity, s.err
}

func TestServiceConfirmNormalizesDraftAndMapsResult(t *testing.T) {
	foodID, importID := uuid.New(), uuid.New()
	store := &importStoreStub{result: repository.CuratedImportConfirmationResult{ImportID: importID, Item: repository.FoodItemEntity{ID: foodID, Name: "Curated tofu", PhysicalState: repository.PhysicalStateSolid}}}
	service := NewService(store)
	result, err := service.Confirm(context.Background(), importExecutorStub{}, uuid.New(), "ignored-natural-key", Request{
		SelectedRecord: providerregistry.Identity{Provider: "usda", ExternalID: "fdc-1"}, Request: validRequest(" Curated tofu "),
	})
	if err != nil || result.ImportID != importID || result.FoodItemID != foodID || store.calls != 1 {
		t.Fatalf("result=%+v calls=%d err=%v", result, store.calls, err)
	}
	if store.claim.SourceProvider != "usda" || store.claim.ExternalID != "fdc-1" || store.claim.Item.Name != "Curated tofu" || len(store.claim.BodyHash) != 64 {
		t.Fatalf("normalized claim=%+v", store.claim)
	}
}

func TestServiceDerivesImportedDensityFromSelectedServerRecord(t *testing.T) {
	store := &importStoreStub{result: repository.CuratedImportConfirmationResult{ImportID: uuid.New(), Item: repository.FoodItemEntity{ID: uuid.New(), Name: "Drink", PhysicalState: repository.PhysicalStateLiquid}}}
	identity := providerregistry.Identity{Provider: "usda", ExternalID: "171265"}
	service := NewService(store, evidenceResolverStub{identity: identity})
	request := Request{ExternalRecordToken: "opaque", Request: customitem.Request{
		Name: "Drink", PhysicalState: repository.PhysicalStateLiquid, DensityGramsPerMilliliter: 1.03, DensitySourceKind: "imported",
		MacrosPer100: repository.MacroValues{}, Micros: repository.MicroValues{}, FoodCategoryIDs: []uuid.UUID{}, CulinaryRoleIDs: []uuid.UUID{},
	}}
	if _, err := service.Confirm(context.Background(), importExecutorStub{}, uuid.New(), "", request); err != nil {
		t.Fatal(err)
	}
	if store.claim.SourceProvider != identity.Provider || store.claim.ExternalID != identity.ExternalID ||
		store.claim.Item.DensitySourceProvider != identity.Provider || store.claim.Item.DensitySourceFoodID != identity.ExternalID ||
		store.claim.Item.DensitySourceKind != "imported" {
		t.Fatalf("trusted claim = %+v", store.claim)
	}
	request.SelectedRecord = providerregistry.Identity{Provider: "usda", ExternalID: "different"}
	if _, err := service.Confirm(context.Background(), importExecutorStub{}, uuid.New(), "", request); !errors.Is(err, ErrExternalRecordEvidence) {
		t.Fatalf("mismatched evidence error = %v", err)
	}
}

func TestServiceRejectsUnknownMalformedAndStaleEvidenceBeforePersistence(t *testing.T) {
	for _, test := range []struct {
		name    string
		service *Service
		request Request
	}{
		{name: "unknown token", service: NewService(&importStoreStub{}, evidenceResolverStub{err: errors.New("unknown")}), request: Request{ExternalRecordToken: "unknown", Request: validRequest("Food")}},
		{name: "resolver unavailable", service: NewService(&importStoreStub{}), request: Request{ExternalRecordToken: "malformed", Request: validRequest("Food")}},
		{name: "unregistered provider", service: NewService(&importStoreStub{}, evidenceResolverStub{identity: providerregistry.Identity{Provider: "fixture", ExternalID: "1"}}), request: Request{ExternalRecordToken: "stale", Request: validRequest("Food")}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := test.service.Confirm(context.Background(), importExecutorStub{}, uuid.New(), "valid-key", test.request); !errors.Is(err, ErrExternalRecordEvidence) {
				t.Fatalf("error = %v", err)
			}
			if test.service.store.(*importStoreStub).calls != 0 {
				t.Fatal("invalid evidence reached persistence")
			}
		})
	}
}

func TestServiceConfirmUsesTypedCurationNormalizationBeforePersistence(t *testing.T) {
	store := &importStoreStub{result: repository.CuratedImportConfirmationResult{ImportID: uuid.New(), Item: repository.FoodItemEntity{ID: uuid.New(), Name: "Café au lait", PhysicalState: repository.PhysicalStateSolid}}}
	service := NewService(store)
	req := Request{SelectedRecord: providerregistry.Identity{Provider: "usda", ExternalID: "fdc-typed-1"}, Request: validRequest("  Cafe\u0301   au lait  ")}
	req.ImageURL = " https://images.example.com/cafe.jpg "

	if _, err := service.Confirm(context.Background(), importExecutorStub{}, uuid.New(), "ignored-natural-key", req); err != nil {
		t.Fatal(err)
	}
	if store.claim.Item.Name != "Café au lait" || store.claim.Item.ImageURL != "https://images.example.com/cafe.jpg" {
		t.Fatalf("typed normalized claim=%+v", store.claim.Item)
	}
}

func TestServiceConfirmRejectsInvalidDraftsBeforePersistence(t *testing.T) {
	store := &importStoreStub{}
	service := NewService(store)
	cases := []struct {
		name string
		key  string
		req  Request
	}{
		{name: "half identity", key: "", req: Request{SelectedRecord: providerregistry.Identity{Provider: "usda"}, Request: validRequest("Food")}},
		{name: "unsupported provider", req: Request{SelectedRecord: providerregistry.Identity{Provider: "other", ExternalID: "1"}, Request: validRequest("Food")}},
		{name: "missing key", req: Request{Request: validRequest("Food")}},
		{name: "liquid density", key: "valid-key", req: Request{Request: customitem.Request{Name: "Drink", PhysicalState: repository.PhysicalStateLiquid, MacrosPer100: repository.MacroValues{}, Micros: repository.MicroValues{}, FoodCategoryIDs: []uuid.UUID{}, CulinaryRoleIDs: []uuid.UUID{}}}},
		{name: "control character", key: "valid-key", req: Request{Request: validRequest("Oat\nMilk")}},
		{name: "unsafe image URL", key: "valid-key", req: Request{Request: func() customitem.Request {
			req := validRequest("Food")
			req.ImageURL = "http://127.0.0.1/image"
			return req
		}()}},
		{name: "numeric upper bound", key: "valid-key", req: Request{Request: func() customitem.Request {
			req := validRequest("Food")
			req.MacrosPer100.Protein = math.MaxFloat64
			return req
		}()}},
		{name: "imported density without evidence", key: "valid-key", req: Request{Request: customitem.Request{Name: "Drink", PhysicalState: repository.PhysicalStateLiquid, DensityGramsPerMilliliter: 1, DensitySourceKind: "imported", MacrosPer100: repository.MacroValues{}, Micros: repository.MicroValues{}, FoodCategoryIDs: []uuid.UUID{}, CulinaryRoleIDs: []uuid.UUID{}}}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if _, err := service.Confirm(context.Background(), importExecutorStub{}, uuid.New(), test.key, test.req); err == nil {
				t.Fatal("Confirm() error = nil")
			}
		})
	}
	if store.calls != 0 {
		t.Fatalf("store calls=%d", store.calls)
	}
}

func TestServiceConfirmMapsConflictClasses(t *testing.T) {
	cases := []struct {
		storeErr error
		want     error
	}{
		{repository.ErrCuratedImportIdentityConflict, ErrProviderConflict},
		{repository.ErrCuratedImportNameConfirmationRequired, ErrNameConfirmation},
		{repository.NewError(repository.ErrorKindIdempotencyConflict, "conflict", nil), ErrIdempotencyConflict},
	}
	for _, test := range cases {
		service := NewService(&importStoreStub{err: test.storeErr})
		_, err := service.Confirm(context.Background(), importExecutorStub{}, uuid.New(), "", Request{SelectedRecord: providerregistry.Identity{Provider: "usda", ExternalID: uuid.NewString()}, Request: validRequest("Food")})
		if !errors.Is(err, test.want) {
			t.Fatalf("error=%v want=%v", err, test.want)
		}
	}
}

func validRequest(name string) customitem.Request {
	return customitem.Request{Name: name, PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{Protein: 10}, Micros: repository.MicroValues{}, FoodCategoryIDs: []uuid.UUID{}, CulinaryRoleIDs: []uuid.UUID{}}
}
