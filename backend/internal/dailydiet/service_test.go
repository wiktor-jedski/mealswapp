package dailydiet

// Implements DESIGN-008 ProfileController and SavedDataRepository verification.

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

type memoryDietRepository struct {
	diets         map[uuid.UUID]repository.SavedDiet
	idempotencies map[string]repository.CheckoutIdempotencyRecord
	createCalls   int
	replaceCalls  int
	deleteCalls   int
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

func (r *memoryDietRepository) CreateWithIdempotency(ctx context.Context, userID uuid.UUID, diet repository.SavedDiet, record repository.CheckoutIdempotencyRecord) (repository.AtomicDailyDietMutationResult, error) {
	if r.idempotencies == nil {
		r.idempotencies = map[string]repository.CheckoutIdempotencyRecord{}
	}
	key := record.UserID.String() + record.Method + record.Route + record.Key
	if existing, ok := r.idempotencies[key]; ok {
		if existing.BodyHash != record.BodyHash {
			return repository.AtomicDailyDietMutationResult{}, repository.NewError(repository.ErrorKindConflict, "idempotency key reused with different body", nil)
		}
		var reference struct {
			DailyDietID uuid.UUID `json:"dailyDietId"`
		}
		if err := json.Unmarshal(existing.ResponseBody, &reference); err != nil {
			return repository.AtomicDailyDietMutationResult{}, err
		}
		return repository.AtomicDailyDietMutationResult{DietID: reference.DailyDietID, Idempotency: existing, Replayed: true}, nil
	}
	id, err := r.Create(ctx, userID, diet)
	if err != nil {
		return repository.AtomicDailyDietMutationResult{}, err
	}
	r.idempotencies[key] = record
	return repository.AtomicDailyDietMutationResult{DietID: id, Idempotency: record}, nil
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
	meals map[uuid.UUID]repository.MealEntity
}

func (r *memoryMealRepository) GetByID(_ context.Context, id uuid.UUID, _ repository.RepositoryContext) (repository.MealEntity, error) {
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

type memoryIdempotencyRepository struct {
	records map[string]repository.CheckoutIdempotencyRecord
}

func (r *memoryIdempotencyRepository) GetCheckoutIdempotency(_ context.Context, userID uuid.UUID, method, route, key string) (repository.CheckoutIdempotencyRecord, error) {
	record, ok := r.records[userID.String()+method+route+key]
	if !ok {
		return repository.CheckoutIdempotencyRecord{}, repository.NewError(repository.ErrorKindNotFound, "missing", nil)
	}
	return record, nil
}

func (r *memoryIdempotencyRepository) StoreCheckoutIdempotency(_ context.Context, record repository.CheckoutIdempotencyRecord) error {
	if r.records == nil {
		r.records = map[string]repository.CheckoutIdempotencyRecord{}
	}
	key := record.UserID.String() + record.Method + record.Route + record.Key
	if _, exists := r.records[key]; exists {
		return repository.NewError(repository.ErrorKindConflict, "idempotency key exists", nil)
	}
	r.records[key] = record
	return nil
}

func TestServiceCreateAggregatesMultipleMealsAndReplaysIdempotently(t *testing.T) {
	userID := uuid.New()
	mealA, mealB := uuid.New(), uuid.New()
	meals := &memoryMealRepository{meals: map[uuid.UUID]repository.MealEntity{
		mealA: {ID: mealA, PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{Protein: 10, Carbohydrates: 20, Fat: 5}},
		mealB: {ID: mealB, PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{Protein: 5, Carbohydrates: 5, Fat: 2}},
	}}
	diets := &memoryDietRepository{}
	service := NewService(diets, meals, &memoryIdempotencyRepository{})
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

	request.Entries[0].Quantity = 101
	if _, err := service.Create(context.Background(), userID, request); err != ErrIdempotencyConflict {
		t.Fatalf("conflicting retry error = %v, want %v", err, ErrIdempotencyConflict)
	}
	if diets.createCalls != 2 {
		t.Fatalf("conflicting retry wrote a diet: createCalls=%d", diets.createCalls)
	}
}

func TestServiceRejectsMissingMealsBeforeWritesAndScopesOwnership(t *testing.T) {
	userID, otherUserID := uuid.New(), uuid.New()
	missingMeal := uuid.New()
	diets := &memoryDietRepository{}
	service := NewService(diets, &memoryMealRepository{meals: map[uuid.UUID]repository.MealEntity{}}, &memoryIdempotencyRepository{})
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
	service = NewService(diets, meals, &memoryIdempotencyRepository{})
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

func TestServiceValidationRejectsInvalidInputs(t *testing.T) {
	service := NewService(&memoryDietRepository{}, &memoryMealRepository{}, &memoryIdempotencyRepository{})
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
var _ repository.CheckoutIdempotencyRepository = (*memoryIdempotencyRepository)(nil)
