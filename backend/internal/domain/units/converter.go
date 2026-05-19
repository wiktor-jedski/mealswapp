package units

import (
	"errors"
	"fmt"
	"math"
)

type Unit string

const (
	Gram       Unit = "gram"
	Milliliter Unit = "milliliter"
	Piece      Unit = "piece"
	Serving    Unit = "serving"
	Ounce      Unit = "ounce"
	FluidOunce Unit = "fluid_ounce"
)

type FoodBasis struct {
	ServingSize            float64
	AverageUnitWeightGrams float64
}

var (
	ErrInvalidQuantity       = errors.New("quantity must be greater than zero")
	ErrUnsupportedConversion = errors.New("unsupported unit conversion")
	ErrMissingServingSize    = errors.New("serving size must be greater than zero")
	ErrMissingUnitWeight     = errors.New("average unit weight grams must be greater than zero")
)

func Convert(value float64, from Unit, to Unit, basis FoodBasis) (float64, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) || value <= 0 {
		return 0, ErrInvalidQuantity
	}

	if from == to {
		return Round(value), nil
	}

	base, err := toMetricBase(value, from, basis)
	if err != nil {
		return 0, err
	}

	converted, err := fromMetricBase(base, to, basis)
	if err != nil {
		return 0, err
	}

	return Round(converted), nil
}

func ToStorageBasis(value float64, unit Unit, basis FoodBasis) (float64, error) {
	switch unit {
	case Gram, Milliliter:
		return Convert(value, unit, unit, basis)
	case Piece:
		return Convert(value, Piece, Gram, basis)
	case Serving:
		if basis.ServingSize <= 0 {
			return 0, ErrMissingServingSize
		}
		return Round(value * basis.ServingSize), nil
	default:
		return 0, fmt.Errorf("%w: %s", ErrUnsupportedConversion, unit)
	}
}

func Round(value float64) float64 {
	return math.Round(value*1000) / 1000
}

func toMetricBase(value float64, from Unit, basis FoodBasis) (float64, error) {
	switch from {
	case Gram, Milliliter:
		return value, nil
	case Ounce:
		return value * 28.349523125, nil
	case FluidOunce:
		return value * 29.5735295625, nil
	case Piece:
		if basis.AverageUnitWeightGrams <= 0 {
			return 0, ErrMissingUnitWeight
		}
		return value * basis.AverageUnitWeightGrams, nil
	case Serving:
		if basis.ServingSize <= 0 {
			return 0, ErrMissingServingSize
		}
		return value * basis.ServingSize, nil
	default:
		return 0, fmt.Errorf("%w: %s", ErrUnsupportedConversion, from)
	}
}

func fromMetricBase(value float64, to Unit, basis FoodBasis) (float64, error) {
	switch to {
	case Gram, Milliliter:
		return value, nil
	case Ounce:
		return value / 28.349523125, nil
	case FluidOunce:
		return value / 29.5735295625, nil
	case Piece:
		if basis.AverageUnitWeightGrams <= 0 {
			return 0, ErrMissingUnitWeight
		}
		return value / basis.AverageUnitWeightGrams, nil
	case Serving:
		if basis.ServingSize <= 0 {
			return 0, ErrMissingServingSize
		}
		return value / basis.ServingSize, nil
	default:
		return 0, fmt.Errorf("%w: %s", ErrUnsupportedConversion, to)
	}
}
