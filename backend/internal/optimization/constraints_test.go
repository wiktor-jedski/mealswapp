// Implements DESIGN-004 ConstraintBuilder verification.
package optimization

import (
	"math"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

var (
	constraintMealA = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	constraintMealB = uuid.MustParse("00000000-0000-0000-0000-000000000002")
	constraintMealC = uuid.MustParse("00000000-0000-0000-0000-000000000003")
)

func TestBuildConstraintsUsesScaledRepositoryMacrosAndToleranceBands(t *testing.T) {
	model, err := BuildConstraints(DietOptimizationRequest{
		TargetMacros:     MacroTarget{Protein: 100, Carbohydrates: 200, Fat: 50},
		TolerancePercent: 10,
		MaxQuantity:      1000,
	}, []repository.MealEntity{
		{ID: constraintMealB, MacrosPer100: repository.MacroValues{Protein: 20, Carbohydrates: 40, Fat: 5}},
		{ID: constraintMealA, MacrosPer100: repository.MacroValues{Protein: 10, Carbohydrates: 30, Fat: 2}},
	})
	if err != nil {
		t.Fatalf("BuildConstraints() error = %v", err)
	}

	if got, want := []string{model.Variables[0].ItemID, model.Variables[1].ItemID}, []string{constraintMealA.String(), constraintMealB.String()}; !reflect.DeepEqual(got, want) {
		t.Fatalf("variable order = %v, want %v", got, want)
	}
	if got, want := model.Variables[0].ProteinPerUnit, 0.1; got != want {
		t.Fatalf("protein coefficient = %v, want %v", got, want)
	}
	if got, want := model.Variables[1].CarbohydratesPerUnit, 0.4; got != want {
		t.Fatalf("carbohydrate coefficient = %v, want %v", got, want)
	}

	assertConstraintBounds(t, model.Constraints[0], 90, 110)
	assertConstraintBounds(t, model.Constraints[1], 180, 220)
	assertConstraintBounds(t, model.Constraints[2], 45, 55)
	if got := model.Constraints[0].Coefficients[constraintMealA.String()]; got != 0.1 {
		t.Fatalf("protein matrix coefficient = %v, want 0.1", got)
	}
}

func TestBuildConstraintsExcludesMealsWithZeroEligibility(t *testing.T) {
	model, err := BuildConstraints(DietOptimizationRequest{
		TargetMacros:     MacroTarget{Protein: 10, Carbohydrates: 10, Fat: 10},
		TolerancePercent: 5,
		ExcludedMealIDs:  []uuid.UUID{constraintMealB},
		MaxQuantity:      500,
	}, []repository.MealEntity{
		{ID: constraintMealA, MacrosPer100: repository.MacroValues{Protein: 1, Carbohydrates: 1, Fat: 1}},
		{ID: constraintMealB, MacrosPer100: repository.MacroValues{Protein: 2, Carbohydrates: 2, Fat: 2}},
	})
	if err != nil {
		t.Fatalf("BuildConstraints() error = %v", err)
	}

	variable := model.Variables[1]
	if variable.ItemID != constraintMealB.String() || variable.UpperBound != 0 {
		t.Fatalf("excluded variable = %+v, want zero upper bound", variable)
	}
	constraint := findConstraint(t, model, "exclude_"+constraintMealB.String())
	assertConstraintBounds(t, constraint, 0, 0)
	if got := constraint.Coefficients[constraintMealB.String()]; got != 1 {
		t.Fatalf("exclusion coefficient = %v, want 1", got)
	}
}

func TestBuildConstraintsDerivesTargetFromPersistedOriginalDiet(t *testing.T) {
	model, err := BuildConstraints(DietOptimizationRequest{
		OriginalDiet: repository.SavedDiet{Entries: []repository.SavedDietMealEntry{
			{MealID: constraintMealA, Quantity: 150, Unit: "g", Position: 0},
			{MealID: constraintMealB, Quantity: 50, Unit: "g", Position: 1},
		}},
		TolerancePercent: 0,
		MaxQuantity:      1000,
	}, []repository.MealEntity{
		{ID: constraintMealA, PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{Protein: 20, Carbohydrates: 10, Fat: 2}},
		{ID: constraintMealB, PhysicalState: repository.PhysicalStateSolid, MacrosPer100: repository.MacroValues{Protein: 10, Carbohydrates: 40, Fat: 8}},
	})
	if err != nil {
		t.Fatalf("BuildConstraints() error = %v", err)
	}

	assertConstraintBounds(t, model.Constraints[0], 35, 35)
	assertConstraintBounds(t, model.Constraints[1], 35, 35)
	assertConstraintBounds(t, model.Constraints[2], 7, 7)
}

func TestBuildConstraintsAddsDeterministicAlternativeConstraint(t *testing.T) {
	req := DietOptimizationRequest{
		TargetMacros:     MacroTarget{Protein: 10, Carbohydrates: 10, Fat: 10},
		TolerancePercent: 5,
		MaxQuantity:      500,
		PreviousSolutions: []map[string]float64{{
			constraintMealB.String(): 100,
			constraintMealA.String(): 25,
		}},
	}
	meals := []repository.MealEntity{
		{ID: constraintMealB, MacrosPer100: repository.MacroValues{Protein: 2, Carbohydrates: 2, Fat: 2}},
		{ID: constraintMealA, MacrosPer100: repository.MacroValues{Protein: 1, Carbohydrates: 1, Fat: 1}},
	}

	first, err := BuildConstraints(req, meals)
	if err != nil {
		t.Fatalf("first BuildConstraints() error = %v", err)
	}
	reversed, err := BuildConstraints(req, []repository.MealEntity{meals[1], meals[0]})
	if err != nil {
		t.Fatalf("reversed BuildConstraints() error = %v", err)
	}
	if !reflect.DeepEqual(first, reversed) {
		t.Fatalf("constraint matrix changes with candidate order:\nfirst=%+v\nreversed=%+v", first, reversed)
	}

	alternative := findConstraint(t, first, "alternative_1")
	assertConstraintBounds(t, alternative, 0, 1.95)
	if len(alternative.Coefficients) != 2 {
		t.Fatalf("alternative coefficients = %v, want both selected meals", alternative.Coefficients)
	}
	if got, want := alternative.Coefficients[constraintMealA.String()], 1.0/25; got != want {
		t.Fatalf("alternative coefficient for A = %v, want %v", got, want)
	}
	if got, want := alternative.Coefficients[constraintMealB.String()], 1.0/100; got != want {
		t.Fatalf("alternative coefficient for B = %v, want %v", got, want)
	}

	nearDuplicate := map[string]float64{constraintMealA.String(): 24.999, constraintMealB.String(): 100}
	if got := constraintValue(alternative, nearDuplicate); got <= alternative.UpperBound {
		t.Fatalf("near-duplicate previous solution value = %v, want > %v", got, alternative.UpperBound)
	}
	differentMealSet := map[string]float64{constraintMealA.String(): 0, constraintMealB.String(): 100}
	if got := constraintValue(alternative, differentMealSet); got > alternative.UpperBound {
		t.Fatalf("distinct solution value = %v, want <= %v", got, alternative.UpperBound)
	}
}

func TestBuildConstraintsMatrixFixturesAreDeterministicAndClassifiable(t *testing.T) {
	fixtures := []struct {
		name       string
		req        DietOptimizationRequest
		meals      []repository.MealEntity
		assignment map[string]float64
		feasible   bool
	}{
		{
			name: "feasible intersection",
			req: DietOptimizationRequest{
				TargetMacros:     MacroTarget{Protein: 10, Carbohydrates: 20, Fat: 5},
				TolerancePercent: 0,
				MaxQuantity:      100,
			},
			meals: []repository.MealEntity{
				{ID: constraintMealB, MacrosPer100: repository.MacroValues{Protein: 5, Carbohydrates: 10, Fat: 2.5}},
				{ID: constraintMealA, MacrosPer100: repository.MacroValues{Protein: 10, Carbohydrates: 20, Fat: 5}},
			},
			assignment: map[string]float64{constraintMealA.String(): 100},
			feasible:   true,
		},
		{
			name: "bounded infeasible intersection",
			req: DietOptimizationRequest{
				TargetMacros:     MacroTarget{Protein: 2, Carbohydrates: 2, Fat: 2},
				TolerancePercent: 0,
				MaxQuantity:      10,
			},
			meals: []repository.MealEntity{
				{ID: constraintMealC, MacrosPer100: repository.MacroValues{Protein: 10, Carbohydrates: 10, Fat: 10}},
			},
			feasible: false,
		},
	}

	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			model, err := BuildConstraints(fixture.req, fixture.meals)
			if err != nil {
				t.Fatalf("BuildConstraints() error = %v", err)
			}
			reversed := append([]repository.MealEntity(nil), fixture.meals...)
			for left, right := 0, len(reversed)-1; left < right; left, right = left+1, right-1 {
				reversed[left], reversed[right] = reversed[right], reversed[left]
			}
			reversedModel, err := BuildConstraints(fixture.req, reversed)
			if err != nil {
				t.Fatalf("BuildConstraints(reversed) error = %v", err)
			}
			if !reflect.DeepEqual(model, reversedModel) {
				t.Fatalf("matrix is not deterministic:\nmodel=%+v\nreversed=%+v", model, reversedModel)
			}

			gotFeasible := false
			if fixture.assignment != nil {
				gotFeasible = modelAccepts(model, fixture.assignment)
			} else {
				gotFeasible = matrixCanReachLowerBounds(model)
			}
			if gotFeasible != fixture.feasible {
				t.Fatalf("fixture feasibility = %v, want %v", gotFeasible, fixture.feasible)
			}
		})
	}
}

func TestBuildConstraintsRejectsInvalidOrNonFiniteInputs(t *testing.T) {
	tests := []struct {
		name string
		req  DietOptimizationRequest
		meal repository.MealEntity
	}{
		{name: "non finite target", req: DietOptimizationRequest{TargetMacros: MacroTarget{Protein: math.NaN()}}, meal: validConstraintMeal()},
		{name: "negative target", req: DietOptimizationRequest{TargetMacros: MacroTarget{Protein: -1}}, meal: validConstraintMeal()},
		{name: "non finite tolerance", req: DietOptimizationRequest{TargetMacros: MacroTarget{Protein: 1}, TolerancePercent: math.Inf(1)}, meal: validConstraintMeal()},
		{name: "negative tolerance", req: DietOptimizationRequest{TargetMacros: MacroTarget{Protein: 1}, TolerancePercent: -1}, meal: validConstraintMeal()},
		{name: "non finite meal macro", req: DietOptimizationRequest{TargetMacros: MacroTarget{Protein: 1}}, meal: repository.MealEntity{ID: constraintMealA, MacrosPer100: repository.MacroValues{Protein: math.Inf(1)}}},
		{name: "non finite quantity bound", req: DietOptimizationRequest{TargetMacros: MacroTarget{Protein: 1}, MaxQuantity: math.Inf(1)}, meal: validConstraintMeal()},
		{name: "non finite prior quantity", req: DietOptimizationRequest{TargetMacros: MacroTarget{Protein: 1}, PreviousSolutions: []map[string]float64{{constraintMealA.String(): math.Inf(1)}}}, meal: validConstraintMeal()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := BuildConstraints(tt.req, []repository.MealEntity{tt.meal}); err == nil {
				t.Fatal("BuildConstraints() accepted invalid input")
			}
		})
	}
}

func validConstraintMeal() repository.MealEntity {
	return repository.MealEntity{ID: constraintMealA, MacrosPer100: repository.MacroValues{Protein: 10, Carbohydrates: 10, Fat: 1}}
}

func assertConstraintBounds(t *testing.T, constraint LPConstraint, lower, upper float64) {
	t.Helper()
	if constraint.LowerBound != lower || constraint.UpperBound != upper {
		t.Fatalf("%s bounds = [%v, %v], want [%v, %v]", constraint.Name, constraint.LowerBound, constraint.UpperBound, lower, upper)
	}
}

func findConstraint(t *testing.T, model LPModel, name string) LPConstraint {
	t.Helper()
	for _, constraint := range model.Constraints {
		if constraint.Name == name {
			return constraint
		}
	}
	t.Fatalf("constraint %q not found in %+v", name, model.Constraints)
	return LPConstraint{}
}

func constraintValue(constraint LPConstraint, quantities map[string]float64) float64 {
	value := 0.0
	for itemID, coefficient := range constraint.Coefficients {
		value += coefficient * quantities[itemID]
	}
	return value
}

func modelAccepts(model LPModel, quantities map[string]float64) bool {
	for _, variable := range model.Variables {
		quantity := quantities[variable.ItemID]
		if quantity < variable.LowerBound || quantity > variable.UpperBound {
			return false
		}
	}
	for _, constraint := range model.Constraints {
		value := constraintValue(constraint, quantities)
		if value < constraint.LowerBound-1e-9 || value > constraint.UpperBound+1e-9 {
			return false
		}
	}
	return true
}

func matrixCanReachLowerBounds(model LPModel) bool {
	for _, constraint := range model.Constraints {
		maximum := 0.0
		for _, variable := range model.Variables {
			coefficient := constraint.Coefficients[variable.ItemID]
			if coefficient > 0 {
				maximum += coefficient * variable.UpperBound
			}
		}
		if maximum < constraint.LowerBound-1e-9 {
			return false
		}
	}
	return true
}
