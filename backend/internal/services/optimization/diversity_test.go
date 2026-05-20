package optimization

import "testing"

func TestApplyDiversityPenaltyMarksOriginalDietItems(t *testing.T) {
	request := validRequest()
	request.OriginalMeals = []MealInput{{ID: "tofu", Quantity: 1}, {ID: "rice", Quantity: 1}}
	variables := []LPVariable{
		{ItemID: "tofu", CaloriesPerUnit: 100},
		{ItemID: "lentils", CaloriesPerUnit: 120},
		{ItemID: "rice", CaloriesPerUnit: 80, DiversityPenalty: 5},
	}

	penalized := ApplyDiversityPenalty(request, variables, DiversityConfig{PenaltyPerOverlap: 250})

	if penalized[0].DiversityPenalty != 250 {
		t.Fatalf("expected tofu penalty, got %#v", penalized[0])
	}
	if penalized[1].DiversityPenalty != 0 {
		t.Fatalf("expected lentils unchanged, got %#v", penalized[1])
	}
	if penalized[2].DiversityPenalty != 255 {
		t.Fatalf("expected rice existing penalty incremented, got %#v", penalized[2])
	}
	if variables[0].DiversityPenalty != 0 {
		t.Fatalf("expected input variables to remain immutable, got %#v", variables[0])
	}
}

func TestDiversityPenaltyAffectsObjectiveOrdering(t *testing.T) {
	request := validRequest()
	request.OriginalMeals = []MealInput{{ID: "same-meal", Quantity: 1}}
	variables := ApplyDiversityPenalty(request, []LPVariable{
		{ItemID: "same-meal", CaloriesPerUnit: 100},
		{ItemID: "alternative", CaloriesPerUnit: 120},
	}, DiversityConfig{PenaltyPerOverlap: 50})
	objective, err := BuildObjective(variables)
	if err != nil {
		t.Fatal(err)
	}

	same := CandidateSolution{Quantities: map[string]float64{"same-meal": 1}}
	alternative := CandidateSolution{Quantities: map[string]float64{"alternative": 1}}

	if ObjectiveValue(objective, same) != 150 || ObjectiveValue(objective, alternative) != 120 {
		t.Fatalf("unexpected objective values: same=%f alternative=%f", ObjectiveValue(objective, same), ObjectiveValue(objective, alternative))
	}
	if preferred := PreferLowerObjective(objective, same, alternative); preferred.Quantities["alternative"] != 1 {
		t.Fatalf("expected diverse alternative preferred, got %#v", preferred)
	}
}

func TestCountOriginalOverlapIgnoresZeroQuantity(t *testing.T) {
	request := validRequest()
	request.OriginalMeals = []MealInput{{ID: "tofu", Quantity: 1}, {ID: "rice", Quantity: 1}}
	solution := CandidateSolution{Quantities: map[string]float64{"tofu": 0, "rice": 1, "lentils": 2}}

	if got := CountOriginalOverlap(request, solution); got != 1 {
		t.Fatalf("expected one overlapping item, got %d", got)
	}
}
