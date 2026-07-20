package dailydiet

// Implements DESIGN-008 ProfileController and SavedDataRepository verification.

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

type memoryDietRepository struct {
	mu            sync.Mutex
	diets         map[uuid.UUID]repository.SavedDiet
	idempotencies map[string]memoryDailyDietClaim
	createCalls   int
	replaceCalls  int
	deleteCalls   int
}

type memoryDailyDietClaim struct {
	bodyHash string
	result   repository.DailyDietCreateClaimResult
}

func (r *memoryDietRepository) Create(_ context.Context, userID uuid.UUID, diet repository.SavedDiet) (uuid.UUID, error) {
	r.createCalls++
	if r.diets == nil {
		r.diets = map[uuid.UUID]repository.SavedDiet{}
	}
	id := diet.ID
	if id == uuid.Nil {
		id = uuid.New()
	}
	now := time.Now().UTC()
	diet.ID, diet.UserID, diet.CreatedAt, diet.UpdatedAt = id, userID, now, now
	diet.Entries = copyEntries(diet.Entries, id)
	r.diets[id] = diet
	return id, nil
}

func (r *memoryDietRepository) GetDailyDietCreateClaim(_ context.Context, userID uuid.UUID, key, bodyHash string) (repository.DailyDietCreateClaimResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	record, ok := r.idempotencies[userID.String()+key]
	if !ok {
		return repository.DailyDietCreateClaimResult{}, repository.NewError(repository.ErrorKindNotFound, "missing", nil)
	}
	if record.bodyHash != bodyHash {
		return repository.DailyDietCreateClaimResult{}, repository.NewError(repository.ErrorKindConflict, "idempotency key reused with different body", nil)
	}
	result := record.result
	result.Replayed = true
	return result, nil
}

func (r *memoryDietRepository) ClaimDailyDietCreate(_ context.Context, claim repository.DailyDietCreateClaim) (repository.DailyDietCreateClaimResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.idempotencies == nil {
		r.idempotencies = map[string]memoryDailyDietClaim{}
	}
	key := claim.UserID.String() + claim.Key
	if existing, ok := r.idempotencies[key]; ok {
		if existing.bodyHash != claim.BodyHash {
			return repository.DailyDietCreateClaimResult{}, repository.NewError(repository.ErrorKindConflict, "idempotency key reused with different body", nil)
		}
		result := existing.result
		result.Replayed = true
		return result, nil
	}
	if r.diets == nil {
		r.diets = map[uuid.UUID]repository.SavedDiet{}
	}
	r.createCalls++
	r.diets[claim.Diet.ID] = claim.Diet
	result := repository.DailyDietCreateClaimResult{Response: claim.Response, StatusCode: claim.StatusCode}
	r.idempotencies[key] = memoryDailyDietClaim{bodyHash: claim.BodyHash, result: result}
	return result, nil
}

func (r *memoryDietRepository) Get(_ context.Context, userID, dietID uuid.UUID) (repository.SavedDiet, error) {
	diet, ok := r.diets[dietID]
	if !ok || diet.UserID != userID {
		return repository.SavedDiet{}, repository.NewError(repository.ErrorKindNotFound, "saved diet not found", nil)
	}
	diet.Entries = copyEntries(diet.Entries, diet.ID)
	return diet, nil
}

func (r *memoryDietRepository) List(_ context.Context, userID uuid.UUID) ([]repository.SavedDiet, error) {
	result := []repository.SavedDiet{}
	for _, diet := range r.diets {
		if diet.UserID == userID {
			diet.Entries = copyEntries(diet.Entries, diet.ID)
			result = append(result, diet)
		}
	}
	return result, nil
}

func (r *memoryDietRepository) Replace(_ context.Context, userID uuid.UUID, diet repository.SavedDiet) error {
	r.replaceCalls++
	stored, ok := r.diets[diet.ID]
	if !ok || stored.UserID != userID {
		return repository.NewError(repository.ErrorKindNotFound, "saved diet not found", nil)
	}
	diet.UserID, diet.CreatedAt, diet.UpdatedAt = userID, stored.CreatedAt, time.Now().UTC()
	diet.Entries = copyEntries(diet.Entries, diet.ID)
	r.diets[diet.ID] = diet
	return nil
}

func (r *memoryDietRepository) Delete(_ context.Context, userID, dietID uuid.UUID) error {
	r.deleteCalls++
	diet, ok := r.diets[dietID]
	if !ok || diet.UserID != userID {
		return repository.NewError(repository.ErrorKindNotFound, "saved diet not found", nil)
	}
	delete(r.diets, dietID)
	return nil
}

func (r *memoryDietRepository) DeleteIfOwned(_ context.Context, userID, dietID uuid.UUID) (bool, bool, error) {
	diet, ok := r.diets[dietID]
	if !ok {
		return false, false, nil
	}
	if diet.UserID != userID {
		return false, true, nil
	}
	r.deleteCalls++
	delete(r.diets, dietID)
	return true, true, nil
}

type memoryMealRepository struct {
	mu      sync.Mutex
	meals   map[uuid.UUID]repository.MealEntity
	calls   map[uuid.UUID]int
	blockID uuid.UUID
	started chan<- struct{}
	release <-chan struct{}
}

func (r *memoryMealRepository) GetByID(ctx context.Context, id uuid.UUID, _ repository.RepositoryContext) (repository.MealEntity, error) {
	if id == r.blockID && r.release != nil {
		if r.started != nil {
			select {
			case r.started <- struct{}{}:
			default:
			}
		}
		select {
		case <-r.release:
		case <-ctx.Done():
			return repository.MealEntity{}, ctx.Err()
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.calls == nil {
		r.calls = map[uuid.UUID]int{}
	}
	r.calls[id]++
	meal, ok := r.meals[id]
	if !ok {
		return repository.MealEntity{}, repository.NewError(repository.ErrorKindNotFound, "meal not found", nil)
	}
	return meal, nil
}

func (r *memoryMealRepository) Search(context.Context, repository.RepositoryQuery) ([]repository.MealEntity, int, error) {
	return nil, 0, nil
}
func (r *memoryMealRepository) CalculateMacros(context.Context, uuid.UUID) (repository.MacroValues, error) {
	return repository.MacroValues{}, nil
}
func (r *memoryMealRepository) Create(context.Context, repository.MealEntity) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (r *memoryMealRepository) Update(context.Context, repository.MealEntity) error { return nil }
func (r *memoryMealRepository) Delete(context.Context, uuid.UUID) error             { return nil }

func TestServiceCreateAggregatesMultipleMealsAndReplaysIdempotently(t *testing.T) {
	userID := uuid.New()
	mealA, mealB := uuid.New(), uuid.New()
	meals := &memoryMealRepository{meals: map[uuid.UUID]repository.MealEntity{
		mealA: {ID: mealA, PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{Protein: 10, Carbohydrates: 20, Fat: 5}},
		mealB: {ID: mealB, PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{Protein: 5, Carbohydrates: 5, Fat: 2}},
	}}
	diets := &memoryDietRepository{}
	service := NewService(diets, meals)
	request := CreateRequest{Name: "  Training Day ", IdempotencyKey: "daily-diet-1", Entries: []MealQuantity{
		{MealID: mealA, Quantity: 100, Unit: "g", Position: 0},
		{MealID: mealB, Quantity: 200, Unit: "g", Position: 1},
	}}

	created, err := service.Create(context.Background(), userID, request)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Status != 201 || created.Replayed || created.Diet.Name != "Training Day" || diets.createCalls != 1 {
		t.Fatalf("created = %+v createCalls=%d", created, diets.createCalls)
	}
	want := MacroProjection{Protein: 20, Carbohydrates: 30, Fat: 9, Calories: 281}
	if created.Diet.AggregateMacros != want {
		t.Fatalf("aggregate = %+v, want %+v", created.Diet.AggregateMacros, want)
	}
	listed, err := service.List(context.Background(), userID)
	if err != nil || len(listed) != 1 || listed[0].ID != created.Diet.ID {
		t.Fatalf("List() diets=%+v error=%v", listed, err)
	}
	replaced, err := service.Replace(context.Background(), userID, created.Diet.ID, ReplaceRequest{Name: "Rest Day", Entries: []MealQuantity{{MealID: mealB, Quantity: 100, Unit: "g", Position: 0}}})
	if err != nil || replaced.Name != "Rest Day" || replaced.AggregateMacros != (MacroProjection{Protein: 5, Carbohydrates: 5, Fat: 2, Calories: 58}) {
		t.Fatalf("Replace() diet=%+v error=%v", replaced, err)
	}
	if err := service.Delete(context.Background(), userID, created.Diet.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if err := service.Delete(context.Background(), userID, created.Diet.ID); err != nil {
		t.Fatalf("repeated Delete() error = %v", err)
	}
	if _, err := service.Get(context.Background(), userID, created.Diet.ID); !repository.IsKind(err, repository.ErrorKindNotFound) {
		t.Fatalf("Get() after Delete() error = %v, want not found", err)
	}

	// Re-create the same request to verify idempotency independently of the CRUD lifecycle.
	request.IdempotencyKey = "daily-diet-5"
	created, err = service.Create(context.Background(), userID, request)
	if err != nil {
		t.Fatalf("second Create() error = %v", err)
	}
	replayed, err := service.Create(context.Background(), userID, request)
	if err != nil {
		t.Fatalf("replay error = %v", err)
	}
	if !replayed.Replayed || replayed.Diet.ID != created.Diet.ID || diets.createCalls != 2 {
		t.Fatalf("replay = %+v createCalls=%d", replayed, diets.createCalls)
	}
	original := replayed.Diet
	meals.mu.Lock()
	changedMeal := meals.meals[mealA]
	changedMeal.MacrosPer100 = repository.MacroValues{Protein: 999, Carbohydrates: 999, Fat: 999}
	meals.meals[mealA] = changedMeal
	meals.mu.Unlock()
	if err := service.Delete(context.Background(), userID, created.Diet.ID); err != nil {
		t.Fatalf("Delete() before replay error = %v", err)
	}
	replayed, err = service.Create(context.Background(), userID, request)
	if err != nil || !reflect.DeepEqual(replayed.Diet, original) {
		t.Fatalf("immutable replay = %+v error=%v, want %+v", replayed.Diet, err, original)
	}

	request.Entries[0].Quantity = 101
	if _, err := service.Create(context.Background(), userID, request); err != ErrIdempotencyConflict {
		t.Fatalf("conflicting retry error = %v, want %v", err, ErrIdempotencyConflict)
	}
	if diets.createCalls != 2 {
		t.Fatalf("conflicting retry wrote a diet: createCalls=%d", diets.createCalls)
	}
}

func TestServiceCreateLooksUpEachDistinctMealOnceAtMaximumEntries(t *testing.T) {
	userID, mealID := uuid.New(), uuid.New()
	meals := &memoryMealRepository{meals: map[uuid.UUID]repository.MealEntity{mealID: {ID: mealID, PhysicalState: repository.PhysicalStateSolid}}}
	entries := make([]MealQuantity, 100)
	for index := range entries {
		entries[index] = MealQuantity{MealID: mealID, Quantity: float64(index + 1), Unit: "g", Position: index}
	}
	if _, err := NewService(&memoryDietRepository{}, meals).Create(context.Background(), userID, CreateRequest{Name: "Maximum", IdempotencyKey: "maximum-entries", Entries: entries}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	meals.mu.Lock()
	defer meals.mu.Unlock()
	if meals.calls[mealID] != 1 {
		t.Fatalf("meal lookups = %d, want 1", meals.calls[mealID])
	}
}

func TestServiceCreateSameKeyIsAtomicAcrossInstances(t *testing.T) {
	userID, mealID := uuid.New(), uuid.New()
	diets := &memoryDietRepository{}
	meals := &memoryMealRepository{meals: map[uuid.UUID]repository.MealEntity{mealID: {ID: mealID, PhysicalState: repository.PhysicalStateSolid}}}
	request := CreateRequest{Name: "Concurrent", IdempotencyKey: "same-key-concurrent", Entries: []MealQuantity{{MealID: mealID, Quantity: 1, Unit: "g", Position: 0}}}
	services := []*Service{NewService(diets, meals), NewService(diets, meals)}
	start := make(chan struct{})
	results := make(chan CreateResult, 2)
	errors := make(chan error, 2)
	var wait sync.WaitGroup
	for _, service := range services {
		wait.Add(1)
		go func(service *Service) {
			defer wait.Done()
			<-start
			result, err := service.Create(context.Background(), userID, request)
			results <- result
			errors <- err
		}(service)
	}
	close(start)
	wait.Wait()
	close(results)
	close(errors)
	var id uuid.UUID
	for err := range errors {
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}
	for result := range results {
		if id == uuid.Nil {
			id = result.Diet.ID
		} else if result.Diet.ID != id {
			t.Fatalf("atomic responses differ: %s and %s", id, result.Diet.ID)
		}
	}
	if diets.createCalls != 1 {
		t.Fatalf("create calls = %d, want 1", diets.createCalls)
	}
}

func TestServiceCreateDoesNotBlockIndependentUsersAndHonorsCancellation(t *testing.T) {
	slowMeal, fastMeal := uuid.New(), uuid.New()
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	meals := &memoryMealRepository{
		meals: map[uuid.UUID]repository.MealEntity{
			slowMeal: {ID: slowMeal, PhysicalState: repository.PhysicalStateSolid},
			fastMeal: {ID: fastMeal, PhysicalState: repository.PhysicalStateSolid},
		},
		blockID: slowMeal, started: started, release: release,
	}
	service := NewService(&memoryDietRepository{}, meals)
	ctx, cancel := context.WithCancel(context.Background())
	slowDone := make(chan error, 1)
	go func() {
		_, err := service.Create(ctx, uuid.New(), CreateRequest{Name: "Slow", IdempotencyKey: "slow-user-key", Entries: []MealQuantity{{MealID: slowMeal, Quantity: 1, Unit: "g", Position: 0}}})
		slowDone <- err
	}()
	<-started
	fastDone := make(chan error, 1)
	go func() {
		_, err := service.Create(context.Background(), uuid.New(), CreateRequest{Name: "Fast", IdempotencyKey: "fast-user-key", Entries: []MealQuantity{{MealID: fastMeal, Quantity: 1, Unit: "g", Position: 0}}})
		fastDone <- err
	}()
	select {
	case err := <-fastDone:
		if err != nil {
			t.Fatalf("independent create error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("independent user was blocked behind unrelated create")
	}
	cancel()
	select {
	case err := <-slowDone:
		if err != context.Canceled {
			t.Fatalf("cancelled create error = %v, want context canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("cancelled create remained blocked")
	}
}

func TestServiceRejectsMissingMealsBeforeWritesAndScopesOwnership(t *testing.T) {
	userID, otherUserID := uuid.New(), uuid.New()
	missingMeal := uuid.New()
	diets := &memoryDietRepository{}
	service := NewService(diets, &memoryMealRepository{meals: map[uuid.UUID]repository.MealEntity{}})
	_, err := service.Create(context.Background(), userID, CreateRequest{Name: "Missing", IdempotencyKey: "daily-diet-2", Entries: []MealQuantity{{MealID: missingMeal, Quantity: 1, Unit: "g", Position: 0}}})
	if !repository.IsKind(err, repository.ErrorKindNotFound) {
		t.Fatalf("missing meal error = %v, want not found", err)
	}
	if diets.createCalls != 0 {
		t.Fatalf("missing meal caused %d writes", diets.createCalls)
	}

	mealID := uuid.New()
	diets = &memoryDietRepository{}
	meals := &memoryMealRepository{meals: map[uuid.UUID]repository.MealEntity{mealID: {ID: mealID, PhysicalState: repository.PhysicalStateSolid}}}
	service = NewService(diets, meals)
	created, err := service.Create(context.Background(), userID, CreateRequest{Name: "Owned", IdempotencyKey: "daily-diet-3", Entries: []MealQuantity{{MealID: mealID, Quantity: 1, Unit: "g", Position: 0}}})
	if err != nil {
		t.Fatalf("owned Create() error = %v", err)
	}
	if _, err := service.Get(context.Background(), otherUserID, created.Diet.ID); !repository.IsKind(err, repository.ErrorKindNotFound) {
		t.Fatalf("cross-user Get() error = %v, want not found", err)
	}
	if _, err := service.Replace(context.Background(), otherUserID, created.Diet.ID, ReplaceRequest{Name: "Nope", Entries: []MealQuantity{{MealID: mealID, Quantity: 2, Unit: "g", Position: 0}}}); !repository.IsKind(err, repository.ErrorKindNotFound) {
		t.Fatalf("cross-user Replace() error = %v, want not found", err)
	}
	if err := service.Delete(context.Background(), otherUserID, created.Diet.ID); !repository.IsKind(err, repository.ErrorKindNotFound) {
		t.Fatalf("cross-user Delete() error = %v, want not found", err)
	}
	if diets.replaceCalls != 0 || diets.deleteCalls != 0 {
		t.Fatalf("cross-user mutation counts replace=%d delete=%d", diets.replaceCalls, diets.deleteCalls)
	}
}

func TestServiceListReturnsNotFoundWhenSavedMealIsUnavailable(t *testing.T) {
	userID, dietID, mealID := uuid.New(), uuid.New(), uuid.New()
	diets := &memoryDietRepository{diets: map[uuid.UUID]repository.SavedDiet{
		dietID: {ID: dietID, UserID: userID, Name: "Unavailable meal", Entries: []repository.SavedDietMealEntry{{MealID: mealID, Quantity: 100, Unit: "g", Position: 0}}},
	}}

	_, err := NewService(diets, &memoryMealRepository{meals: map[uuid.UUID]repository.MealEntity{}}).List(context.Background(), userID)

	if !repository.IsKind(err, repository.ErrorKindNotFound) {
		t.Fatalf("List() error = %v, want not found", err)
	}
}

func TestServiceValidationRejectsInvalidInputs(t *testing.T) {
	service := NewService(&memoryDietRepository{}, &memoryMealRepository{})
	mealID := uuid.New()
	tests := []CreateRequest{
		{Name: "Diet", Entries: []MealQuantity{{MealID: mealID, Quantity: 1, Unit: "g", Position: 0}}},
		{Name: "Diet", IdempotencyKey: "short", Entries: []MealQuantity{{MealID: mealID, Quantity: 1, Unit: "g", Position: 0}}},
		{Name: "Diet", IdempotencyKey: "daily-diet-4", Entries: []MealQuantity{{MealID: mealID, Quantity: 1, Unit: "ml", Position: 0}}},
	}
	for _, request := range tests {
		if _, err := service.Create(context.Background(), uuid.New(), request); err == nil {
			t.Fatalf("Create() accepted invalid request %+v", request)
		}
	}
}

func copyEntries(entries []repository.SavedDietMealEntry, dietID uuid.UUID) []repository.SavedDietMealEntry {
	result := make([]repository.SavedDietMealEntry, len(entries))
	copy(result, entries)
	for index := range result {
		result[index].SavedDietID = dietID
		if result[index].ID == uuid.Nil {
			result[index].ID = uuid.New()
		}
	}
	return result
}

var _ repository.DailyDietMutationRepository = (*memoryDietRepository)(nil)
var _ repository.MealRepository = (*memoryMealRepository)(nil)
