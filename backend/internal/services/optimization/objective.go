package optimization

import (
	"errors"
	"math"
)

var ErrInvalidObjective = errors.New("objective coefficients must be finite and non-negative")

type ObjectiveFunction struct {
	Sense        string             `json:"sense"`
	Coefficients map[string]float64 `json:"coefficients"`
}

type CandidateSolution struct {
	Quantities map[string]float64
}

func BuildObjective(variables []LPVariable) (ObjectiveFunction, error) {
	coefficients := make(map[string]float64, len(variables))
	for _, variable := range variables {
		coefficient := variable.CaloriesPerUnit + variable.DiversityPenalty
		if math.IsNaN(coefficient) || math.IsInf(coefficient, 0) || coefficient < 0 {
			return ObjectiveFunction{}, ErrInvalidObjective
		}
		coefficients[variable.ItemID] = coefficient
	}
	return ObjectiveFunction{Sense: "minimize", Coefficients: coefficients}, nil
}

func ObjectiveValue(objective ObjectiveFunction, solution CandidateSolution) float64 {
	var total float64
	for itemID, quantity := range solution.Quantities {
		total += objective.Coefficients[itemID] * quantity
	}
	return total
}

func PreferLowerObjective(objective ObjectiveFunction, left CandidateSolution, right CandidateSolution) CandidateSolution {
	if ObjectiveValue(objective, left) <= ObjectiveValue(objective, right) {
		return left
	}
	return right
}
