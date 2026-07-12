package optimization

import (
	"sort"

	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// ObjectiveFunction is the primary LP objective. Its coefficients express
// total calories plus soft diversity penalties per repository base unit, and
// the fixed direction is minimization as required by SW-REQ-022.
// Implements DESIGN-004 ObjectiveFunction.
type ObjectiveFunction struct {
	Coefficients       map[string]float64
	DiversityPenalties map[string]float64
	VariableIDs        []string
}

// BuildObjective creates a deterministic calorie-minimization objective from
// server-derived LP variable coefficients plus any soft diversity penalties.
// A zero calorie value is treated as a missing coefficient because a valid
// objective variable must have a positive calorie basis.
// Implements DESIGN-004 ObjectiveFunction.
func BuildObjective(variables []LPVariable) (ObjectiveFunction, error) {
	if len(variables) == 0 {
		return ObjectiveFunction{}, objectiveValidationError("at least one LP variable is required")
	}

	coefficients := make(map[string]float64, len(variables))
	penalties := make(map[string]float64, len(variables))
	for _, variable := range variables {
		if variable.ItemID == "" {
			return ObjectiveFunction{}, objectiveValidationError("LP variable item ID is required")
		}
		if _, exists := coefficients[variable.ItemID]; exists {
			return ObjectiveFunction{}, objectiveValidationError("duplicate LP variable item ID: " + variable.ItemID)
		}
		if variable.CaloriesPerUnit == 0 {
			return ObjectiveFunction{}, objectiveValidationError("calorie coefficient is missing for " + variable.ItemID)
		}
		if !finite(variable.CaloriesPerUnit) {
			return ObjectiveFunction{}, objectiveValidationError("calorie coefficients must be finite")
		}
		if variable.CaloriesPerUnit < 0 {
			return ObjectiveFunction{}, objectiveValidationError("calorie coefficients cannot be negative")
		}
		if !finite(variable.DiversityPenalty) || variable.DiversityPenalty < 0 {
			return ObjectiveFunction{}, objectiveValidationError("diversity penalties must be finite and non-negative")
		}
		coefficient := variable.CaloriesPerUnit + variable.DiversityPenalty
		if !finite(coefficient) {
			return ObjectiveFunction{}, objectiveValidationError("objective coefficients must be finite")
		}
		coefficients[variable.ItemID] = coefficient
		penalties[variable.ItemID] = variable.DiversityPenalty
	}

	variableIDs := make([]string, 0, len(coefficients))
	for itemID := range coefficients {
		variableIDs = append(variableIDs, itemID)
	}
	sort.Strings(variableIDs)
	return ObjectiveFunction{Coefficients: coefficients, DiversityPenalties: penalties, VariableIDs: variableIDs}, nil
}

// objectiveValidationError returns a typed validation failure for objective inputs.
// Implements DESIGN-004 ObjectiveFunction.
func objectiveValidationError(message string) error {
	return repository.NewError(repository.ErrorKindValidation, message, nil)
}
