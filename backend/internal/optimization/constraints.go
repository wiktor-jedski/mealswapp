// Package optimization contains the pure linear-programming model construction
// used by the Phase 07 worker.
package optimization

import (
	"context"
	"fmt"
	"math"
	"sort"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/search"
)

// Implements DESIGN-004 ConstraintBuilder.
const (
	// DefaultMaxQuantity bounds an unconstrained meal variable in its repository
	// base unit (g for solids and ml for liquids).
	DefaultMaxQuantity = 1_000_000
	// alternativeOverlapLoss requires each repeated solution to lose a material
	// amount of normalized overlap with its previously selected meals.
	alternativeOverlapLoss = 0.05
	mealSearchPageSize     = 100
)

// MacroTarget identifies the daily protein, carbohydrate, and fat targets.
// Implements DESIGN-004 ConstraintBuilder.
type MacroTarget struct {
	Protein       float64
	Carbohydrates float64
	Fat           float64
	// Carbs is accepted as the short form used by DESIGN-004's language model
	// contract. Carbohydrates remains the canonical Go field.
	Carbs float64
}

// MealQuantity identifies one persisted meal quantity in a Daily Diet.
// Implements DESIGN-004 ConstraintBuilder and DESIGN-008 SavedDataRepository.
type MealQuantity struct {
	MealID   uuid.UUID
	Quantity float64
	Unit     string
	Position int
}

// DietOptimizationRequest carries the validated inputs needed to build a
// constraint matrix. When OriginalDiet or OriginalMeals is present, its
// server-side meal data derives the macro target and takes precedence over
// TargetMacros.
// Implements DESIGN-004 ConstraintBuilder.
type DietOptimizationRequest struct {
	OriginalDiet  repository.SavedDiet
	OriginalMeals []MealQuantity
	// RepositoryMeals is an optional dependency-injection field for direct
	// package-level solution validation. Production generation passes meals
	// explicitly to the validator.
	RepositoryMeals   []repository.MealEntity
	TargetMacros      MacroTarget
	TolerancePercent  float64
	ExcludedMealIDs   []uuid.UUID
	ExcludedIDs       []string
	MaxQuantity       float64
	PreviousSolutions []map[string]float64
}

// LPVariable is one nonnegative meal quantity and its server-derived macro,
// calorie, and soft-diversity coefficients per repository base unit.
// Implements DESIGN-004 ConstraintBuilder.
type LPVariable struct {
	ItemID               string
	MealID               uuid.UUID
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
func NewConstraintBuilder(meals repository.MealRepository, diets repository.DailyDietRepository) *ConstraintBuilder {
	return &ConstraintBuilder{meals: meals, diets: diets}
}

// BuildConstraints constructs variables and hard constraints in stable meal-ID
// order. It performs all numeric validation before returning a model.
// Implements DESIGN-004 ConstraintBuilder.
func BuildConstraints(req DietOptimizationRequest, meals []repository.MealEntity) (LPModel, error) {
	maxQuantity, err := validateRequest(req)
	if err != nil {
		return LPModel{}, err
	}

	byID := make(map[string]repository.MealEntity, len(meals))
	for _, meal := range meals {
		if meal.ID == uuid.Nil {
			return LPModel{}, validationError("meal id is required")
		}
		itemID := meal.ID.String()
		if _, exists := byID[itemID]; exists {
			return LPModel{}, validationError("duplicate meal id: " + itemID)
		}
		if err := validateMeal(meal); err != nil {
			return LPModel{}, err
		}
		byID[itemID] = meal
	}
	if len(byID) == 0 {
		return LPModel{}, validationError("at least one repository meal is required")
	}

	target, err := targetForRequest(req, byID)
	if err != nil {
		return LPModel{}, err
	}

	excluded := excludedMealIDs(req)
	variables := make([]LPVariable, 0, len(byID))
	ids := make([]string, 0, len(byID))
	for itemID := range byID {
		ids = append(ids, itemID)
	}
	sort.Strings(ids)
	for _, itemID := range ids {
		meal := byID[itemID]
		macros := meal.MacrosPer100
		upper := maxQuantity
		if excluded[itemID] {
			upper = 0
		}
		caloriesPerUnit := search.CalculateCalories(macros) / 100
		if !finite(caloriesPerUnit) || caloriesPerUnit < 0 {
			return LPModel{}, validationError("meal calorie coefficient must be finite and non-negative")
		}
		variables = append(variables, LPVariable{
			ItemID:               itemID,
			MealID:               meal.ID,
			LowerBound:           0,
			UpperBound:           upper,
			CaloriesPerUnit:      caloriesPerUnit,
			ProteinPerUnit:       macros.Protein / 100,
			CarbohydratesPerUnit: macros.Carbohydrates / 100,
			FatPerUnit:           macros.Fat / 100,
		})
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
	for _, variable := range variables {
		constraints = append(constraints, LPConstraint{
			Name:         "quantity_" + variable.ItemID,
			LowerBound:   variable.LowerBound,
			UpperBound:   variable.UpperBound,
			Coefficients: map[string]float64{variable.ItemID: 1},
		})
	}
	for _, itemID := range ids {
		if !excluded[itemID] {
			continue
		}
		constraints = append(constraints, LPConstraint{
			Name:         "exclude_" + itemID,
			LowerBound:   0,
			UpperBound:   0,
			Coefficients: map[string]float64{itemID: 1},
		})
	}

	alternativeConstraints, err := buildAlternativeConstraints(req.PreviousSolutions, byID, excluded)
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
	return BuildConstraints(inputs.Request, inputs.Meals)
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
	if diet.ID != uuid.Nil && diet.ID != dietID {
		return SavedDietOptimizationInputs{}, validationError("saved diet id does not match requested id")
	}

	unitSystem := repository.UnitSystemMetric
	owner := userID
	context := repository.RepositoryContext{UserID: &owner, UnitSystem: unitSystem}
	meals := make([]repository.MealEntity, 0)
	seen := make(map[uuid.UUID]struct{})
	for _, entry := range diet.Entries {
		if _, ok := seen[entry.MealID]; ok {
			continue
		}
		meal, err := b.meals.GetByID(ctx, entry.MealID, context)
		if err != nil {
			return SavedDietOptimizationInputs{}, err
		}
		meals = append(meals, meal)
		seen[entry.MealID] = struct{}{}
	}

	for offset := 0; ; offset += mealSearchPageSize {
		matches, total, err := b.meals.Search(ctx, repository.RepositoryQuery{
			RepositoryContext: context,
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
func validateRequest(req DietOptimizationRequest) (float64, error) {
	if err := validateMacroTarget(req.TargetMacros); err != nil {
		return 0, err
	}
	if !finite(req.TolerancePercent) || req.TolerancePercent < 0 || req.TolerancePercent > 100 {
		return 0, validationError("tolerance percent must be finite and between 0 and 100")
	}
	if req.MaxQuantity < 0 || !finite(req.MaxQuantity) {
		return 0, validationError("maximum meal quantity must be finite and non-negative")
	}
	if req.MaxQuantity == 0 {
		return DefaultMaxQuantity, nil
	}
	return req.MaxQuantity, nil
}

// validateMeal validates the repository macro basis used by one variable.
// Implements DESIGN-004 ConstraintBuilder.
func validateMeal(meal repository.MealEntity) error {
	if meal.PhysicalState != "" {
		if err := repository.ValidatePhysicalState(meal.PhysicalState); err != nil {
			return err
		}
		if err := repository.ValidateMacrosPer100(meal.MacrosPer100, meal.PhysicalState); err != nil {
			return err
		}
	} else if err := repository.ValidateMacros(meal.MacrosPer100); err != nil {
		return err
	}
	return nil
}

// targetForRequest derives authoritative macro targets from original meals.
// Implements DESIGN-004 ConstraintBuilder.
func targetForRequest(req DietOptimizationRequest, meals map[string]repository.MealEntity) (MacroTarget, error) {
	entries := req.OriginalDiet.Entries
	if req.OriginalDiet.ID != uuid.Nil && len(entries) == 0 {
		return MacroTarget{}, validationError("persisted original diet must contain at least one meal")
	}
	if len(entries) == 0 {
		entries = originalMealsToEntries(req.OriginalMeals)
	}
	if len(entries) == 0 {
		return canonicalTarget(req.TargetMacros)
	}

	target := MacroTarget{}
	for _, entry := range entries {
		if entry.MealID == uuid.Nil {
			return MacroTarget{}, validationError("original diet meal id is required")
		}
		meal, ok := meals[entry.MealID.String()]
		if !ok {
			return MacroTarget{}, validationError("original diet meal is not available: " + entry.MealID.String())
		}
		if err := validateMealQuantity(entry, meal); err != nil {
			return MacroTarget{}, err
		}
		factor := entry.Quantity / 100
		target.Protein += meal.MacrosPer100.Protein * factor
		target.Carbohydrates += meal.MacrosPer100.Carbohydrates * factor
		target.Fat += meal.MacrosPer100.Fat * factor
	}
	if err := validateMacroTarget(target); err != nil {
		return MacroTarget{}, err
	}
	return target, nil
}

// originalMealsToEntries converts the optimization request's meal shape.
// Implements DESIGN-004 ConstraintBuilder.
func originalMealsToEntries(meals []MealQuantity) []repository.SavedDietMealEntry {
	entries := make([]repository.SavedDietMealEntry, len(meals))
	for index, meal := range meals {
		entries[index] = repository.SavedDietMealEntry{MealID: meal.MealID, Quantity: meal.Quantity, Unit: meal.Unit, Position: index}
	}
	return entries
}

// validateMealQuantity validates a persisted original-diet quantity and unit.
// Implements DESIGN-004 ConstraintBuilder.
func validateMealQuantity(entry repository.SavedDietMealEntry, meal repository.MealEntity) error {
	if !finite(entry.Quantity) || entry.Quantity <= 0 {
		return validationError("original diet quantities must be finite and positive")
	}
	if entry.Unit != "g" && entry.Unit != "ml" {
		return validationError("original diet quantity unit must be g or ml")
	}
	if meal.PhysicalState == repository.PhysicalStateSolid && entry.Unit != "g" {
		return validationError("solid meal quantities must use g")
	}
	if meal.PhysicalState == repository.PhysicalStateLiquid && entry.Unit != "ml" {
		return validationError("liquid meal quantities must use ml")
	}
	return nil
}

// validateMacroTarget validates all macro target aliases and values.
// Implements DESIGN-004 ConstraintBuilder.
func validateMacroTarget(target MacroTarget) error {
	if !finite(target.Protein) || !finite(target.Carbohydrates) || !finite(target.Carbs) || !finite(target.Fat) {
		return validationError("macro targets must be finite")
	}
	if target.Protein < 0 || target.Carbohydrates < 0 || target.Carbs < 0 || target.Fat < 0 {
		return validationError("macro targets cannot be negative")
	}
	if target.Carbohydrates != 0 && target.Carbs != 0 && target.Carbohydrates != target.Carbs {
		return validationError("carbohydrate target fields disagree")
	}
	return nil
}

// canonicalTarget resolves the Carbs alias into the canonical field.
// Implements DESIGN-004 ConstraintBuilder.
func canonicalTarget(target MacroTarget) (MacroTarget, error) {
	if err := validateMacroTarget(target); err != nil {
		return MacroTarget{}, err
	}
	if target.Carbohydrates == 0 {
		target.Carbohydrates = target.Carbs
	}
	target.Carbs = target.Carbohydrates
	return target, nil
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

// excludedMealIDs combines UUID and string exclusion inputs.
// Implements DESIGN-004 ConstraintBuilder.
func excludedMealIDs(req DietOptimizationRequest) map[string]bool {
	excluded := make(map[string]bool, len(req.ExcludedMealIDs)+len(req.ExcludedIDs))
	for _, id := range req.ExcludedMealIDs {
		if id != uuid.Nil {
			excluded[id.String()] = true
		}
	}
	for _, id := range req.ExcludedIDs {
		if id != "" {
			excluded[id] = true
		}
	}
	return excluded
}

// buildAlternativeConstraints rejects high-overlap assignments from prior solves.
// Implements DESIGN-004 ConstraintBuilder and DiversityPenalizer.
func buildAlternativeConstraints(solutions []map[string]float64, meals map[string]repository.MealEntity, excluded map[string]bool) ([]LPConstraint, error) {
	constraints := make([]LPConstraint, 0, len(solutions))
	for index, solution := range solutions {
		ids := make([]string, 0, len(solution))
		for itemID, quantity := range solution {
			if _, ok := meals[itemID]; !ok {
				return nil, validationError("alternative solution contains unknown meal: " + itemID)
			}
			if !finite(quantity) || quantity < 0 {
				return nil, validationError("alternative solution quantities must be finite and non-negative")
			}
			if excluded[itemID] && quantity > 0 {
				return nil, validationError("alternative solution selects an excluded meal: " + itemID)
			}
			if quantity == 0 {
				continue
			}
			ids = append(ids, itemID)
		}
		if len(solution) > 0 && len(ids) == 0 {
			return nil, validationError("alternative solution must select a positive finite quantity")
		}
		if len(ids) == 0 {
			continue
		}
		sort.Strings(ids)
		coefficients := make(map[string]float64, len(ids))
		for _, itemID := range ids {
			coefficient := 1 / solution[itemID]
			if !finite(coefficient) {
				return nil, validationError("alternative solution coefficients must be finite")
			}
			coefficients[itemID] = coefficient
		}
		upper := float64(len(ids)) - alternativeOverlapLoss
		if upper < 0 || !finite(upper) {
			return nil, validationError("alternative solution bound is invalid")
		}
		constraints = append(constraints, LPConstraint{
			Name:         fmt.Sprintf("alternative_%d", index+1),
			LowerBound:   0,
			UpperBound:   upper,
			Coefficients: coefficients,
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
