// Package optimization contains the pure linear-programming model construction
// used by the Phase 07 worker.
package optimization

import (
	"context"
	"fmt"
	"math"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/search"
)

// Implements DESIGN-004 ConstraintBuilder.
const (
	// MaximumMealQuantity bounds each recommendation in its repository nutrition
	// basis: grams for solids and millilitres for liquids.
	MaximumMealQuantity = 10_000
	mealSearchPageSize  = 100
)

// MacroTarget identifies the daily protein, carbohydrate, and fat targets.
// Implements DESIGN-004 ConstraintBuilder.
type MacroTarget struct {
	Protein       float64
	Carbohydrates float64
	Fat           float64
}

// MealQuantity identifies one persisted meal quantity in a Daily Diet.
// Implements DESIGN-004 ConstraintBuilder and DESIGN-008 SavedDataRepository.
type MealQuantity struct {
	MealID   uuid.UUID
	Name     string
	Quantity float64
	Unit     string
	Position int
}

// DietOptimizationRequest carries only the server-owned saved-diet inputs
// needed to build a constraint matrix.
// Implements DESIGN-004 ConstraintBuilder.
type DietOptimizationRequest struct {
	OriginalDiet     repository.SavedDiet
	TolerancePercent float64
	ExcludedMealIDs  []uuid.UUID
}

// LPVariable is one nonnegative meal quantity and its server-derived macro,
// calorie, and soft-diversity coefficients per repository base unit.
// Implements DESIGN-004 ConstraintBuilder.
type LPVariable struct {
	ItemID               string
	LowerBound           float64
	UpperBound           float64
	CaloriesPerUnit      float64
	DiversityPenalty     float64
	ProteinPerUnit       float64
	CarbohydratesPerUnit float64
	FatPerUnit           float64
}

// LPConstraint is a bounded linear expression over LP variables.
// Implements DESIGN-004 ConstraintBuilder.
type LPConstraint struct {
	Name         string
	LowerBound   float64
	UpperBound   float64
	Coefficients map[string]float64
}

// LPModel is the deterministic matrix consumed by the later solver wrapper.
// Implements DESIGN-004 ConstraintBuilder.
type LPModel struct {
	Variables   []LPVariable
	Constraints []LPConstraint
}

// ConstraintBuilder loads persisted meals and delegates pure model assembly.
// Implements DESIGN-004 ConstraintBuilder.
type ConstraintBuilder struct {
	meals repository.MealRepository
	diets repository.DailyDietRepository
	foods repository.FoodItemRepository
}

// SavedDietOptimizationInputs contains the server-owned diet snapshot and all
// eligible repository meals used by the worker's alternative generator.
// Implements DESIGN-004 ConstraintBuilder and JobQueueManager.
type SavedDietOptimizationInputs struct {
	Request DietOptimizationRequest
	Meals   []repository.MealEntity
}

// NewConstraintBuilder creates a repository-backed constraint builder.
// Implements DESIGN-004 ConstraintBuilder.
func NewConstraintBuilder(meals repository.MealRepository, diets repository.DailyDietRepository, foods ...repository.FoodItemRepository) *ConstraintBuilder {
	builder := &ConstraintBuilder{meals: meals, diets: diets}
	if len(foods) > 0 {
		builder.foods = foods[0]
	}
	return builder
}

// BuildConstraints constructs variables and hard constraints in stable meal-ID
// order. It performs all numeric validation before returning a model.
// Implements DESIGN-004 ConstraintBuilder.
func BuildConstraints(req DietOptimizationRequest, meals []repository.MealEntity, previousSolutions []LPSolution) (LPModel, error) {
	byID, ids, err := immutableMealSnapshot(meals, nil)
	if err != nil {
		return LPModel{}, validationError(err.Error())
	}
	return buildConstraintsFromIndex(req, byID, ids, previousSolutions)
}

// buildConstraintsFromIndex builds a model from the generation call's single
// detached meal index and deterministic ID order.
// Implements DESIGN-004 ConstraintBuilder and SolutionValidator.
func buildConstraintsFromIndex(req DietOptimizationRequest, byID map[string]repository.MealEntity, ids []string, previousSolutions []LPSolution) (LPModel, error) {
	if err := validateRequest(req); err != nil {
		return LPModel{}, err
	}
	if len(byID) == 0 {
		return LPModel{}, validationError("at least one repository meal is required")
	}

	target, err := targetForRequest(req, byID)
	if err != nil {
		return LPModel{}, err
	}

	excluded, err := excludedMealIDs(req)
	if err != nil {
		return LPModel{}, err
	}
	original := originalMealIDs(req.OriginalDiet)
	sourceOnlyFoodItems := originalFoodItemIDs(req.OriginalDiet)
	variables := make([]LPVariable, 0, len(byID))
	for _, itemID := range ids {
		meal, ok := byID[itemID]
		if !ok {
			return LPModel{}, validationError("meal index order is invalid")
		}
		if _, isExcluded := excluded[meal.ID]; isExcluded {
			continue
		}
		if _, sourceOnly := sourceOnlyFoodItems[meal.ID]; sourceOnly {
			continue
		}
		if err := validateMeal(meal); err != nil {
			if _, isOriginal := original[meal.ID]; isOriginal {
				return LPModel{}, validationError("original diet meal has no usable nutrition basis")
			}
			continue
		}
		macros := meal.MacrosPer100
		caloriesPerUnit := search.CalculateCalories(macros) / 100
		if !finite(caloriesPerUnit) || caloriesPerUnit < 0 {
			return LPModel{}, validationError("meal calorie coefficient must be finite and non-negative")
		}
		if caloriesPerUnit == 0 {
			if _, isOriginal := original[meal.ID]; isOriginal {
				return LPModel{}, validationError("original diet meal has no objective information")
			}
			continue
		}
		variables = append(variables, LPVariable{
			ItemID:               itemID,
			LowerBound:           0,
			UpperBound:           MaximumMealQuantity,
			CaloriesPerUnit:      caloriesPerUnit,
			ProteinPerUnit:       macros.Protein / 100,
			CarbohydratesPerUnit: macros.Carbohydrates / 100,
			FatPerUnit:           macros.Fat / 100,
		})
	}
	if len(variables) == 0 {
		return LPModel{}, validationError("no eligible repository meals are available")
	}
	variables, err = NewDiversityPenalizer(req).Apply(variables)
	if err != nil {
		return LPModel{}, err
	}

	constraints := []LPConstraint{
		macroConstraint("protein", target.Protein, req.TolerancePercent, variables, func(variable LPVariable) float64 { return variable.ProteinPerUnit }),
		macroConstraint("carbohydrate", target.Carbohydrates, req.TolerancePercent, variables, func(variable LPVariable) float64 { return variable.CarbohydratesPerUnit }),
		macroConstraint("fat", target.Fat, req.TolerancePercent, variables, func(variable LPVariable) float64 { return variable.FatPerUnit }),
	}
	for _, constraint := range constraints {
		if !finite(constraint.LowerBound) || !finite(constraint.UpperBound) {
			return LPModel{}, validationError("macro constraint bounds must be finite")
		}
	}
	alternativeConstraints, err := buildAlternativeConstraints(previousSolutions, byID, excluded)
	if err != nil {
		return LPModel{}, err
	}
	constraints = append(constraints, alternativeConstraints...)

	return LPModel{Variables: variables, Constraints: constraints}, nil
}

// BuildFromSavedDiet reloads the persisted Daily Diet and all repository meals
// under the authenticated owner before building the matrix. Macro totals are
// therefore derived from current repository data, never from caller totals.
// Implements DESIGN-004 ConstraintBuilder and DESIGN-008 SavedDataRepository.
func (b *ConstraintBuilder) BuildFromSavedDiet(ctx context.Context, userID, dietID uuid.UUID, req DietOptimizationRequest) (LPModel, error) {
	inputs, err := b.LoadFromSavedDiet(ctx, userID, dietID, req)
	if err != nil {
		return LPModel{}, err
	}
	return BuildConstraints(inputs.Request, inputs.Meals, nil)
}

// LoadFromSavedDiet reloads one owned saved diet and all repository meals for
// worker-side validation and repeated alternative generation. Macro totals are
// derived from this server-owned snapshot, never from queue payload totals.
// Implements DESIGN-004 ConstraintBuilder and JobQueueManager.
func (b *ConstraintBuilder) LoadFromSavedDiet(ctx context.Context, userID, dietID uuid.UUID, req DietOptimizationRequest) (SavedDietOptimizationInputs, error) {
	if b == nil || b.meals == nil || b.diets == nil {
		return SavedDietOptimizationInputs{}, validationError("constraint builder repositories are required")
	}
	if userID == uuid.Nil {
		return SavedDietOptimizationInputs{}, validationError("user id is required")
	}
	if dietID == uuid.Nil {
		return SavedDietOptimizationInputs{}, validationError("saved diet id is required")
	}

	diet, err := b.diets.Get(ctx, userID, dietID)
	if err != nil {
		return SavedDietOptimizationInputs{}, err
	}
	if diet.ID != dietID {
		return SavedDietOptimizationInputs{}, validationError("saved diet id does not match requested id")
	}
	if diet.UserID != userID {
		return SavedDietOptimizationInputs{}, validationError("saved diet owner does not match authenticated owner")
	}
	if len(diet.Entries) == 0 {
		return SavedDietOptimizationInputs{}, validationError("persisted original diet must contain at least one meal")
	}

	unitSystem := repository.UnitSystemMetric
	owner := userID
	context := repository.RepositoryContext{UserID: &owner, UnitSystem: unitSystem}
	meals := make([]repository.MealEntity, 0)
	seen := make(map[uuid.UUID]struct{})
	for index, entry := range diet.Entries {
		objectID, objectType := entry.FoodObjectID, entry.FoodObjectType
		if objectID == uuid.Nil && entry.MealID != uuid.Nil {
			objectID, objectType = entry.MealID, repository.FoodObjectTypeMeal
		}
		if _, ok := seen[objectID]; ok {
			continue
		}
		var meal repository.MealEntity
		if objectType == repository.FoodObjectTypeFoodItem {
			if b.foods == nil {
				return SavedDietOptimizationInputs{}, validationError("Food Item repository is required for this saved diet")
			}
			food, err := b.foods.GetByID(ctx, objectID, context)
			if err != nil {
				return SavedDietOptimizationInputs{}, err
			}
			meal = repository.MealEntity{ID: food.ID, Type: repository.MealTypeSingle, Name: food.Name, PhysicalState: food.PhysicalState, MacrosPer100: food.MacrosPer100, NormalizedMacrosAvailable: true}
		} else {
			var err error
			meal, err = b.meals.GetByID(ctx, objectID, context)
			if err != nil {
				return SavedDietOptimizationInputs{}, err
			}
		}
		if err := validateMeal(meal); err != nil {
			return SavedDietOptimizationInputs{}, validationError("original diet meal has no usable nutrition basis")
		}
		meals = append(meals, meal)
		seen[objectID] = struct{}{}
		diet.Entries[index].MealID = objectID
	}

	for offset := 0; ; offset += mealSearchPageSize {
		matches, total, err := b.meals.Search(ctx, repository.RepositoryQuery{
			RepositoryContext: context,
			FoodObjectTypes:   []repository.PhysicalState{repository.PhysicalStateSolid, repository.PhysicalStateLiquid},
			Limit:             mealSearchPageSize,
			Offset:            offset,
		})
		if err != nil {
			return SavedDietOptimizationInputs{}, err
		}
		for _, meal := range matches {
			if _, ok := seen[meal.ID]; ok {
				continue
			}
			if err := validateMeal(meal); err != nil {
				continue
			}
			meals = append(meals, meal)
			seen[meal.ID] = struct{}{}
		}
		if total <= offset+len(matches) || len(matches) == 0 {
			break
		}
	}

	req.OriginalDiet = diet
	return SavedDietOptimizationInputs{Request: req, Meals: meals}, nil
}

// validateRequest validates request-wide numeric bounds before matrix assembly.
// Implements DESIGN-004 ConstraintBuilder.
func validateRequest(req DietOptimizationRequest) error {
	if req.OriginalDiet.ID == uuid.Nil {
		return validationError("saved diet id is required")
	}
	if req.OriginalDiet.UserID == uuid.Nil {
		return validationError("saved diet owner is required")
	}
	if len(req.OriginalDiet.Entries) == 0 {
		return validationError("persisted original diet must contain at least one meal")
	}
	if !finite(req.TolerancePercent) || req.TolerancePercent < 0 || req.TolerancePercent > 100 {
		return validationError("tolerance percent must be finite and between 0 and 100")
	}
	_, err := excludedMealIDs(req)
	return err
}

// validateMeal validates the repository macro basis used by one variable.
// Implements DESIGN-004 ConstraintBuilder.
func validateMeal(meal repository.MealEntity) error {
	if meal.PhysicalState != repository.PhysicalStateSolid && meal.PhysicalState != repository.PhysicalStateLiquid {
		return validationError("meal physical state is not supported")
	}
	if !meal.NormalizedMacrosAvailable {
		return validationError("meal normalized macro basis is unavailable")
	}
	return repository.ValidateMacrosPer100(meal.MacrosPer100, meal.PhysicalState)
}

// targetForRequest derives authoritative macro targets from original meals.
// Implements DESIGN-004 ConstraintBuilder.
func targetForRequest(req DietOptimizationRequest, meals map[string]repository.MealEntity) (MacroTarget, error) {
	entries := req.OriginalDiet.Entries
	target := MacroTarget{}
	for _, entry := range entries {
		objectID := entry.MealID
		if entry.FoodObjectID != uuid.Nil {
			objectID = entry.FoodObjectID
		}
		if objectID == uuid.Nil {
			return MacroTarget{}, validationError("original diet Food Object id is required")
		}
		meal, ok := meals[objectID.String()]
		if !ok {
			return MacroTarget{}, validationError("original diet Food Object is not available: " + objectID.String())
		}
		if err := validateMeal(meal); err != nil {
			return MacroTarget{}, validationError("original diet meal has no usable nutrition basis")
		}
		baseQuantity, err := quantityInNutritionBasis(entry, meal)
		if err != nil {
			return MacroTarget{}, err
		}
		factor := baseQuantity / 100
		target.Protein += meal.MacrosPer100.Protein * factor
		target.Carbohydrates += meal.MacrosPer100.Carbohydrates * factor
		target.Fat += meal.MacrosPer100.Fat * factor
	}
	if err := validateMacroTarget(target); err != nil {
		return MacroTarget{}, err
	}
	if target == (MacroTarget{}) {
		return MacroTarget{}, validationError("saved diet macro target cannot be all zero")
	}
	return target, nil
}

// quantityInNutritionBasis converts a persisted original-diet quantity to the
// meal's per-100 g or per-100 ml repository macro basis.
// Implements DESIGN-004 ConstraintBuilder and DESIGN-005 UnitConverter.
func quantityInNutritionBasis(entry repository.SavedDietMealEntry, meal repository.MealEntity) (float64, error) {
	if !finite(entry.Quantity) || entry.Quantity <= 0 {
		return 0, validationError("original diet quantities must be finite and positive")
	}

	baseUnit := ""
	switch meal.PhysicalState {
	case repository.PhysicalStateSolid:
		if entry.Unit != "g" && entry.Unit != "oz" {
			return 0, validationError("solid meal quantities must use g or oz")
		}
		baseUnit = "g"
	case repository.PhysicalStateLiquid:
		if entry.Unit != "ml" && entry.Unit != "fl_oz" {
			return 0, validationError("liquid meal quantities must use ml or fl_oz")
		}
		baseUnit = "ml"
	default:
		return 0, validationError("original diet meal physical state is invalid")
	}

	quantity, err := repository.ConvertUnit(entry.Quantity, entry.Unit, baseUnit)
	if err != nil {
		return 0, validationError("original diet quantity unit conversion failed")
	}
	if !finite(quantity) || quantity <= 0 {
		return 0, validationError("converted original diet quantity must be finite and positive")
	}
	return quantity, nil
}

// validateMacroTarget validates the canonical macro target values.
// Implements DESIGN-004 ConstraintBuilder.
func validateMacroTarget(target MacroTarget) error {
	if !finite(target.Protein) || !finite(target.Carbohydrates) || !finite(target.Fat) {
		return validationError("macro targets must be finite")
	}
	if target.Protein < 0 || target.Carbohydrates < 0 || target.Fat < 0 {
		return validationError("macro targets cannot be negative")
	}
	return nil
}

// macroConstraint creates one bounded macro expression over LP variables.
// Implements DESIGN-004 ConstraintBuilder.
func macroConstraint(name string, target, tolerance float64, variables []LPVariable, coefficient func(LPVariable) float64) LPConstraint {
	margin := target * tolerance / 100
	coefficients := make(map[string]float64, len(variables))
	for _, variable := range variables {
		coefficients[variable.ItemID] = coefficient(variable)
	}
	return LPConstraint{
		Name:         name,
		LowerBound:   target - margin,
		UpperBound:   target + margin,
		Coefficients: coefficients,
	}
}

// excludedMealIDs validates and indexes the one typed exclusion representation.
// Implements DESIGN-004 ConstraintBuilder.
func excludedMealIDs(req DietOptimizationRequest) (map[uuid.UUID]struct{}, error) {
	excluded := make(map[uuid.UUID]struct{}, len(req.ExcludedMealIDs))
	for _, id := range req.ExcludedMealIDs {
		if id == uuid.Nil {
			return nil, validationError("excluded meal id is required")
		}
		if _, duplicate := excluded[id]; duplicate {
			return nil, validationError("excluded meal ids must be unique")
		}
		excluded[id] = struct{}{}
	}
	return excluded, nil
}

// originalMealIDs indexes the authoritative saved-diet meal set.
// Implements DESIGN-004 ConstraintBuilder.
func originalMealIDs(diet repository.SavedDiet) map[uuid.UUID]struct{} {
	result := make(map[uuid.UUID]struct{}, len(diet.Entries))
	for _, entry := range diet.Entries {
		if entry.FoodObjectID != uuid.Nil {
			result[entry.FoodObjectID] = struct{}{}
		} else if entry.MealID != uuid.Nil {
			result[entry.MealID] = struct{}{}
		}
	}
	return result
}

// originalFoodItemIDs indexes source Food Items that cannot become generated Meal alternatives.
// Implements DESIGN-004 ConstraintBuilder.
func originalFoodItemIDs(diet repository.SavedDiet) map[uuid.UUID]struct{} {
	result := make(map[uuid.UUID]struct{})
	for _, entry := range diet.Entries {
		if entry.FoodObjectType == repository.FoodObjectTypeFoodItem && entry.FoodObjectID != uuid.Nil {
			result[entry.FoodObjectID] = struct{}{}
		}
	}
	return result
}

// buildAlternativeConstraints excludes the highest-quantity selected meal
// from each prior solution. This deterministic bounded heuristic guarantees a
// changed meal-ID set; quantity drift alone can never satisfy it.
// Implements DESIGN-004 ConstraintBuilder and DiversityPenalizer.
func buildAlternativeConstraints(solutions []LPSolution, meals map[string]repository.MealEntity, excluded map[uuid.UUID]struct{}) ([]LPConstraint, error) {
	constraints := make([]LPConstraint, 0, len(solutions))
	for index, solution := range solutions {
		selectedID := ""
		selectedQuantity := 0.0
		for itemID, quantity := range solution {
			meal, ok := meals[itemID]
			if !ok {
				return nil, validationError("alternative solution contains unknown meal: " + itemID)
			}
			if !finite(quantity) || quantity < 0 {
				return nil, validationError("alternative solution quantities must be finite and non-negative")
			}
			if _, isExcluded := excluded[meal.ID]; isExcluded && quantity > 0 {
				return nil, validationError("alternative solution selects an excluded meal: " + itemID)
			}
			if quantity == 0 {
				continue
			}
			if quantity > selectedQuantity || quantity == selectedQuantity && (selectedID == "" || itemID < selectedID) {
				selectedID = itemID
				selectedQuantity = quantity
			}
		}
		if len(solution) > 0 && selectedID == "" {
			return nil, validationError("alternative solution must select a positive finite quantity")
		}
		if selectedID == "" {
			continue
		}
		constraints = append(constraints, LPConstraint{
			Name:         fmt.Sprintf("alternative_%d", index+1),
			LowerBound:   0,
			UpperBound:   0,
			Coefficients: map[string]float64{selectedID: 1},
		})
	}
	return constraints, nil
}

// validationError creates the package's typed model-input validation failure.
// Implements DESIGN-004 ConstraintBuilder.
func validationError(message string) error {
	return repository.NewError(repository.ErrorKindValidation, message, nil)
}

// finite reports whether a solver-facing numeric value is finite.
// Implements DESIGN-004 ConstraintBuilder.
func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
