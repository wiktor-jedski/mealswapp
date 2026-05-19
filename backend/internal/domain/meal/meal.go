package meal

import (
	"errors"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
)

type MealType string

const (
	MealTypeSingle MealType = "single"
	MealTypeRecipe MealType = "recipe"
)

type IngredientUnit string

const (
	IngredientUnitGram       IngredientUnit = "gram"
	IngredientUnitMilliliter IngredientUnit = "milliliter"
	IngredientUnitPiece      IngredientUnit = "piece"
	IngredientUnitServing    IngredientUnit = "serving"
)

type MealEntity struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Name      string
	Type      MealType
	Items     []MealItemEntity
	CreatedAt time.Time
	UpdatedAt time.Time
}

type MealItemEntity struct {
	ID         uuid.UUID
	MealID     uuid.UUID
	FoodItemID uuid.UUID
	Quantity   float64
	Unit       IngredientUnit
	Position   int
}

var (
	ErrMissingUserID          = errors.New("meal user id is required")
	ErrMissingName            = errors.New("meal name is required")
	ErrInvalidMealType        = errors.New("meal type must be single or recipe")
	ErrMissingItems           = errors.New("meal must contain at least one item")
	ErrMissingFoodItemID      = errors.New("meal item food item id is required")
	ErrInvalidItemQuantity    = errors.New("meal item quantity must be greater than zero")
	ErrUnsupportedItemUnit    = errors.New("unsupported meal item unit")
	ErrInvalidItemPosition    = errors.New("meal item position must be zero or greater")
	ErrSingleMealItemMismatch = errors.New("single meals must contain exactly one item")
)

func (meal MealEntity) Validate() error {
	if meal.UserID == uuid.Nil {
		return ErrMissingUserID
	}

	if strings.TrimSpace(meal.Name) == "" {
		return ErrMissingName
	}

	if !meal.Type.Valid() {
		return ErrInvalidMealType
	}

	if len(meal.Items) == 0 {
		return ErrMissingItems
	}

	if meal.Type == MealTypeSingle && len(meal.Items) != 1 {
		return ErrSingleMealItemMismatch
	}

	for _, item := range meal.Items {
		if err := item.Validate(); err != nil {
			return err
		}
	}

	return nil
}

func (item MealItemEntity) Validate() error {
	if item.FoodItemID == uuid.Nil {
		return ErrMissingFoodItemID
	}

	if math.IsNaN(item.Quantity) || math.IsInf(item.Quantity, 0) || item.Quantity <= 0 {
		return ErrInvalidItemQuantity
	}

	if !item.Unit.Valid() {
		return ErrUnsupportedItemUnit
	}

	if item.Position < 0 {
		return ErrInvalidItemPosition
	}

	return nil
}

func (mealType MealType) Valid() bool {
	return mealType == MealTypeSingle || mealType == MealTypeRecipe
}

func (unit IngredientUnit) Valid() bool {
	switch unit {
	case IngredientUnitGram, IngredientUnitMilliliter, IngredientUnitPiece, IngredientUnitServing:
		return true
	default:
		return false
	}
}
