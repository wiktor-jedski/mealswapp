package recipe

import (
	"errors"
	"math"
	"strings"
	"time"

	"mealswapp/backend/internal/domain/food"
	"mealswapp/backend/internal/domain/meal"

	"github.com/google/uuid"
)

type RecipeEntity struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	Name           string
	Ingredients    []RecipeIngredientEntity
	CaloriesTotal  float64
	MacrosTotal    food.MacroValues
	SourceProvider string
	SourceID       string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type RecipeIngredientEntity struct {
	ID         uuid.UUID
	RecipeID   uuid.UUID
	FoodItemID uuid.UUID
	Quantity   float64
	Unit       meal.IngredientUnit
	Position   int
}

var (
	ErrMissingUserID      = errors.New("recipe user id is required")
	ErrMissingName        = errors.New("recipe name is required")
	ErrMissingIngredients = errors.New("recipe must contain at least one ingredient")
	ErrMissingFoodItemID  = errors.New("recipe ingredient food item id is required")
	ErrInvalidQuantity    = errors.New("recipe ingredient quantity must be greater than zero")
	ErrUnsupportedUnit    = errors.New("unsupported recipe ingredient unit")
	ErrInvalidPosition    = errors.New("recipe ingredient position must be zero or greater")
	ErrInvalidAggregate   = errors.New("recipe aggregate values must be zero or greater")
)

func (recipe RecipeEntity) Validate() error {
	if recipe.UserID == uuid.Nil {
		return ErrMissingUserID
	}

	if strings.TrimSpace(recipe.Name) == "" {
		return ErrMissingName
	}

	if len(recipe.Ingredients) == 0 {
		return ErrMissingIngredients
	}

	for _, ingredient := range recipe.Ingredients {
		if err := ingredient.Validate(); err != nil {
			return err
		}
	}

	if !validNumber(recipe.CaloriesTotal) {
		return ErrInvalidAggregate
	}
	if err := recipe.MacrosTotal.Validate(); err != nil {
		return ErrInvalidAggregate
	}

	return nil
}

func (ingredient RecipeIngredientEntity) Validate() error {
	if ingredient.FoodItemID == uuid.Nil {
		return ErrMissingFoodItemID
	}
	if !validNonZeroNumber(ingredient.Quantity) {
		return ErrInvalidQuantity
	}
	if !ingredient.Unit.Valid() {
		return ErrUnsupportedUnit
	}
	if ingredient.Position < 0 {
		return ErrInvalidPosition
	}

	return nil
}

func validNumber(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0
}

func validNonZeroNumber(value float64) bool {
	return validNumber(value) && value > 0
}
