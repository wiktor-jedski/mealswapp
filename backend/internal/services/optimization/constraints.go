package optimization

import (
	"errors"
	"math"
	"slices"
)

var ErrInfeasibleConstraints = errors.New("optimization constraints are infeasible")

type LPVariable struct {
	ItemID           string
	Quantity         float64
	MinQuantity      float64
	MaxQuantity      float64
	CaloriesPerUnit  float64
	ProteinPerUnit   float64
	CarbsPerUnit     float64
	FatPerUnit       float64
	DiversityPenalty float64
}

type LPConstraint struct {
	Name         string             `json:"name"`
	LowerBound   float64            `json:"lowerBound"`
	UpperBound   float64            `json:"upperBound"`
	Coefficients map[string]float64 `json:"coefficients"`
}

func BuildConstraints(request DietOptimizationRequest, variables []LPVariable) ([]LPConstraint, error) {
	if len(variables) == 0 {
		return nil, ErrInfeasibleConstraints
	}
	tolerance := request.TolerancePercent / 100
	if tolerance <= 0 {
		return nil, ErrInfeasibleConstraints
	}

	constraints := []LPConstraint{
		macroConstraint("protein", request.TargetMacros.Protein, tolerance, variables, func(variable LPVariable) float64 { return variable.ProteinPerUnit }),
		macroConstraint("carbs", request.TargetMacros.Carbs, tolerance, variables, func(variable LPVariable) float64 { return variable.CarbsPerUnit }),
		macroConstraint("fat", request.TargetMacros.Fat, tolerance, variables, func(variable LPVariable) float64 { return variable.FatPerUnit }),
	}
	for _, variable := range variables {
		minQuantity := variable.MinQuantity
		maxQuantity := variable.MaxQuantity
		if maxQuantity <= 0 {
			maxQuantity = math.Inf(1)
		}
		if slices.Contains(request.ExcludedIDs, variable.ItemID) {
			maxQuantity = 0
		}
		if minQuantity > maxQuantity {
			return nil, ErrInfeasibleConstraints
		}
		constraints = append(constraints, LPConstraint{
			Name:       "quantity:" + variable.ItemID,
			LowerBound: minQuantity,
			UpperBound: maxQuantity,
			Coefficients: map[string]float64{
				variable.ItemID: 1,
			},
		})
	}
	return constraints, nil
}

func macroConstraint(name string, target float64, tolerance float64, variables []LPVariable, coefficient func(LPVariable) float64) LPConstraint {
	coefficients := make(map[string]float64, len(variables))
	for _, variable := range variables {
		coefficients[variable.ItemID] = coefficient(variable)
	}
	return LPConstraint{
		Name:         "macro:" + name,
		LowerBound:   target * (1 - tolerance),
		UpperBound:   target * (1 + tolerance),
		Coefficients: coefficients,
	}
}
