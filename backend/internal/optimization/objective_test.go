// Implements DESIGN-004 ObjectiveFunction.
package optimization

import (
	"math"
	"testing"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/search"
)

var (
	objectiveMealA = uuid.MustParse("00000000-0000-0000-0000-000000000011")
	objectiveMealB = uuid.MustParse("00000000-0000-0000-0000-000000000012")
	objectiveMealC = uuid.MustParse("00000000-0000-0000-0000-000000000013")
)

func TestBuildObjectiveRanksFeasibleFixturesByServerCalories(t *testing.T) {
	meals := []repository.MealEntity{
		objectiveMeal(objectiveMealB, 2),
		objectiveMeal(objectiveMealA, 1),
	}
	model, err := BuildConstraints(objectiveRequest(meals[0], 100, nil), meals, nil)
	if err != nil {
		t.Fatalf("BuildConstraints() error = %v", err)
	}
	policy, err := BuildObjective(model.Variables)
	if err != nil {
		t.Fatalf("BuildObjective() error = %v", err)
	}

	lowCalorie := map[string]float64{objectiveMealA.String(): 100}
	highCalorie := map[string]float64{objectiveMealB.String(): 100}
	if !objectiveFixtureAccepts(model, lowCalorie) || !objectiveFixtureAccepts(model, highCalorie) {
		t.Fatal("objective fixtures must both satisfy the LP constraints")
	}
	if got, want := objectiveValue(policy.Primary, lowCalorie), 89.0; got != want {
		t.Fatalf("low-calorie objective value = %v, want %v", got, want)
	}
	if got, want := objectiveValue(policy.Primary, highCalorie), 98.0; math.Abs(got-want) > 1e-12 {
		t.Fatalf("high-calorie objective value = %v, want %v", got, want)
	}
	if objectiveValue(policy.Primary, lowCalorie) >= objectiveValue(policy.Primary, highCalorie) {
		t.Fatal("objective did not rank the lower-calorie feasible fixture first")
	}
	if got := objectiveValue(policy.Secondary, highCalorie); got != 100 {
		t.Fatalf("original-meal secondary value = %v, want 100 base units", got)
	}
	if got := objectiveValue(policy.Secondary, lowCalorie); got != 0 {
		t.Fatalf("unrelated-meal secondary value = %v, want 0", got)
	}
}

func TestBuildObjectiveUsesServerCalculatedCaloriesAndStableEqualTies(t *testing.T) {
	meals := []repository.MealEntity{
		objectiveMeal(objectiveMealB, 1),
		objectiveMeal(objectiveMealA, 1),
		objectiveMeal(objectiveMealC, 1),
	}
	model, err := BuildConstraints(objectiveRequest(meals[2], 0, []uuid.UUID{objectiveMealC}), meals, nil)
	if err != nil {
		t.Fatalf("BuildConstraints() error = %v", err)
	}
	policy, err := BuildObjective(model.Variables)
	if err != nil {
		t.Fatalf("BuildObjective() error = %v", err)
	}

	if got, want := policy.Primary.Coefficients[objectiveMealA.String()], search.CalculateCalories(meals[1].MacrosPer100)/100; got != want {
		t.Fatalf("server calorie coefficient = %v, want %v", got, want)
	}
	if got, want := policy.Primary.Coefficients[objectiveMealB.String()], policy.Primary.Coefficients[objectiveMealA.String()]; got != want {
		t.Fatalf("equal calorie coefficients = %v and %v, want equal", got, want)
	}
	if len(policy.Primary.Coefficients) != 2 || len(policy.Secondary.Coefficients) != 2 {
		t.Fatalf("objective policy field sizes = primary %d, secondary %d; want 2 each", len(policy.Primary.Coefficients), len(policy.Secondary.Coefficients))
	}
}

func objectiveMeal(id uuid.UUID, fat float64) repository.MealEntity {
	return repository.MealEntity{
		ID: id, Type: repository.MealTypeSingle, PhysicalState: repository.PhysicalStateSolid,
		MacrosPer100:              repository.MacroValues{Protein: 10, Carbohydrates: 10, Fat: fat},
		NormalizedMacrosAvailable: true,
	}
}

func objectiveRequest(original repository.MealEntity, tolerance float64, excluded []uuid.UUID) DietOptimizationRequest {
	return DietOptimizationRequest{
		OriginalDiet: repository.SavedDiet{
			ID:      uuid.MustParse("00000000-0000-4000-8000-000000000014"),
			UserID:  uuid.MustParse("00000000-0000-4000-8000-000000000015"),
			Entries: []repository.SavedDietMealEntry{{MealID: original.ID, Quantity: 100, Unit: "g"}},
		},
		TolerancePercent: tolerance,
		ExcludedMealIDs:  excluded,
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
		{name: "invalid diversity", variable: LPVariable{ItemID: objectiveMealA.String(), CaloriesPerUnit: 1, DiversityPenalty: math.NaN()}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := BuildObjective([]LPVariable{tt.variable}); err == nil {
				t.Fatal("BuildObjective() accepted invalid calorie coefficient")
			}
		})
	}
}

func TestObjectivePolicyFieldsDriveDistinctSerializedObjectives(t *testing.T) {
	variables := []LPVariable{
		{ItemID: objectiveMealA.String(), LowerBound: 0, UpperBound: 1, CaloriesPerUnit: 1, DiversityPenalty: 1},
		{ItemID: objectiveMealB.String(), LowerBound: 0, UpperBound: 1, CaloriesPerUnit: 2},
	}
	policy, err := BuildObjective(variables)
	if err != nil {
		t.Fatalf("BuildObjective() error = %v", err)
	}
	model := LPModel{Variables: variables, Constraints: []LPConstraint{{Name: "quantity", LowerBound: 1, UpperBound: 1, Coefficients: map[string]float64{objectiveMealA.String(): 1, objectiveMealB.String(): 1}}}}
	primary, _, err := serializeLP(model, policy.Primary)
	if err != nil {
		t.Fatalf("serialize primary: %v", err)
	}
	secondary, _, err := serializeLP(model, policy.Secondary)
	if err != nil {
		t.Fatalf("serialize secondary: %v", err)
	}
	if string(primary) == string(secondary) {
		t.Fatal("primary and secondary objective fields serialized identically")
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
	for itemID, coefficient := range objective.Coefficients {
		value += coefficient * quantities[itemID]
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
