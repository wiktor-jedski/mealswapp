package optimization

import (
	"errors"
	"math"
	"testing"
)

func TestBuildObjectiveUsesCaloriesPlusDiversityPenalty(t *testing.T) {
	objective, err := BuildObjective([]LPVariable{
		{ItemID: "tofu", CaloriesPerUnit: 120, DiversityPenalty: 10},
		{ItemID: "lentils", CaloriesPerUnit: 180},
	})
	if err != nil {
		t.Fatal(err)
	}
	if objective.Sense != "minimize" {
		t.Fatalf("expected minimize objective, got %#v", objective)
	}
	if objective.Coefficients["tofu"] != 130 || objective.Coefficients["lentils"] != 180 {
		t.Fatalf("unexpected coefficients: %#v", objective.Coefficients)
	}
}

func TestObjectiveValueAndPreferenceSelectLowerCalories(t *testing.T) {
	objective, err := BuildObjective([]LPVariable{
		{ItemID: "low-calorie", CaloriesPerUnit: 100},
		{ItemID: "high-calorie", CaloriesPerUnit: 250},
	})
	if err != nil {
		t.Fatal(err)
	}
	low := CandidateSolution{Quantities: map[string]float64{"low-calorie": 2}}
	high := CandidateSolution{Quantities: map[string]float64{"high-calorie": 1}}

	if ObjectiveValue(objective, low) != 200 || ObjectiveValue(objective, high) != 250 {
		t.Fatalf("unexpected objective values: low=%f high=%f", ObjectiveValue(objective, low), ObjectiveValue(objective, high))
	}
	if preferred := PreferLowerObjective(objective, low, high); preferred.Quantities["low-calorie"] != 2 {
		t.Fatalf("expected lower-calorie solution preferred, got %#v", preferred)
	}
}

func TestBuildObjectiveRejectsInvalidCoefficients(t *testing.T) {
	cases := []LPVariable{
		{ItemID: "nan", CaloriesPerUnit: math.NaN()},
		{ItemID: "inf", CaloriesPerUnit: math.Inf(1)},
		{ItemID: "negative", CaloriesPerUnit: -1},
	}
	for _, variable := range cases {
		if _, err := BuildObjective([]LPVariable{variable}); !errors.Is(err, ErrInvalidObjective) {
			t.Fatalf("expected invalid objective for %#v, got %v", variable, err)
		}
	}
}
