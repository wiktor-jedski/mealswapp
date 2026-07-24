package itemcurator

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// memoryStore verifies ItemCurator behavior without weakening the global-only store boundary.
// Implements DESIGN-009 ItemCurator unit-test persistence boundary.
type memoryStore struct {
	items  map[uuid.UUID]repository.FoodItemEntity
	claims map[string]memoryClaim
}

type memoryClaim struct {
	hash string
	body []byte
}

func newMemoryStore() *memoryStore {
	return &memoryStore{items: map[uuid.UUID]repository.FoodItemEntity{}, claims: map[string]memoryClaim{}}
}

func (s *memoryStore) GetByID(_ context.Context, id uuid.UUID, _ bool) (repository.FoodItemEntity, error) {
	item, ok := s.items[id]
	if !ok {
		return repository.FoodItemEntity{}, repository.NewError(repository.ErrorKindNotFound, "not found", nil)
	}
	return item, nil
}

func (s *memoryStore) GetByIDInMutation(ctx context.Context, _ repository.AdminMutationExecutor, id uuid.UUID, deleted bool) (repository.FoodItemEntity, error) {
	return s.GetByID(ctx, id, deleted)
}

func (s *memoryStore) ClaimCreate(_ context.Context, _ repository.AdminMutationExecutor, claim repository.ManualFoodItemCreateClaim, encode repository.ManualFoodItemResponseEncoder) (repository.ManualFoodItemCreateClaimResult, error) {
	if existing, ok := s.claims[claim.Key]; ok {
		if existing.hash != claim.BodyHash {
			return repository.ManualFoodItemCreateClaimResult{}, repository.NewError(repository.ErrorKindIdempotencyConflict, "conflict", nil)
		}
		return repository.ManualFoodItemCreateClaimResult{ResponseBody: existing.body, StatusCode: 201, Replayed: true}, nil
	}
	for _, item := range s.items {
		if item.Name == claim.Item.Name {
			return repository.ManualFoodItemCreateClaimResult{}, repository.NewError(repository.ErrorKindConflict, "duplicate", nil)
		}
	}
	claim.Item.ID = uuid.New()
	s.items[claim.Item.ID] = claim.Item
	body, err := encode(claim.Item)
	if err != nil {
		return repository.ManualFoodItemCreateClaimResult{}, err
	}
	s.claims[claim.Key] = memoryClaim{hash: claim.BodyHash, body: body}
	return repository.ManualFoodItemCreateClaimResult{ResponseBody: body, StatusCode: 201}, nil
}

func (s *memoryStore) Update(_ context.Context, _ repository.AdminMutationExecutor, item repository.FoodItemEntity) error {
	if _, ok := s.items[item.ID]; !ok {
		return repository.NewError(repository.ErrorKindNotFound, "not found", nil)
	}
	s.items[item.ID] = item
	return nil
}

func (s *memoryStore) Delete(_ context.Context, _ repository.AdminMutationExecutor, id uuid.UUID) error {
	if _, ok := s.items[id]; !ok {
		return repository.NewError(repository.ErrorKindNotFound, "not found", nil)
	}
	delete(s.items, id)
	return nil
}

// inertExecutor is a non-nil transaction token for store-isolated service tests.
// Implements DESIGN-009 ItemCurator unit-test transaction boundary.
type inertExecutor struct{}

func (inertExecutor) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (inertExecutor) Query(context.Context, string, ...any) (pgx.Rows, error) { return nil, nil }
func (inertExecutor) QueryRow(context.Context, string, ...any) pgx.Row        { return nil }

func solidRequest(name string) Request {
	return Request{Name: name, PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{Protein: 20, Carbohydrates: 10, Fat: 5}, Micros: repository.MicroValues{}, ImageURL: "https://images.example.test/item.png"}
}

func TestServiceCreateReplayConflictDuplicateAndCRUD(t *testing.T) {
	store := newMemoryStore()
	service := NewService(store)
	tx := inertExecutor{}
	adminID := uuid.New()
	req := solidRequest("Manual tofu")
	first, err := service.Create(context.Background(), tx, adminID, "manual-key-0001", req)
	if err != nil || first.Replayed || first.Item.ID == uuid.Nil {
		t.Fatalf("first create = %+v err=%v", first, err)
	}
	reordered := req
	reordered.Name = "Manual seitan"
	reordered.FoodCategoryIDs = []uuid.UUID{uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"), uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")}
	if _, err := service.Create(context.Background(), tx, adminID, "different-key", reordered); err != nil {
		t.Fatalf("normalized classifications create: %v", err)
	}
	replay, err := service.Create(context.Background(), tx, adminID, "manual-key-0001", req)
	if err != nil || !replay.Replayed || replay.Item.ID != first.Item.ID {
		t.Fatalf("replay = %+v err=%v", replay, err)
	}
	changed := req
	changed.Name = "Changed"
	if _, err := service.Create(context.Background(), tx, adminID, "manual-key-0001", changed); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("changed-key error = %v", err)
	}
	if _, err := service.Create(context.Background(), tx, adminID, "duplicate-key", req); !repository.IsKind(err, repository.ErrorKindConflict) {
		t.Fatalf("duplicate-name error = %v", err)
	}
	loaded, err := service.Get(context.Background(), first.Item.ID)
	if err != nil || loaded.Name != req.Name {
		t.Fatalf("get = %+v err=%v", loaded, err)
	}
	updated := solidRequest("Manual tempeh")
	mutation, err := service.Update(context.Background(), tx, first.Item.ID, updated)
	if err != nil || mutation.Before.Name != req.Name || mutation.After.Name != updated.Name {
		t.Fatalf("update = %+v err=%v", mutation, err)
	}
	deleted, err := service.Delete(context.Background(), tx, first.Item.ID)
	if err != nil || deleted.Before.Name != updated.Name {
		t.Fatalf("delete = %+v err=%v", deleted, err)
	}
	if _, err := service.Get(context.Background(), first.Item.ID); !repository.IsKind(err, repository.ErrorKindNotFound) {
		t.Fatalf("get deleted error = %v", err)
	}
}

func TestServiceRejectsInvalidFieldsAndLiquidDensity(t *testing.T) {
	service := NewService(newMemoryStore())
	tx := inertExecutor{}
	adminID := uuid.New()
	cases := []Request{
		{Name: "", PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{}},
		{Name: "Bad macros", PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{Protein: 101}},
		{Name: "Bad image", PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{}, ImageURL: "ftp://example.test/a"},
		{Name: "No density", PhysicalState: repository.PhysicalStateLiquid, MacrosPer100: repository.MacroValues{}},
		{Name: "Bad density source", PhysicalState: repository.PhysicalStateLiquid, DensityGramsPerMilliliter: 1, DensitySourceKind: "invented", MacrosPer100: repository.MacroValues{}},
		{Name: "Bad micro", PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{}, Micros: repository.MicroValues{"bad\x00key": 1}},
	}
	for index, req := range cases {
		if _, err := service.Create(context.Background(), tx, adminID, "invalid-key-0001", req); !repository.IsKind(err, repository.ErrorKindValidation) {
			encoded, _ := json.Marshal(req)
			t.Fatalf("case %d (%s) error = %v", index, encoded, err)
		}
	}
	liquid := Request{Name: "Manual milk", PhysicalState: repository.PhysicalStateLiquid, AverageServingVolumeMilliliters: 250, DensityGramsPerMilliliter: 1.03, DensitySourceKind: "manual", MacrosPer100: repository.MacroValues{Protein: 3}}
	if result, err := service.Create(context.Background(), tx, adminID, "liquid-key-0001", liquid); err != nil || result.Item.DensityGramsPerMilliliter != 1.03 {
		t.Fatalf("valid liquid = %+v err=%v", result, err)
	}
}
