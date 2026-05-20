package optimization

import (
	"context"
	"errors"
	"testing"
)

func TestFixtureSolverSolvesFeasibleDeterministicFixture(t *testing.T) {
	variables := []LPVariable{
		{ItemID: "beans", MaxQuantity: 3, CaloriesPerUnit: 200, ProteinPerUnit: 20, CarbsPerUnit: 30, FatPerUnit: 5},
		{ItemID: "tofu", MaxQuantity: 3, CaloriesPerUnit: 120, ProteinPerUnit: 15, CarbsPerUnit: 5, FatPerUnit: 10},
	}
	request := DietOptimizationRequest{
		OriginalMeals:    []MealInput{{ID: "original", Quantity: 1}},
		TargetMacros:     MacroTarget{Protein: 35, Carbs: 35, Fat: 15},
		TolerancePercent: 1,
	}
	constraints, err := BuildConstraints(request, variables)
	if err != nil {
		t.Fatal(err)
	}
	objective, err := BuildObjective(variables)
	if err != nil {
		t.Fatal(err)
	}

	solution, err := FixtureSolver{}.SolveLP(context.Background(), objective, constraints)
	if err != nil {
		t.Fatal(err)
	}
	if solution.Status != "optimal" || solution.Quantities["beans"] != 1 || solution.Quantities["tofu"] != 1 {
		t.Fatalf("unexpected solution: %#v", solution)
	}
	if solution.ObjectiveValue != 320 {
		t.Fatalf("expected 320 objective value, got %f", solution.ObjectiveValue)
	}
}

func TestFixtureSolverReturnsInfeasibleStatus(t *testing.T) {
	objective, err := BuildObjective([]LPVariable{{ItemID: "tofu", CaloriesPerUnit: 100}})
	if err != nil {
		t.Fatal(err)
	}
	constraints := []LPConstraint{{
		Name:       "impossible",
		LowerBound: 10,
		UpperBound: 10,
		Coefficients: map[string]float64{
			"tofu": 1,
		},
	}}

	_, err = FixtureSolver{MaxUnboundedQuantity: 3}.SolveLP(context.Background(), objective, constraints)
	if !errors.Is(err, ErrSolverInfeasible) {
		t.Fatalf("expected infeasible error, got %v", err)
	}
}

func TestFixtureSolverHonorsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	objective, err := BuildObjective([]LPVariable{{ItemID: "tofu", CaloriesPerUnit: 100}})
	if err != nil {
		t.Fatal(err)
	}

	_, err = FixtureSolver{}.SolveLP(ctx, objective, []LPConstraint{})
	if !errors.Is(err, ErrSolverCancelled) {
		t.Fatalf("expected cancellation error, got %v", err)
	}
}
