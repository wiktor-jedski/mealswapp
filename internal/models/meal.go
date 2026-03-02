// Phase: phase-01 | Task: 19 | Architecture: ARCH-005 | Design: MealEntity

package models

import (
	"time"

	"github.com/google/uuid"
)

type MealType string

const (
	MealTypeSingle MealType = "single"
	MealTypeRecipe MealType = "recipe"
)

type UnitSystem string

const (
	UnitSystemMetric   UnitSystem = "metric"
	UnitSystemImperial UnitSystem = "imperial"
)

type UnitConversionFactors struct {
	WeightGramsToOunces   float64 `json:"weight_grams_to_ounces"`
	VolumeMlToFluidOunces float64 `json:"volume_ml_to_fluid_ounces"`
	GramsToPounds         float64 `json:"grams_to_pounds"`
	MLToCups              float64 `json:"ml_to_cups"`
}

var DefaultUnitConversionFactors = UnitConversionFactors{
	WeightGramsToOunces:   0.035274,
	VolumeMlToFluidOunces: 0.033814,
	GramsToPounds:         0.00220462,
	MLToCups:              0.00422675,
}

type RecipeIngredient struct {
	FoodItemID    uuid.UUID     `json:"food_item_id"`
	FoodItemName  string        `json:"food_item_name"`
	Quantity      float64       `json:"quantity"`
	Unit          string        `json:"unit"`
	Macros        Macros        `json:"macros"`
	Micros        Micros        `json:"micros"`
	PhysicalState PhysicalState `json:"physical_state"`
}

type RecipeComposition struct {
	Ingredients []RecipeIngredient `json:"ingredients"`
	TotalMacros Macros             `json:"total_macros"`
	TotalMicros Micros             `json:"total_micros"`
	TotalWeight float64            `json:"total_weight"`
	Servings    int                `json:"servings"`
	PrepTime    int                `json:"prep_time"`
}

type MealTag struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	TagType     TagType   `json:"tag_type"`
	ColorHex    string    `json:"color_hex,omitempty"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Meal struct {
	ID                uuid.UUID          `json:"id"`
	Name              string             `json:"name"`
	Type              MealType           `json:"type"`
	PhysicalState     PhysicalState      `json:"physical_state"`
	PrepTime          int                `json:"prep_time"`
	AverageUnitWeight float64            `json:"average_unit_weight"`
	Macros            Macros             `json:"macros"`
	Micros            Micros             `json:"micros"`
	CategoryTags      []MealTag          `json:"category_tags"`
	FunctionalityTags []MealTag          `json:"functionality_tags"`
	RecipeComposition *RecipeComposition `json:"recipe_composition,omitempty"`
	ImageURL          *string            `json:"image_url,omitempty"`
	CreatedAt         time.Time          `json:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at"`
}

type CreateMealInput struct {
	Name                string                  `json:"name" validate:"required,min=1,max=255"`
	Type                MealType                `json:"type" validate:"required,oneof=single recipe"`
	PhysicalState       PhysicalState           `json:"physical_state" validate:"required,oneof=solid liquid"`
	PrepTime            int                     `json:"prep_time" validate:"required,min=0"`
	AverageUnitWeight   float64                 `json:"average_unit_weight" validate:"required,min=0"`
	Macros              Macros                  `json:"macros" validate:"required"`
	Micros              Micros                  `json:"micros"`
	CategoryTagIDs      []uuid.UUID             `json:"category_tag_ids"`
	FunctionalityTagIDs []uuid.UUID             `json:"functionality_tag_ids"`
	Ingredients         []RecipeIngredientInput `json:"ingredients,omitempty"`
	ImageURL            *string                 `json:"image_url,omitempty"`
}

type RecipeIngredientInput struct {
	FoodItemID uuid.UUID `json:"food_item_id" validate:"required"`
	Quantity   float64   `json:"quantity" validate:"required,min=0"`
	Unit       string    `json:"unit" validate:"required"`
}

type UpdateMealInput struct {
	Name                *string                 `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	PhysicalState       *PhysicalState          `json:"physical_state,omitempty" validate:"omitempty,oneof=solid liquid"`
	PrepTime            *int                    `json:"prep_time,omitempty" validate:"omitempty,min=0"`
	AverageUnitWeight   *float64                `json:"average_unit_weight,omitempty" validate:"omitempty,min=0"`
	Macros              *Macros                 `json:"macros,omitempty"`
	Micros              *Micros                 `json:"micros,omitempty"`
	CategoryTagIDs      []uuid.UUID             `json:"category_tag_ids,omitempty"`
	FunctionalityTagIDs []uuid.UUID             `json:"functionality_tag_ids,omitempty"`
	Ingredients         []RecipeIngredientInput `json:"ingredients,omitempty"`
	ImageURL            *string                 `json:"image_url,omitempty"`
}

type MealQueryOptions struct {
	IDs                 []uuid.UUID    `json:"ids,omitempty"`
	Name                *string        `json:"name,omitempty"`
	Type                *MealType      `json:"type,omitempty"`
	PhysicalState       *PhysicalState `json:"physical_state,omitempty"`
	CategoryTagIDs      []uuid.UUID    `json:"category_tag_ids,omitempty"`
	FunctionalityTagIDs []uuid.UUID    `json:"functionality_tag_ids,omitempty"`
	MinProtein          *float64       `json:"min_protein,omitempty"`
	MaxProtein          *float64       `json:"max_protein,omitempty"`
	MinCarbs            *float64       `json:"min_carbs,omitempty"`
	MaxCarbs            *float64       `json:"max_carbs,omitempty"`
	MinFat              *float64       `json:"min_fat,omitempty"`
	MaxFat              *float64       `json:"max_fat,omitempty"`
	Page                int            `json:"page"`
	PageSize            int            `json:"page_size"`
	SortBy              string         `json:"sort_by"`
	SortOrder           string         `json:"sort_order"`
}

type MealResponse struct {
	ID                uuid.UUID          `json:"id"`
	Name              string             `json:"name"`
	Type              MealType           `json:"type"`
	PhysicalState     PhysicalState      `json:"physical_state"`
	PrepTime          int                `json:"prep_time"`
	AverageUnitWeight float64            `json:"average_unit_weight"`
	Macros            Macros             `json:"macros"`
	Micros            Micros             `json:"micros"`
	CategoryTags      []MealTag          `json:"category_tags"`
	FunctionalityTags []MealTag          `json:"functionality_tags"`
	RecipeComposition *RecipeComposition `json:"recipe_composition,omitempty"`
	ImageURL          *string            `json:"image_url,omitempty"`
	CreatedAt         time.Time          `json:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at"`
}

type ConvertedMeal struct {
	*Meal
	ConvertedMacros Macros     `json:"converted_macros"`
	UnitPreference  UnitSystem `json:"unit_preference"`
	DisplayWeight   float64    `json:"display_weight"`
}

type ScaledMeal struct {
	*ConvertedMeal
	OriginalQuantity float64 `json:"original_quantity"`
	ScaledQuantity   float64 `json:"scaled_quantity"`
	ScaledMacros     Macros  `json:"scaled_macros"`
}

type MealListResult struct {
	Meals   []Meal `json:"meals"`
	Total   int    `json:"total"`
	Limit   int    `json:"limit"`
	Offset  int    `json:"offset"`
	HasMore bool   `json:"has_more"`
}

type MealValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

const (
	ErrCodeMealNotFound      = "MEAL_NOT_FOUND"
	ErrCodeMealAlreadyExists = "MEAL_ALREADY_EXISTS"
	ErrCodeMealInvalidInput  = "MEAL_INVALID_INPUT"
	ErrCodeMealInUse         = "MEAL_IN_USE"
	ErrCodeMealDatabaseError = "MEAL_DATABASE_ERROR"
	ErrCodeMealValidation    = "MEAL_VALIDATION_ERROR"
	ErrCodeMealInvalidType   = "MEAL_INVALID_TYPE"
	ErrCodeMealMissingRecipe = "MEAL_MISSING_RECIPE"
)
