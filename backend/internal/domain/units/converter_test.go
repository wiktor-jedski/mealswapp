package units

import (
	"errors"
	"testing"
)

func TestConvertSupportedMetricAndImperialUnits(t *testing.T) {
	cases := []struct {
		name string
		from Unit
		to   Unit
		in   float64
		want float64
	}{
		{name: "grams to ounces", from: Gram, to: Ounce, in: 100, want: 3.527},
		{name: "ounces to grams", from: Ounce, to: Gram, in: 1, want: 28.35},
		{name: "milliliters to fluid ounces", from: Milliliter, to: FluidOunce, in: 100, want: 3.381},
		{name: "fluid ounces to milliliters", from: FluidOunce, to: Milliliter, in: 1, want: 29.574},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Convert(tc.in, tc.from, tc.to, FoodBasis{})
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("expected %.3f, got %.3f", tc.want, got)
			}
		})
	}
}

func TestConvertPiecesAndServings(t *testing.T) {
	basis := FoodBasis{ServingSize: 125, AverageUnitWeightGrams: 50}

	got, err := Convert(2, Piece, Gram, basis)
	if err != nil {
		t.Fatal(err)
	}
	if got != 100 {
		t.Fatalf("expected 100 grams, got %.3f", got)
	}

	got, err = Convert(250, Gram, Serving, basis)
	if err != nil {
		t.Fatal(err)
	}
	if got != 2 {
		t.Fatalf("expected 2 servings, got %.3f", got)
	}
}

func TestToStorageBasisForRecipeQuantities(t *testing.T) {
	got, err := ToStorageBasis(2, Serving, FoodBasis{ServingSize: 125})
	if err != nil {
		t.Fatal(err)
	}

	if got != 250 {
		t.Fatalf("expected 250 storage units, got %.3f", got)
	}
}

func TestUnsupportedConversionFailures(t *testing.T) {
	if _, err := Convert(1, "cup", Gram, FoodBasis{}); !errors.Is(err, ErrUnsupportedConversion) {
		t.Fatalf("expected unsupported conversion error, got %v", err)
	}

	if _, err := Convert(1, Piece, Gram, FoodBasis{}); !errors.Is(err, ErrMissingUnitWeight) {
		t.Fatalf("expected missing unit weight error, got %v", err)
	}

	if _, err := Convert(0, Gram, Ounce, FoodBasis{}); !errors.Is(err, ErrInvalidQuantity) {
		t.Fatalf("expected invalid quantity error, got %v", err)
	}
}
