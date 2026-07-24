package customitem

// Implements DESIGN-008 ProfileController custom-item service verification.

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

type memoryItems struct {
	mu       sync.Mutex
	items    map[uuid.UUID]repository.CustomFoodItemEntity
	claims   map[string]memoryClaim
	claimErr error
}

type memoryClaim struct {
	bodyHash string
	result   repository.CustomFoodItemCreateClaimResult
}

func (r *memoryItems) GetByID(_ context.Context, ownerID, id uuid.UUID, _ repository.RepositoryContext) (repository.CustomFoodItemEntity, error) {
	item, ok := r.items[id]
	if !ok || item.OwnerID != ownerID {
		return repository.CustomFoodItemEntity{}, repository.NewError(repository.ErrorKindNotFound, "custom food item not found", nil)
	}
	return item, nil
}
func (r *memoryItems) List(_ context.Context, ownerID uuid.UUID, _ repository.RepositoryContext) ([]repository.CustomFoodItemEntity, error) {
	items := []repository.CustomFoodItemEntity{}
	for _, item := range r.items {
		if item.OwnerID == ownerID {
			items = append(items, item)
		}
	}
	return items, nil
}
func (r *memoryItems) ClaimCreate(_ context.Context, claim repository.CustomFoodItemCreateClaim, encode repository.CustomFoodItemResponseEncoder) (repository.CustomFoodItemCreateClaimResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.claimErr != nil {
		return repository.CustomFoodItemCreateClaimResult{}, r.claimErr
	}
	if r.claims == nil {
		r.claims = map[string]memoryClaim{}
	}
	scope := claim.UserID.String() + "|" + claim.Key
	if existing, ok := r.claims[scope]; ok {
		if existing.bodyHash != claim.BodyHash {
			return repository.CustomFoodItemCreateClaimResult{}, repository.NewError(repository.ErrorKindIdempotencyConflict, "idempotency key reused", nil)
		}
		result := existing.result
		result.Replayed = true
		return result, nil
	}
	claim.Item.ID = uuid.New()
	payload, err := encode(claim.Item)
	if err != nil {
		return repository.CustomFoodItemCreateClaimResult{}, err
	}
	result := repository.CustomFoodItemCreateClaimResult{ResponseBody: payload, StatusCode: 201}
	r.items[claim.Item.ID] = claim.Item
	r.claims[scope] = memoryClaim{bodyHash: claim.BodyHash, result: result}
	return result, nil
}
func (r *memoryItems) Create(_ context.Context, item repository.CustomFoodItemEntity) (uuid.UUID, error) {
	item.ID = uuid.New()
	r.items[item.ID] = item
	return item.ID, nil
}
func (r *memoryItems) Update(_ context.Context, item repository.CustomFoodItemEntity) error {
	stored, ok := r.items[item.ID]
	if !ok || stored.OwnerID != item.OwnerID {
		return repository.NewError(repository.ErrorKindNotFound, "custom food item not found", nil)
	}
	r.items[item.ID] = item
	return nil
}
func (r *memoryItems) Delete(_ context.Context, ownerID, id uuid.UUID) error {
	stored, ok := r.items[id]
	if !ok || stored.OwnerID != ownerID {
		return repository.NewError(repository.ErrorKindNotFound, "custom food item not found", nil)
	}
	delete(r.items, id)
	return nil
}

func solidRequest(name string) Request {
	return Request{Name: name, PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{Protein: 10}, Micros: repository.MicroValues{}}
}

func TestServiceCreateReplayNormalizesBodyAndRejectsKeyReuse(t *testing.T) {
	ownerID := uuid.New()
	items := &memoryItems{items: map[uuid.UUID]repository.CustomFoodItemEntity{}}
	service := NewService(items)
	request := CreateRequest{Request: solidRequest("  Tofu  "), IdempotencyKey: "custom-key-1"}

	created, err := service.Create(context.Background(), ownerID, request)
	if err != nil || created.Status != 201 || created.Replayed || created.Item.Name != "Tofu" || len(items.items) != 1 {
		t.Fatalf("create = %+v items=%d err=%v", created, len(items.items), err)
	}
	request.Name = "Tofu"
	replayed, err := service.Create(context.Background(), ownerID, request)
	if err != nil || !replayed.Replayed || replayed.Item.ID != created.Item.ID || len(items.items) != 1 {
		t.Fatalf("replay = %+v items=%d err=%v", replayed, len(items.items), err)
	}
	request.Name = "Tempeh"
	if _, err := service.Create(context.Background(), ownerID, request); err != ErrIdempotencyConflict || len(items.items) != 1 {
		t.Fatalf("conflicting retry err=%v items=%d", err, len(items.items))
	}
}

func TestServiceCreatePreservesResourceConflictClassification(t *testing.T) {
	conflict := repository.NewError(repository.ErrorKindConflict, "duplicate custom item name", nil)
	items := &memoryItems{items: map[uuid.UUID]repository.CustomFoodItemEntity{}, claimErr: conflict}
	_, err := NewService(items).Create(context.Background(), uuid.New(), CreateRequest{Request: solidRequest("Tofu"), IdempotencyKey: "different-key"})
	if err != conflict || errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("duplicate-name error = %v, want unchanged resource conflict", err)
	}
}

func TestFromEntityStripsClassificationHierarchyFromPublicProjection(t *testing.T) {
	parentID, childID := uuid.New(), uuid.New()
	item := fromEntity(repository.CustomFoodItemEntity{FoodItemEntity: repository.FoodItemEntity{
		FoodCategories: []repository.ClassificationEntity{{ID: childID, Name: "Child", Kind: repository.ClassificationKindFoodCategory, ParentID: &parentID}},
	}})
	if len(item.FoodCategories) != 1 || item.FoodCategories[0].ID != childID || item.FoodCategories[0].Name != "Child" {
		t.Fatalf("classification projection = %#v", item.FoodCategories)
	}
	payload, err := json.Marshal(item)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(payload), "parentId") {
		t.Fatalf("public projection leaked hierarchy: %s", payload)
	}
	if empty := classificationSummaries(nil); empty == nil || len(empty) != 0 {
		t.Fatalf("empty classification projection = %#v", empty)
	}
}

func TestServiceDerivesOwnershipAndKeepsCrossUserItemsNotFound(t *testing.T) {
	ownerID, otherID := uuid.New(), uuid.New()
	items := &memoryItems{items: map[uuid.UUID]repository.CustomFoodItemEntity{}}
	service := NewService(items)
	created, err := service.Create(context.Background(), ownerID, CreateRequest{Request: solidRequest("Private"), IdempotencyKey: "custom-key-2"})
	if err != nil {
		t.Fatal(err)
	}
	if items.items[created.Item.ID].OwnerID != ownerID {
		t.Fatalf("stored owner = %s, want authenticated owner %s", items.items[created.Item.ID].OwnerID, ownerID)
	}
	if _, err := service.Get(context.Background(), otherID, created.Item.ID); !repository.IsKind(err, repository.ErrorKindNotFound) {
		t.Fatalf("cross-user get err = %v", err)
	}
	if _, err := service.Update(context.Background(), otherID, created.Item.ID, solidRequest("Stolen")); !repository.IsKind(err, repository.ErrorKindNotFound) {
		t.Fatalf("cross-user update err = %v", err)
	}
	if err := service.Delete(context.Background(), otherID, created.Item.ID); !repository.IsKind(err, repository.ErrorKindNotFound) {
		t.Fatalf("cross-user delete err = %v", err)
	}
	if _, err := service.Get(context.Background(), ownerID, created.Item.ID); err != nil {
		t.Fatalf("owner item changed after cross-user attempts: %v", err)
	}
}

func TestServiceConcurrentCreateHasOneSideEffect(t *testing.T) {
	ownerID := uuid.New()
	items := &memoryItems{items: map[uuid.UUID]repository.CustomFoodItemEntity{}}
	service := NewService(items)
	req := CreateRequest{Request: solidRequest("Concurrent"), IdempotencyKey: "custom-concurrent"}
	results := make(chan CreateResult, 2)
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := service.Create(context.Background(), ownerID, req)
			results <- result
			errs <- err
		}()
	}
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	var first uuid.UUID
	for result := range results {
		if first == uuid.Nil {
			first = result.Item.ID
		} else if result.Item.ID != first {
			t.Fatalf("concurrent result id = %s, want %s", result.Item.ID, first)
		}
	}
	if len(items.items) != 1 {
		t.Fatalf("concurrent side effects = %d, want 1", len(items.items))
	}
}

func TestValidateRequestRejectsDomainViolations(t *testing.T) {
	valid := solidRequest("Valid")
	tests := []Request{
		{Name: " ", PhysicalState: repository.PhysicalStateSolid, Micros: repository.MicroValues{}},
		{Name: "Valid", PhysicalState: "gas", Micros: repository.MicroValues{}},
		func() Request { r := valid; r.PrepTimeMinutes = -1; return r }(),
		func() Request { r := valid; r.MacrosPer100.Protein = math.NaN(); return r }(),
		func() Request {
			r := valid
			r.MacrosPer100 = repository.MacroValues{Protein: 60, Carbohydrates: 41}
			return r
		}(),
		func() Request { r := valid; r.AverageUnitWeightGrams = -1; return r }(),
		func() Request { r := valid; r.ImageURL = "://bad"; return r }(),
		func() Request { r := valid; r.ImageURL = "FTP://example.test/item.png"; return r }(),
		func() Request { r := valid; r.Name = "invalid\x00name"; return r }(),
		func() Request { r := valid; r.ImageURL = "https://example.test/invalid\x00image"; return r }(),
		func() Request { r := valid; r.Micros = repository.MicroValues{"invalid\x00key": 1}; return r }(),
		func() Request {
			r := valid
			r.PhysicalState = repository.PhysicalStateLiquid
			r.DensityGramsPerMilliliter = 1
			r.DensitySourceKind = "manual"
			r.DensitySourceProvider = "invalid\x00provider"
			return r
		}(),
		func() Request {
			r := valid
			r.PhysicalState = repository.PhysicalStateLiquid
			r.DensityGramsPerMilliliter = 1
			r.DensitySourceKind = "manual"
			r.DensitySourceFoodID = "invalid\x00food"
			return r
		}(),
		func() Request {
			r := valid
			r.PhysicalState = repository.PhysicalStateLiquid
			r.DensityGramsPerMilliliter = 1
			r.DensitySourceKind = "manual\x00"
			return r
		}(),
		func() Request { r := valid; r.Micros = repository.MicroValues{"Sodium": -1}; return r }(),
		func() Request { r := valid; r.FoodCategoryIDs = []uuid.UUID{uuid.Nil}; return r }(),
		func() Request { r := valid; r.PhysicalState = repository.PhysicalStateLiquid; return r }(),
		func() Request { r := valid; r.DensityGramsPerMilliliter = 1; return r }(),
	}
	for index, req := range tests {
		if _, err := ValidateRequest(req); !repository.IsKind(err, repository.ErrorKindValidation) {
			t.Fatalf("case %d error = %v", index, err)
		}
	}
	liquid := valid
	liquid.PhysicalState = repository.PhysicalStateLiquid
	liquid.DensityGramsPerMilliliter = 1
	liquid.DensitySourceKind = "manual"
	if _, err := ValidateRequest(liquid); err != nil {
		t.Fatalf("valid liquid error = %v", err)
	}
}

func TestServiceClaimAndDependencyErrorBranches(t *testing.T) {
	if _, err := NewService(nil).Create(context.Background(), uuid.New(), CreateRequest{Request: solidRequest("Unavailable"), IdempotencyKey: "custom-unavailable"}); !repository.IsKind(err, repository.ErrorKindConnection) {
		t.Fatalf("nil create error = %v", err)
	}
	items := &memoryItems{items: map[uuid.UUID]repository.CustomFoodItemEntity{}}
	service := NewService(items)
	_, err := service.Create(context.Background(), uuid.New(), CreateRequest{Request: solidRequest("Decode"), IdempotencyKey: "custom-decode"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := createResultFromClaim(repository.CustomFoodItemCreateClaimResult{ResponseBody: []byte("{"), StatusCode: 201}); err == nil {
		t.Fatal("malformed claim response accepted")
	}
}

func TestServiceCRUDListAndInvalidRequestsHaveExpectedSideEffects(t *testing.T) {
	ctx := context.Background()
	ownerID := uuid.New()
	items := &memoryItems{items: map[uuid.UUID]repository.CustomFoodItemEntity{}}
	service := NewService(items)
	invalid := solidRequest("Invalid")
	invalid.MacrosPer100.Protein = -1
	if _, err := service.Create(ctx, ownerID, CreateRequest{Request: invalid, IdempotencyKey: "invalid-no-claim"}); !repository.IsKind(err, repository.ErrorKindValidation) || len(items.items) != 0 || len(items.claims) != 0 {
		t.Fatalf("invalid create err=%v items=%d claims=%d", err, len(items.items), len(items.claims))
	}
	created, err := service.Create(ctx, ownerID, CreateRequest{Request: solidRequest("CRUD"), IdempotencyKey: "custom-crud-key"})
	if err != nil {
		t.Fatal(err)
	}
	if got, err := service.Get(ctx, ownerID, created.Item.ID); err != nil || got.ID != created.Item.ID {
		t.Fatalf("Get() = %+v err=%v", got, err)
	}
	updated, err := service.Update(ctx, ownerID, created.Item.ID, solidRequest("Updated CRUD"))
	if err != nil || updated.Name != "Updated CRUD" {
		t.Fatalf("Update() = %+v err=%v", updated, err)
	}
	listed, err := service.List(ctx, ownerID)
	if err != nil || len(listed) != 1 || listed[0].ID != created.Item.ID {
		t.Fatalf("List() = %+v err=%v", listed, err)
	}
	if err := service.Delete(ctx, ownerID, created.Item.ID); err != nil {
		t.Fatal(err)
	}
	if listed, err := service.List(ctx, ownerID); err != nil || len(listed) != 0 {
		t.Fatalf("List() after delete = %+v err=%v", listed, err)
	}
	if _, err := service.Get(ctx, uuid.Nil, created.Item.ID); !repository.IsKind(err, repository.ErrorKindValidation) {
		t.Fatalf("nil-owner Get() err=%v", err)
	}
	if _, err := service.Update(ctx, ownerID, uuid.Nil, solidRequest("X")); !repository.IsKind(err, repository.ErrorKindValidation) {
		t.Fatalf("nil-item Update() err=%v", err)
	}
	if err := service.Delete(ctx, ownerID, uuid.Nil); !repository.IsKind(err, repository.ErrorKindValidation) {
		t.Fatalf("nil-item Delete() err=%v", err)
	}
	if _, err := service.List(ctx, uuid.Nil); !repository.IsKind(err, repository.ErrorKindValidation) {
		t.Fatalf("nil-owner List() err=%v", err)
	}
	if _, err := NewService(nil).List(ctx, ownerID); !repository.IsKind(err, repository.ErrorKindConnection) {
		t.Fatalf("nil-repository List() err=%v", err)
	}
}
