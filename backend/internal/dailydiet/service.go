// Package dailydiet owns authenticated saved one-day diet collections.
package dailydiet

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// Implements DESIGN-008 ProfileController and SavedDataRepository.
const (
	maxEntries       = 100
	maxNameLength    = 120
	maxQuantity      = 1_000_000
	minimumKeyLength = 8
)

// ErrMissingIdempotencyKey means a daily-diet create was attempted without a key.
// Implements DESIGN-008 ProfileController daily-diet creation.
var ErrMissingIdempotencyKey = errors.New("idempotency key is required")

// ErrIdempotencyConflict means a key was reused with a different request body.
// Implements DESIGN-008 ProfileController daily-diet creation.
var ErrIdempotencyConflict = errors.New("idempotency key reused with different body")

// ErrDuplicateName means one user already owns a Daily Diet with the same normalized name.
// Implements DESIGN-008 SavedDataRepository user-scoped Daily Diet naming.
var ErrDuplicateName = errors.New("daily diet name already exists")

// FoodObjectQuantity is one client-selected Meal or Food Item and its canonical quantity.
// Implements DESIGN-008 SavedDataRepository daily-diet collection contract.
type FoodObjectQuantity struct {
	FoodObjectID   uuid.UUID                 `json:"foodObjectId"`
	FoodObjectType repository.FoodObjectType `json:"foodObjectType"`
	Quantity       float64                   `json:"quantity"`
	Unit           string                    `json:"unit"`
	Position       int                       `json:"position"`
	MealID         uuid.UUID                 `json:"-"`
}

// MealQuantity preserves source compatibility for internal Meal-only callers.
// Implements DESIGN-008 SavedDataRepository daily-diet collection contract.
type MealQuantity = FoodObjectQuantity

// CreateRequest contains only client-editable saved-diet fields.
// Implements DESIGN-008 SavedDataRepository daily-diet collection contract.
type CreateRequest struct {
	Name           string               `json:"name"`
	Entries        []FoodObjectQuantity `json:"entries"`
	IdempotencyKey string               `json:"-"`
}

// ReplaceRequest contains only client-editable replacement fields.
// Implements DESIGN-008 SavedDataRepository daily-diet collection contract.
type ReplaceRequest struct {
	Name    string               `json:"name"`
	Entries []FoodObjectQuantity `json:"entries"`
}

// DailyDiet is the API-safe saved-diet projection with server-derived totals.
// Implements DESIGN-008 SavedDataRepository daily-diet collection contract.
type DailyDiet struct {
	ID              uuid.UUID        `json:"id"`
	Name            string           `json:"name"`
	Entries         []DailyDietEntry `json:"entries"`
	AggregateMacros MacroProjection  `json:"aggregateMacros"`
	CreatedAt       time.Time        `json:"createdAt"`
	UpdatedAt       time.Time        `json:"updatedAt"`
}

// DailyDietEntry is one persisted meal entry returned by the API.
// Implements DESIGN-008 SavedDataRepository daily-diet collection contract.
type DailyDietEntry struct {
	ID             uuid.UUID                 `json:"id"`
	FoodObjectID   uuid.UUID                 `json:"foodObjectId"`
	FoodObjectType repository.FoodObjectType `json:"foodObjectType"`
	Quantity       float64                   `json:"quantity"`
	Unit           string                    `json:"unit"`
	Position       int                       `json:"position"`
	MealID         uuid.UUID                 `json:"-"`
}

// MacroProjection contains server-derived one-day totals.
// Implements DESIGN-008 SavedDataRepository daily-diet collection contract.
type MacroProjection struct {
	Protein       float64 `json:"protein"`
	Carbohydrates float64 `json:"carbohydrates"`
	Fat           float64 `json:"fat"`
	Calories      float64 `json:"calories"`
}

// CreateResult carries a created or replayed daily diet.
// Implements DESIGN-008 ProfileController daily-diet creation.
type CreateResult struct {
	Diet     DailyDiet
	Status   int
	Replayed bool
}

// Service coordinates saved-diet persistence, meal validation, and aggregation.
// Implements DESIGN-008 ProfileController and SavedDataRepository.
type Service struct {
	diets repository.DailyDietMutationRepository
	meals repository.MealRepository
	foods repository.FoodItemRepository
}

// NewService creates authenticated saved-diet behavior.
// Implements DESIGN-008 ProfileController and SavedDataRepository.
func NewService(diets repository.DailyDietMutationRepository, meals repository.MealRepository, foods ...repository.FoodItemRepository) *Service {
	service := &Service{diets: diets, meals: meals}
	if len(foods) > 0 {
		service.foods = foods[0]
	}
	return service
}

// Create persists a user-owned daily diet and replays exact idempotent retries.
// Implements DESIGN-008 ProfileController and SavedDataRepository.
func (s *Service) Create(ctx context.Context, userID uuid.UUID, req CreateRequest) (CreateResult, error) {
	if userID == uuid.Nil {
		return CreateResult{}, validationError("user id is required")
	}
	key, err := validateIdempotencyKey(req.IdempotencyKey)
	if err != nil {
		return CreateResult{}, err
	}
	name, err := normalizeRequest(req.Name, req.Entries)
	if err != nil {
		return CreateResult{}, err
	}
	if s == nil || s.diets == nil || s.meals == nil {
		return CreateResult{}, repository.NewError(repository.ErrorKindConnection, "daily diet service is unavailable", nil)
	}
	entries := normalizeFoodObjectEntries(req.Entries)
	bodyHash, err := requestHash(name, entries)
	if err != nil {
		return CreateResult{}, err
	}
	if result, err := s.diets.GetDailyDietCreateClaim(ctx, userID, key, bodyHash); err == nil {
		return createResultFromClaim(result), nil
	} else if !repository.IsKind(err, repository.ErrorKindNotFound) {
		if repository.IsKind(err, repository.ErrorKindConflict) {
			return CreateResult{}, ErrIdempotencyConflict
		}
		return CreateResult{}, err
	}

	diet, response, err := s.prepareCreate(ctx, userID, name, entries)
	if err != nil {
		return CreateResult{}, err
	}
	claimResult, err := s.diets.ClaimDailyDietCreate(ctx, repository.DailyDietCreateClaim{
		UserID: userID, Key: key, BodyHash: bodyHash, Diet: diet, Response: response, StatusCode: 201,
	})
	if err != nil {
		if repository.IsKind(err, repository.ErrorKindConflict) {
			var repositoryError *repository.Error
			if errors.As(err, &repositoryError) && strings.Contains(repositoryError.Message, "idempotency key") {
				return CreateResult{}, ErrIdempotencyConflict
			}
			return CreateResult{}, ErrDuplicateName
		}
		return CreateResult{}, err
	}
	return createResultFromClaim(claimResult), nil
}

// Get loads one user-owned daily diet and recalculates its aggregate totals.
// Implements DESIGN-008 ProfileController and SavedDataRepository.
func (s *Service) Get(ctx context.Context, userID, dietID uuid.UUID) (DailyDiet, error) {
	if err := validateIdentity(userID, dietID); err != nil {
		return DailyDiet{}, err
	}
	return s.load(ctx, userID, dietID)
}

// List loads all daily diets owned by one authenticated user.
// Implements DESIGN-008 ProfileController and SavedDataRepository.
func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]DailyDiet, error) {
	if userID == uuid.Nil {
		return nil, validationError("user id is required")
	}
	if s == nil || s.diets == nil || s.meals == nil {
		return nil, repository.NewError(repository.ErrorKindConnection, "daily diet service is unavailable", nil)
	}
	diets, err := s.diets.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make([]DailyDiet, 0, len(diets))
	for _, diet := range diets {
		projected, err := s.project(ctx, diet)
		if err != nil {
			return nil, err
		}
		result = append(result, projected)
	}
	return result, nil
}

// Replace atomically replaces one user-owned daily diet.
// Implements DESIGN-008 ProfileController and SavedDataRepository.
func (s *Service) Replace(ctx context.Context, userID, dietID uuid.UUID, req ReplaceRequest) (DailyDiet, error) {
	if err := validateIdentity(userID, dietID); err != nil {
		return DailyDiet{}, err
	}
	name, err := normalizeRequest(req.Name, req.Entries)
	if err != nil {
		return DailyDiet{}, err
	}
	entries := normalizeFoodObjectEntries(req.Entries)
	if err := s.validateFoodObjects(ctx, entries); err != nil {
		return DailyDiet{}, err
	}
	if s == nil || s.diets == nil || s.meals == nil {
		return DailyDiet{}, repository.NewError(repository.ErrorKindConnection, "daily diet service is unavailable", nil)
	}
	if _, err := s.diets.Get(ctx, userID, dietID); err != nil {
		return DailyDiet{}, err
	}
	if err := s.diets.Replace(ctx, userID, repository.SavedDiet{ID: dietID, Name: name, Entries: toRepositoryEntries(entries)}); err != nil {
		if repository.IsKind(err, repository.ErrorKindConflict) {
			return DailyDiet{}, ErrDuplicateName
		}
		return DailyDiet{}, err
	}
	return s.load(ctx, userID, dietID)
}

// Delete removes one user-owned daily diet.
// Implements DESIGN-008 ProfileController and SavedDataRepository.
func (s *Service) Delete(ctx context.Context, userID, dietID uuid.UUID) error {
	if err := validateIdentity(userID, dietID); err != nil {
		return err
	}
	if s == nil || s.diets == nil || s.meals == nil {
		return repository.NewError(repository.ErrorKindConnection, "daily diet service is unavailable", nil)
	}
	deleted, exists, err := s.diets.DeleteIfOwned(ctx, userID, dietID)
	if err != nil {
		return err
	}
	if !deleted && exists {
		return repository.NewError(repository.ErrorKindNotFound, "saved diet not found", nil)
	}
	return nil
}

// load reads a saved diet and projects current server-owned meal totals.
// Implements DESIGN-008 SavedDataRepository.
func (s *Service) load(ctx context.Context, userID, dietID uuid.UUID) (DailyDiet, error) {
	if s == nil || s.diets == nil || s.meals == nil {
		return DailyDiet{}, repository.NewError(repository.ErrorKindConnection, "daily diet service is unavailable", nil)
	}
	diet, err := s.diets.Get(ctx, userID, dietID)
	if err != nil {
		return DailyDiet{}, err
	}
	return s.project(ctx, diet)
}

// validateFoodObjects verifies every Meal or Food Item before any saved-diet write begins.
// Implements DESIGN-008 SavedDataRepository.
func (s *Service) validateFoodObjects(ctx context.Context, entries []FoodObjectQuantity) error {
	for _, entry := range entries {
		physicalState, _, err := s.foodObjectNutrition(ctx, entry.FoodObjectID, entry.FoodObjectType)
		if err != nil {
			return err
		}
		if _, err := quantityInMealBase(entry.Quantity, entry.Unit, physicalState); err != nil {
			return err
		}
	}
	return nil
}

// prepareCreate validates each distinct meal once and builds the immutable persisted projection.
// Implements DESIGN-008 SavedDataRepository durable create idempotency.
func (s *Service) prepareCreate(ctx context.Context, userID uuid.UUID, name string, entries []FoodObjectQuantity) (repository.SavedDiet, repository.DailyDietCreateResponse, error) {
	now := time.Now().UTC()
	dietID := uuid.New()
	diet := repository.SavedDiet{ID: dietID, UserID: userID, Name: name, CreatedAt: now, UpdatedAt: now, Entries: make([]repository.SavedDietMealEntry, 0, len(entries))}
	response := repository.DailyDietCreateResponse{ID: dietID, Name: name, CreatedAt: now, UpdatedAt: now, Entries: make([]repository.DailyDietCreateResponseEntry, 0, len(entries))}
	type nutrition struct {
		physicalState repository.PhysicalState
		macros        repository.MacroValues
	}
	objects := make(map[string]nutrition, len(entries))
	for _, entry := range entries {
		objectKey := string(entry.FoodObjectType) + ":" + entry.FoodObjectID.String()
		object, ok := objects[objectKey]
		if !ok {
			physicalState, macros, err := s.foodObjectNutrition(ctx, entry.FoodObjectID, entry.FoodObjectType)
			if err != nil {
				return repository.SavedDiet{}, repository.DailyDietCreateResponse{}, err
			}
			object = nutrition{physicalState: physicalState, macros: macros}
			objects[objectKey] = object
		}
		baseQuantity, err := quantityInMealBase(entry.Quantity, entry.Unit, object.physicalState)
		if err != nil {
			return repository.SavedDiet{}, repository.DailyDietCreateResponse{}, err
		}
		macros := repository.ScaleMacros(object.macros, baseQuantity, 100)
		response.AggregateMacros.Protein += macros.Protein
		response.AggregateMacros.Carbohydrates += macros.Carbohydrates
		response.AggregateMacros.Fat += macros.Fat
		entryID := uuid.New()
		diet.Entries = append(diet.Entries, repository.SavedDietMealEntry{ID: entryID, SavedDietID: dietID, FoodObjectID: entry.FoodObjectID, FoodObjectType: entry.FoodObjectType, Quantity: entry.Quantity, Unit: entry.Unit, Position: entry.Position, CreatedAt: now})
		response.Entries = append(response.Entries, repository.DailyDietCreateResponseEntry{ID: entryID, FoodObjectID: entry.FoodObjectID, FoodObjectType: entry.FoodObjectType, Quantity: entry.Quantity, Unit: entry.Unit, Position: entry.Position})
	}
	response.AggregateMacros.Protein = round4(response.AggregateMacros.Protein)
	response.AggregateMacros.Carbohydrates = round4(response.AggregateMacros.Carbohydrates)
	response.AggregateMacros.Fat = round4(response.AggregateMacros.Fat)
	response.AggregateMacros.Calories = round4(response.AggregateMacros.Protein*4 + response.AggregateMacros.Carbohydrates*4 + response.AggregateMacros.Fat*9)
	return diet, response, nil
}

// createResultFromClaim maps the repository's exact persisted response without mutable reloads.
// Implements DESIGN-008 SavedDataRepository durable create idempotency.
func createResultFromClaim(result repository.DailyDietCreateClaimResult) CreateResult {
	entries := make([]DailyDietEntry, len(result.Response.Entries))
	for index, entry := range result.Response.Entries {
		entries[index] = DailyDietEntry{ID: entry.ID, FoodObjectID: entry.FoodObjectID, FoodObjectType: entry.FoodObjectType, Quantity: entry.Quantity, Unit: entry.Unit, Position: entry.Position}
	}
	macros := result.Response.AggregateMacros
	return CreateResult{Diet: DailyDiet{ID: result.Response.ID, Name: result.Response.Name, Entries: entries, AggregateMacros: MacroProjection{Protein: macros.Protein, Carbohydrates: macros.Carbohydrates, Fat: macros.Fat, Calories: macros.Calories}, CreatedAt: result.Response.CreatedAt, UpdatedAt: result.Response.UpdatedAt}, Status: result.StatusCode, Replayed: result.Replayed}
}

// project maps persistence rows to the API projection and recalculates totals.
// Implements DESIGN-008 SavedDataRepository.
func (s *Service) project(ctx context.Context, diet repository.SavedDiet) (DailyDiet, error) {
	entries := make([]DailyDietEntry, 0, len(diet.Entries))
	projection := MacroProjection{}
	for _, entry := range diet.Entries {
		objectID, objectType := repositoryEntryFoodObject(entry)
		physicalState, objectMacros, err := s.foodObjectNutrition(ctx, objectID, objectType)
		if err != nil {
			return DailyDiet{}, err
		}
		baseQuantity, err := quantityInMealBase(entry.Quantity, entry.Unit, physicalState)
		if err != nil {
			return DailyDiet{}, err
		}
		macros := repository.ScaleMacros(objectMacros, baseQuantity, 100)
		projection.Protein += macros.Protein
		projection.Carbohydrates += macros.Carbohydrates
		projection.Fat += macros.Fat
		entries = append(entries, DailyDietEntry{ID: entry.ID, FoodObjectID: objectID, FoodObjectType: objectType, Quantity: entry.Quantity, Unit: entry.Unit, Position: entry.Position})
	}
	projection.Protein = round4(projection.Protein)
	projection.Carbohydrates = round4(projection.Carbohydrates)
	projection.Fat = round4(projection.Fat)
	projection.Calories = round4(projection.Protein*4 + projection.Carbohydrates*4 + projection.Fat*9)
	return DailyDiet{ID: diet.ID, Name: diet.Name, Entries: entries, AggregateMacros: projection, CreatedAt: diet.CreatedAt, UpdatedAt: diet.UpdatedAt}, nil
}

// repositoryEntryFoodObject resolves legacy Meal-only rows into the canonical Food Object identity.
// Implements DESIGN-008 SavedDataRepository daily-diet collection contract.
func repositoryEntryFoodObject(entry repository.SavedDietMealEntry) (uuid.UUID, repository.FoodObjectType) {
	if entry.FoodObjectID != uuid.Nil {
		return entry.FoodObjectID, entry.FoodObjectType
	}
	return entry.MealID, repository.FoodObjectTypeMeal
}

// normalizeRequest validates client-editable saved-diet fields.
// Implements DESIGN-008 SavedDataRepository.
func normalizeRequest(name string, entries []FoodObjectQuantity) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > maxNameLength || strings.ContainsRune(name, '\x00') {
		return "", validationError("daily diet name is invalid")
	}
	if len(entries) == 0 || len(entries) > maxEntries {
		return "", validationError("daily diet entries must contain between 1 and 100 meals")
	}
	seenPositions := make(map[int]struct{}, len(entries))
	for _, entry := range normalizeFoodObjectEntries(entries) {
		if entry.FoodObjectID == uuid.Nil || (entry.FoodObjectType != repository.FoodObjectTypeMeal && entry.FoodObjectType != repository.FoodObjectTypeFoodItem) {
			return "", validationError("saved diet Food Object identity is required")
		}
		if entry.Quantity <= 0 || entry.Quantity > maxQuantity || math.IsNaN(entry.Quantity) || math.IsInf(entry.Quantity, 0) {
			return "", validationError("saved diet meal quantity must be finite and positive")
		}
		if repository.ValidateQuantityUnit(entry.Unit) != nil {
			return "", validationError("saved diet meal unit is invalid")
		}
		if entry.Position < 0 || entry.Position >= maxEntries {
			return "", validationError("saved diet meal position is invalid")
		}
		if _, exists := seenPositions[entry.Position]; exists {
			return "", validationError("saved diet meal positions must be unique")
		}
		seenPositions[entry.Position] = struct{}{}
	}
	return name, nil
}

// normalizeFoodObjectEntries upgrades internal Meal-only aliases to canonical Food Object entries.
// Implements DESIGN-008 SavedDataRepository daily-diet collection contract.
func normalizeFoodObjectEntries(entries []FoodObjectQuantity) []FoodObjectQuantity {
	result := append([]FoodObjectQuantity(nil), entries...)
	for index := range result {
		if result[index].FoodObjectID == uuid.Nil && result[index].MealID != uuid.Nil {
			result[index].FoodObjectID = result[index].MealID
			result[index].FoodObjectType = repository.FoodObjectTypeMeal
		}
	}
	return result
}

// toRepositoryEntries maps API quantities to persistence entries.
// Implements DESIGN-008 SavedDataRepository.
func toRepositoryEntries(entries []FoodObjectQuantity) []repository.SavedDietMealEntry {
	result := make([]repository.SavedDietMealEntry, 0, len(entries))
	for _, entry := range entries {
		result = append(result, repository.SavedDietMealEntry{FoodObjectID: entry.FoodObjectID, FoodObjectType: entry.FoodObjectType, Quantity: entry.Quantity, Unit: entry.Unit, Position: entry.Position})
	}
	return result
}

// foodObjectNutrition resolves the authoritative nutrition basis for one Daily Diet entry.
// Implements DESIGN-005 FoodItemRepository/MealRepository and DESIGN-008 SavedDataRepository.
func (s *Service) foodObjectNutrition(ctx context.Context, id uuid.UUID, objectType repository.FoodObjectType) (repository.PhysicalState, repository.MacroValues, error) {
	switch objectType {
	case repository.FoodObjectTypeMeal:
		if s == nil || s.meals == nil {
			return "", repository.MacroValues{}, repository.NewError(repository.ErrorKindConnection, "meal service is unavailable", nil)
		}
		meal, err := s.meals.GetByID(ctx, id, repository.RepositoryContext{UnitSystem: repository.UnitSystemMetric})
		return meal.PhysicalState, meal.MacrosPer100, err
	case repository.FoodObjectTypeFoodItem:
		if s == nil || s.foods == nil {
			return "", repository.MacroValues{}, repository.NewError(repository.ErrorKindConnection, "Food Item service is unavailable", nil)
		}
		food, err := s.foods.GetByID(ctx, id, repository.RepositoryContext{UnitSystem: repository.UnitSystemMetric})
		return food.PhysicalState, food.MacrosPer100, err
	default:
		return "", repository.MacroValues{}, validationError("Food Object type is invalid")
	}
}

// quantityInMealBase converts a canonical quantity to the meal's metric basis.
// Implements DESIGN-005 UnitConverter and DESIGN-008 SavedDataRepository.
func quantityInMealBase(quantity float64, unit string, state repository.PhysicalState) (float64, error) {
	var base string
	switch state {
	case repository.PhysicalStateSolid:
		if unit != "g" && unit != "oz" {
			return 0, validationError("solid meal quantity must use g or oz")
		}
		base = "g"
	case repository.PhysicalStateLiquid:
		if unit != "ml" && unit != "fl_oz" {
			return 0, validationError("liquid meal quantity must use ml or fl_oz")
		}
		base = "ml"
	default:
		return 0, validationError("meal physical state is invalid")
	}
	converted, err := repository.ConvertUnit(quantity, unit, base)
	if err != nil {
		return 0, err
	}
	return converted, nil
}

// validateIdentity validates authenticated and resource identifiers.
// Implements DESIGN-008 ProfileController.
func validateIdentity(userID, dietID uuid.UUID) error {
	if userID == uuid.Nil {
		return validationError("user id is required")
	}
	if dietID == uuid.Nil {
		return validationError("saved diet id is required")
	}
	return nil
}

// validateIdempotencyKey validates the cross-phase mutation key.
// Implements DESIGN-008 ProfileController daily-diet idempotency.
func validateIdempotencyKey(value string) (string, error) {
	value = strings.TrimSpace(value)
	if len(value) < minimumKeyLength || len(value) > 255 {
		return "", ErrMissingIdempotencyKey
	}
	return value, nil
}

// requestHash hashes only server-accepted create fields.
// Implements DESIGN-008 ProfileController daily-diet idempotency.
func requestHash(name string, entries []FoodObjectQuantity) (string, error) {
	payload, err := json.Marshal(struct {
		Name    string               `json:"name"`
		Entries []FoodObjectQuantity `json:"entries"`
	}{Name: strings.TrimSpace(name), Entries: entries})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

// validationError creates a stable repository validation error.
// Implements DESIGN-008 SavedDataRepository.
func validationError(message string) error {
	return repository.NewError(repository.ErrorKindValidation, message, nil)
}

// round4 keeps aggregate projections stable across repeated reads.
// Implements DESIGN-008 SavedDataRepository.
func round4(value float64) float64 {
	return math.Round(value*10_000) / 10_000
}
