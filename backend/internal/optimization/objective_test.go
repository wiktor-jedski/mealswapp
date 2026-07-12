// Implements DESIGN-004 ObjectiveFunction.
package optimization

import (
	"math"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/search"
)

var (
	objectiveMealA = uuid.MustParse("00000000-0000-0000-0000-000000000011")
	objectiveMealB = uuid.MustParse("00000000-0000-0000-0000-000000000012")
)

func TestBuildObjectiveRanksFeasibleFixturesByServerCalories(t *testing.T) {
	meals := []repository.MealEntity{
		{ID: objectiveMealB, MacrosPer100: repository.MacroValues{Protein: 10, Carbohydrates: 10, Fat: 2}},
		{ID: objectiveMealA, MacrosPer100: repository.MacroValues{Protein: 10, Carbohydrates: 10, Fat: 1}},
	}
	model, err := BuildConstraints(DietOptimizationRequest{
		TargetMacros:     MacroTarget{Protein: 10, Carbohydrates: 10, Fat: 1},
		TolerancePercent: 100,
		MaxQuantity:      100,
	}, meals)
	if err != nil {
		t.Fatalf("BuildConstraints() error = %v", err)
	}
	objective, err := BuildObjective(model.Variables)
	if err != nil {
		t.Fatalf("BuildObjective() error = %v", err)
	}

	lowCalorie := map[string]float64{objectiveMealA.String(): 100}
	highCalorie := map[string]float64{objectiveMealB.String(): 100}
	if !objectiveFixtureAccepts(model, lowCalorie) || !objectiveFixtureAccepts(model, highCalorie) {
		t.Fatal("objective fixtures must both satisfy the LP constraints")
	}
	if got, want := objectiveValue(objective, lowCalorie), 89.0; got != want {
		t.Fatalf("low-calorie objective value = %v, want %v", got, want)
	}
	if got, want := objectiveValue(objective, highCalorie), 98.0; got != want {
		t.Fatalf("high-calorie objective value = %v, want %v", got, want)
	}
	if objectiveValue(objective, lowCalorie) >= objectiveValue(objective, highCalorie) {
		t.Fatal("objective did not rank the lower-calorie feasible fixture first")
	}
}

func TestBuildObjectiveUsesServerCalculatedCaloriesAndStableEqualTies(t *testing.T) {
	meals := []repository.MealEntity{
		{ID: objectiveMealB, MacrosPer100: repository.MacroValues{Protein: 10, Carbohydrates: 10, Fat: 1}},
		{ID: objectiveMealA, MacrosPer100: repository.MacroValues{Protein: 10, Carbohydrates: 10, Fat: 1}},
	}
	model, err := BuildConstraints(DietOptimizationRequest{
		TargetMacros:     MacroTarget{Protein: 10, Carbohydrates: 10, Fat: 1},
		TolerancePercent: 0,
		MaxQuantity:      100,
	}, meals)
	if err != nil {
		t.Fatalf("BuildConstraints() error = %v", err)
	}
	objective, err := BuildObjective(model.Variables)
	if err != nil {
		t.Fatalf("BuildObjective() error = %v", err)
	}

	if got, want := objective.Coefficients[objectiveMealA.String()], search.CalculateCalories(meals[1].MacrosPer100)/100; got != want {
		t.Fatalf("server calorie coefficient = %v, want %v", got, want)
	}
	if got, want := objective.Coefficients[objectiveMealB.String()], objective.Coefficients[objectiveMealA.String()]; got != want {
		t.Fatalf("equal calorie coefficients = %v and %v, want equal", got, want)
	}
	if want := []string{objectiveMealA.String(), objectiveMealB.String()}; !reflect.DeepEqual(objective.VariableIDs, want) {
		t.Fatalf("objective variable order = %v, want %v", objective.VariableIDs, want)
	}
}

func TestBuildObjectiveRejectsMissingInvalidAndNegativeCoefficients(t *testing.T) {
	tests := []struct {
		name     string
		variable LPVariable
	}{
		{name: "missing", variable: LPVariable{ItemID: objectiveMealA.String()}},
		{name: "non finite", variable: LPVariable{ItemID: objectiveMealA.String(), CaloriesPerUnit: math.NaN()}},
		{name: "infinite", variable: LPVariable{ItemID: objectiveMealA.String(), CaloriesPerUnit: math.Inf(1)}},
		{name: "negative", variable: LPVariable{ItemID: objectiveMealA.String(), CaloriesPerUnit: -1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := BuildObjective([]LPVariable{tt.variable}); err == nil {
				t.Fatal("BuildObjective() accepted invalid calorie coefficient")
			}
		})
	}
}

func TestBuildObjectiveRejectsEmptyAndDuplicateVariables(t *testing.T) {
	if _, err := BuildObjective(nil); err == nil {
		t.Fatal("BuildObjective() accepted an empty variable list")
	}
	if _, err := BuildObjective([]LPVariable{{CaloriesPerUnit: 1}}); err == nil {
		t.Fatal("BuildObjective() accepted a variable without an item ID")
	}
	variable := LPVariable{ItemID: objectiveMealA.String(), CaloriesPerUnit: 1}
	if _, err := BuildObjective([]LPVariable{variable, variable}); err == nil {
		t.Fatal("BuildObjective() accepted duplicate item IDs")
	}
}

func objectiveValue(objective ObjectiveFunction, quantities map[string]float64) float64 {
	value := 0.0
	for _, itemID := range objective.VariableIDs {
		value += objective.Coefficients[itemID] * quantities[itemID]
	}
	return value
}

func objectiveFixtureAccepts(model LPModel, quantities map[string]float64) bool {
	for _, variable := range model.Variables {
		quantity := quantities[variable.ItemID]
		if quantity < variable.LowerBound || quantity > variable.UpperBound {
			return false
		}
	}
	for _, constraint := range model.Constraints {
		value := 0.0
		for itemID, coefficient := range constraint.Coefficients {
			value += coefficient * quantities[itemID]
		}
		if value < constraint.LowerBound-1e-9 || value > constraint.UpperBound+1e-9 {
			return false
		}
	}
	return true
}
