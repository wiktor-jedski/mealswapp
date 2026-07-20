package optimization

// ObjectiveFunction is one serialized minimization objective.
// Implements DESIGN-004 ObjectiveFunction.
type ObjectiveFunction struct {
	Coefficients map[string]float64
}

// ObjectivePolicy defines lexicographic solving: Primary is minimized first,
// then Secondary is minimized while the primary optimum is fixed exactly.
// Implements DESIGN-004 ObjectiveFunction and DiversityPenalizer.
type ObjectivePolicy struct {
	Primary   ObjectiveFunction
	Secondary ObjectiveFunction
}

// BuildObjective creates deterministic primary calorie and secondary diversity
// objectives. Every variable must have a positive server-derived calorie basis;
// the builder filters unrelated zero-information candidates before this point.
// Implements DESIGN-004 ObjectiveFunction.
func BuildObjective(variables []LPVariable) (ObjectivePolicy, error) {
	if len(variables) == 0 {
		return ObjectivePolicy{}, validationError("at least one LP variable is required")
	}

	primary := make(map[string]float64, len(variables))
	secondary := make(map[string]float64, len(variables))
	for _, variable := range variables {
		if variable.ItemID == "" {
			return ObjectivePolicy{}, validationError("LP variable item ID is required")
		}
		if _, exists := primary[variable.ItemID]; exists {
			return ObjectivePolicy{}, validationError("duplicate LP variable item ID: " + variable.ItemID)
		}
		if !finite(variable.CaloriesPerUnit) || variable.CaloriesPerUnit <= 0 {
			return ObjectivePolicy{}, validationError("calorie coefficients must be finite and positive")
		}
		if !finite(variable.DiversityPenalty) || variable.DiversityPenalty < 0 {
			return ObjectivePolicy{}, validationError("diversity coefficients must be finite and non-negative")
		}
		primary[variable.ItemID] = variable.CaloriesPerUnit
		secondary[variable.ItemID] = variable.DiversityPenalty
	}

	return ObjectivePolicy{
		Primary:   ObjectiveFunction{Coefficients: primary},
		Secondary: ObjectiveFunction{Coefficients: secondary},
	}, nil
}
