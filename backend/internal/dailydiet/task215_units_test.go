package dailydiet

// Implements DESIGN-005 UnitConverter saved-diet service boundary verification.

import (
	"testing"

	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

func TestTask215SavedDietQuantityUnitBoundaries(t *testing.T) {
	for _, unit := range []string{"g", "oz"} {
		if _, err := quantityInMealBase(1, unit, repository.PhysicalStateSolid); err != nil {
			t.Fatalf("solid quantityInMealBase(%q) error = %v", unit, err)
		}
	}
	for _, unit := range []string{"ml", "fl_oz"} {
		if _, err := quantityInMealBase(1, unit, repository.PhysicalStateLiquid); err != nil {
			t.Fatalf("liquid quantityInMealBase(%q) error = %v", unit, err)
		}
	}
	for _, test := range []struct {
		unit  string
		state repository.PhysicalState
	}{
		{unit: "serving", state: repository.PhysicalStateSolid},
		{unit: "ml", state: repository.PhysicalStateSolid},
		{unit: "g", state: repository.PhysicalStateLiquid},
		{unit: "cup", state: repository.PhysicalStateLiquid},
	} {
		if _, err := quantityInMealBase(1, test.unit, test.state); !repository.IsKind(err, repository.ErrorKindValidation) {
			t.Fatalf("quantityInMealBase(%q, %q) error = %v, want validation", test.unit, test.state, err)
		}
	}
}
