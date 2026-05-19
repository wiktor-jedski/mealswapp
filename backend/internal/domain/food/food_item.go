package food

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
)

type PhysicalState string

const (
	PhysicalStateSolid  PhysicalState = "solid"
	PhysicalStateLiquid PhysicalState = "liquid"
)

type ServingUnit string

const (
	ServingUnitGram       ServingUnit = "gram"
	ServingUnitMilliliter ServingUnit = "milliliter"
	ServingUnitPiece      ServingUnit = "piece"
	ServingUnitServing    ServingUnit = "serving"
)

type MacroValues struct {
	ProteinGrams float64
	CarbsGrams   float64
	FatGrams     float64
}

type SourceMetadata struct {
	Provider      string
	ExternalID    string
	ProviderURL   string
	ImportedAt    *time.Time
	CurationState string
}

type FoodItemEntity struct {
	ID                     uuid.UUID
	Name                   string
	PhysicalState          PhysicalState
	ServingUnit            ServingUnit
	ServingSize            float64
	CaloriesPer100         float64
	MacrosPer100           MacroValues
	Micros                 map[string]float64
	Source                 SourceMetadata
	ImageURL               string
	PrepTimeMinutes        int
	AverageUnitWeightGrams float64
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

var (
	ErrMissingName            = errors.New("food item name is required")
	ErrInvalidPhysicalState   = errors.New("physical state must be solid or liquid")
	ErrUnsupportedServingUnit = errors.New("unsupported serving unit")
	ErrInvalidServingSize     = errors.New("serving size must be greater than zero")
	ErrInvalidCalories        = errors.New("calories per 100 must be zero or greater")
	ErrInvalidMacros          = errors.New("macro values must be zero or greater")
	ErrInvalidMicronutrients  = errors.New("micronutrient values must be zero or greater")
	ErrInvalidPrepTime        = errors.New("prep time minutes must be zero or greater")
	ErrInvalidUnitWeight      = errors.New("average unit weight grams must be zero or greater")
)

func (item FoodItemEntity) Validate() error {
	if strings.TrimSpace(item.Name) == "" {
		return ErrMissingName
	}

	if !item.PhysicalState.Valid() {
		return ErrInvalidPhysicalState
	}

	if !item.ServingUnit.Valid() {
		return ErrUnsupportedServingUnit
	}

	if !validNonZeroNumber(item.ServingSize) {
		return ErrInvalidServingSize
	}

	if !validNumber(item.CaloriesPer100) {
		return ErrInvalidCalories
	}

	if err := item.MacrosPer100.Validate(); err != nil {
		return err
	}

	for key, value := range item.Micros {
		if strings.TrimSpace(key) == "" || !validNumber(value) {
			return fmt.Errorf("%w: %s", ErrInvalidMicronutrients, key)
		}
	}

	if item.PrepTimeMinutes < 0 {
		return ErrInvalidPrepTime
	}

	if !validNumber(item.AverageUnitWeightGrams) {
		return ErrInvalidUnitWeight
	}

	return nil
}

func (state PhysicalState) Valid() bool {
	return state == PhysicalStateSolid || state == PhysicalStateLiquid
}

func (unit ServingUnit) Valid() bool {
	switch unit {
	case ServingUnitGram, ServingUnitMilliliter, ServingUnitPiece, ServingUnitServing:
		return true
	default:
		return false
	}
}

func (macros MacroValues) Validate() error {
	if !validNumber(macros.ProteinGrams) || !validNumber(macros.CarbsGrams) || !validNumber(macros.FatGrams) {
		return ErrInvalidMacros
	}

	return nil
}

func validNumber(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0
}

func validNonZeroNumber(value float64) bool {
	return validNumber(value) && value > 0
}
