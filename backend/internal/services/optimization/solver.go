package optimization

import (
	"context"
	"errors"
	"math"
)

var (
	ErrSolverInfeasible = errors.New("solver infeasible")
	ErrSolverCancelled  = errors.New("solver cancelled")
)

type Solver interface {
	SolveLP(ctx context.Context, objective ObjectiveFunction, constraints []LPConstraint) (LPSolution, error)
}

type LPSolution struct {
	Status         string             `json:"status"`
	Quantities     map[string]float64 `json:"quantities"`
	ObjectiveValue float64            `json:"objectiveValue"`
}

type FixtureSolver struct {
	MaxUnboundedQuantity int
}

func (solver FixtureSolver) SolveLP(ctx context.Context, objective ObjectiveFunction, constraints []LPConstraint) (LPSolution, error) {
	variableIDs := sortedVariableIDs(objective)
	if len(variableIDs) == 0 {
		return LPSolution{}, ErrSolverInfeasible
	}
	maxUnbounded := solver.MaxUnboundedQuantity
	if maxUnbounded <= 0 {
		maxUnbounded = 10
	}

	domains, err := quantityDomains(variableIDs, constraints, maxUnbounded)
	if err != nil {
		return LPSolution{}, err
	}

	bestValue := math.Inf(1)
	best := map[string]float64{}
	current := map[string]float64{}
	var search func(index int) error
	search = func(index int) error {
		select {
		case <-ctx.Done():
			return ErrSolverCancelled
		default:
		}
		if index == len(variableIDs) {
			if !satisfiesConstraints(current, constraints) {
				return nil
			}
			value := ObjectiveValue(objective, CandidateSolution{Quantities: current})
			if value < bestValue {
				bestValue = value
				best = cloneQuantities(current)
			}
			return nil
		}
		itemID := variableIDs[index]
		for _, quantity := range domains[itemID] {
			current[itemID] = quantity
			if err := search(index + 1); err != nil {
				return err
			}
		}
		return nil
	}
	if err := search(0); err != nil {
		return LPSolution{}, err
	}
	if math.IsInf(bestValue, 1) {
		return LPSolution{}, ErrSolverInfeasible
	}
	return LPSolution{Status: "optimal", Quantities: best, ObjectiveValue: bestValue}, nil
}

func sortedVariableIDs(objective ObjectiveFunction) []string {
	ids := make([]string, 0, len(objective.Coefficients))
	for id := range objective.Coefficients {
		ids = append(ids, id)
	}
	for i := 0; i < len(ids); i++ {
		for j := i + 1; j < len(ids); j++ {
			if ids[j] < ids[i] {
				ids[i], ids[j] = ids[j], ids[i]
			}
		}
	}
	return ids
}

func quantityDomains(variableIDs []string, constraints []LPConstraint, maxUnbounded int) (map[string][]float64, error) {
	domains := make(map[string][]float64, len(variableIDs))
	for _, itemID := range variableIDs {
		lower := 0.0
		upper := float64(maxUnbounded)
		for _, constraint := range constraints {
			if len(constraint.Coefficients) == 1 && constraint.Coefficients[itemID] == 1 {
				lower = math.Max(lower, constraint.LowerBound)
				if !math.IsInf(constraint.UpperBound, 1) {
					upper = math.Min(upper, constraint.UpperBound)
				}
			}
		}
		if lower > upper {
			return nil, ErrSolverInfeasible
		}
		start := int(math.Ceil(lower))
		end := int(math.Floor(upper))
		for quantity := start; quantity <= end; quantity++ {
			domains[itemID] = append(domains[itemID], float64(quantity))
		}
		if len(domains[itemID]) == 0 {
			return nil, ErrSolverInfeasible
		}
	}
	return domains, nil
}

func satisfiesConstraints(quantities map[string]float64, constraints []LPConstraint) bool {
	const epsilon = 0.000001
	for _, constraint := range constraints {
		value := 0.0
		for itemID, coefficient := range constraint.Coefficients {
			value += coefficient * quantities[itemID]
		}
		if value+epsilon < constraint.LowerBound || value-epsilon > constraint.UpperBound {
			return false
		}
	}
	return true
}

func cloneQuantities(input map[string]float64) map[string]float64 {
	output := make(map[string]float64, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}
