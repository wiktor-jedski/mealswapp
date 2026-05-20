package optimization

const DefaultDiversityPenalty = 1000

type DiversityConfig struct {
	PenaltyPerOverlap float64
}

func ApplyDiversityPenalty(request DietOptimizationRequest, variables []LPVariable, config DiversityConfig) []LPVariable {
	penalty := config.PenaltyPerOverlap
	if penalty <= 0 {
		penalty = DefaultDiversityPenalty
	}

	originalIDs := map[string]bool{}
	for _, meal := range request.OriginalMeals {
		if meal.ID != "" {
			originalIDs[meal.ID] = true
		}
	}

	penalized := make([]LPVariable, len(variables))
	for i, variable := range variables {
		penalized[i] = variable
		if originalIDs[variable.ItemID] {
			penalized[i].DiversityPenalty += penalty
		}
	}
	return penalized
}

func CountOriginalOverlap(request DietOptimizationRequest, solution CandidateSolution) int {
	originalIDs := map[string]bool{}
	for _, meal := range request.OriginalMeals {
		originalIDs[meal.ID] = true
	}

	count := 0
	for itemID, quantity := range solution.Quantities {
		if quantity > 0 && originalIDs[itemID] {
			count++
		}
	}
	return count
}
