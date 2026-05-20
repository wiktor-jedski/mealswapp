package optimization

import (
	"errors"
	"math"
	"testing"
)

func TestBuildConstraintsCreatesMacroToleranceBands(t *testing.T) {
	constraints, err := BuildConstraints(validRequest(), fixtureVariables())
	if err != nil {
		t.Fatal(err)
	}

	protein := findConstraint(t, constraints, "macro:protein")
	if !almostEqual(protein.LowerBound, 90) || !almostEqual(protein.UpperBound, 110) {
		t.Fatalf("unexpected protein bounds: %#v", protein)
	}
	if protein.Coefficients["tofu"] != 20 || protein.Coefficients["lentils"] != 12 {
		t.Fatalf("unexpected protein coefficients: %#v", protein.Coefficients)
	}

	carbs := findConstraint(t, constraints, "macro:carbs")
	if !almostEqual(carbs.LowerBound, 135) || !almostEqual(carbs.UpperBound, 165) {
		t.Fatalf("unexpected carbs bounds: %#v", carbs)
	}
	fat := findConstraint(t, constraints, "macro:fat")
	if !almostEqual(fat.LowerBound, 54) || !almostEqual(fat.UpperBound, 66) {
		t.Fatalf("unexpected fat bounds: %#v", fat)
	}
}

func TestBuildConstraintsAddsExcludedItemAndQuantityBounds(t *testing.T) {
	request := validRequest()
	request.ExcludedIDs = []string{"lentils"}
	constraints, err := BuildConstraints(request, fixtureVariables())
	if err != nil {
		t.Fatal(err)
	}

	tofu := findConstraint(t, constraints, "quantity:tofu")
	if tofu.LowerBound != 0 || tofu.UpperBound != 4 {
		t.Fatalf("unexpected tofu quantity bounds: %#v", tofu)
	}
	lentils := findConstraint(t, constraints, "quantity:lentils")
	if lentils.LowerBound != 0 || lentils.UpperBound != 0 {
		t.Fatalf("expected excluded lentils upper bound 0, got %#v", lentils)
	}
}

func TestBuildConstraintsDetectsInfeasibleInputs(t *testing.T) {
	if _, err := BuildConstraints(validRequest(), nil); !errors.Is(err, ErrInfeasibleConstraints) {
		t.Fatalf("expected no-variable infeasible error, got %v", err)
	}

	request := validRequest()
	request.ExcludedIDs = []string{"tofu"}
	variables := []LPVariable{{ItemID: "tofu", MinQuantity: 1, MaxQuantity: 4}}
	if _, err := BuildConstraints(request, variables); !errors.Is(err, ErrInfeasibleConstraints) {
		t.Fatalf("expected excluded required variable infeasible error, got %v", err)
	}
}

func TestBuildConstraintsAllowsUnboundedQuantityWhenNoMaxProvided(t *testing.T) {
	constraints, err := BuildConstraints(validRequest(), []LPVariable{{ItemID: "oats", ProteinPerUnit: 5, CarbsPerUnit: 20, FatPerUnit: 3}})
	if err != nil {
		t.Fatal(err)
	}
	quantity := findConstraint(t, constraints, "quantity:oats")
	if !math.IsInf(quantity.UpperBound, 1) {
		t.Fatalf("expected unbounded quantity upper bound, got %#v", quantity)
	}
}

func findConstraint(t *testing.T, constraints []LPConstraint, name string) LPConstraint {
	t.Helper()
	for _, constraint := range constraints {
		if constraint.Name == name {
			return constraint
		}
	}
	t.Fatalf("missing constraint %q in %#v", name, constraints)
	return LPConstraint{}
}

func fixtureVariables() []LPVariable {
	return []LPVariable{
		{ItemID: "tofu", MaxQuantity: 4, ProteinPerUnit: 20, CarbsPerUnit: 3, FatPerUnit: 10},
		{ItemID: "lentils", MaxQuantity: 3, ProteinPerUnit: 12, CarbsPerUnit: 30, FatPerUnit: 1},
	}
}

func almostEqual(left float64, right float64) bool {
	return math.Abs(left-right) < 0.000001
}
