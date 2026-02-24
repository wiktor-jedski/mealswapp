// Phase: phase-01 | Task: 2 | Architecture: ARCH-005 | Design: FoodItemEntity
package models

import (
	"time"

	"github.com/google/uuid"
)

type PhysicalState string

const (
	PhysicalStateSolid  PhysicalState = "solid"
	PhysicalStateLiquid PhysicalState = "liquid"
)

type Macros struct {
	Protein float64 `json:"protein"`
	Carbs   float64 `json:"carbs"`
	Fat     float64 `json:"fat"`
}

type Micros struct {
	Sodium      float64            `json:"sodium"`
	Fiber       float64            `json:"fiber"`
	Sugar       float64            `json:"sugar"`
	Cholesterol float64            `json:"cholesterol"`
	Potassium   float64            `json:"potassium"`
	VitaminA    float64            `json:"vitamin_a"`
	VitaminC    float64            `json:"vitamin_c"`
	Calcium     float64            `json:"calcium"`
	Iron        float64            `json:"iron"`
	Additional  map[string]float64 `json:"additional,omitempty"`
}

type TagType string

const (
	TagTypeCategory      TagType = "category"
	TagTypeFunctionality TagType = "functionality"
)

type Tag struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	TagType     TagType   `json:"tag_type"`
	ColorHex    string    `json:"color_hex,omitempty"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type FoodItem struct {
	ID                uuid.UUID     `json:"id"`
	Name              string        `json:"name"`
	PhysicalState     PhysicalState `json:"physical_state"`
	PrepTime          int           `json:"prep_time"`
	AverageUnitWeight float64       `json:"average_unit_weight"`
	Macros            Macros        `json:"macros"`
	Micros            Micros        `json:"micros"`
	CategoryTags      []Tag         `json:"category_tags"`
	FunctionalityTags []Tag         `json:"functionality_tags"`
	ImageURL          *string       `json:"image_url,omitempty"`
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`
}

type UnitPreference string

const (
	UnitPreferenceMetric   UnitPreference = "metric"
	UnitPreferenceImperial UnitPreference = "imperial"
)

type FoodItemQuery struct {
	IDs                 []uuid.UUID    `query:"ids,omitempty"`
	Name                *string        `query:"name,omitempty"`
	PhysicalState       *PhysicalState `query:"physical_state,omitempty"`
	CategoryTagIDs      []uuid.UUID    `query:"category_tag_ids,omitempty"`
	FunctionalityTagIDs []uuid.UUID    `query:"functionality_tag_ids,omitempty"`
	MinProtein          *float64       `query:"min_protein,omitempty"`
	MaxProtein          *float64       `query:"max_protein,omitempty"`
	MinCarbs            *float64       `query:"min_carbs,omitempty"`
	MaxCarbs            *float64       `query:"max_carbs,omitempty"`
	MinFat              *float64       `query:"min_fat,omitempty"`
	MaxFat              *float64       `query:"max_fat,omitempty"`
	Page                int            `query:"page"`
	PageSize            int            `query:"page_size"`
	SortBy              string         `query:"sort_by"`
	SortOrder           string         `query:"sort_order"`
}

type FoodItemCreate struct {
	Name                string        `json:"name" validate:"required,min=1,max=255"`
	PhysicalState       PhysicalState `json:"physical_state" validate:"required,oneof=solid liquid"`
	PrepTime            int           `json:"prep_time" validate:"required,min=0"`
	AverageUnitWeight   float64       `json:"average_unit_weight" validate:"required,min=0"`
	Macros              Macros        `json:"macros" validate:"required"`
	Micros              Micros        `json:"micros"`
	CategoryTagIDs      []uuid.UUID   `json:"category_tag_ids"`
	FunctionalityTagIDs []uuid.UUID   `json:"functionality_tag_ids"`
	ImageURL            *string       `json:"image_url,omitempty"`
}

type FoodItemUpdate struct {
	Name                *string        `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	PhysicalState       *PhysicalState `json:"physical_state,omitempty" validate:"omitempty,oneof=solid liquid"`
	PrepTime            *int           `json:"prep_time,omitempty" validate:"omitempty,min=0"`
	AverageUnitWeight   *float64       `json:"average_unit_weight,omitempty" validate:"omitempty,min=0"`
	Macros              *Macros        `json:"macros,omitempty"`
	Micros              *Micros        `json:"micros,omitempty"`
	CategoryTagIDs      []uuid.UUID    `json:"category_tag_ids,omitempty"`
	FunctionalityTagIDs []uuid.UUID    `json:"functionality_tag_ids,omitempty"`
	ImageURL            *string        `json:"image_url,omitempty"`
}

type FoodItemResponse struct {
	ID                uuid.UUID     `json:"id"`
	Name              string        `json:"name"`
	PhysicalState     PhysicalState `json:"physical_state"`
	PrepTime          int           `json:"prep_time"`
	AverageUnitWeight float64       `json:"average_unit_weight"`
	Macros            Macros        `json:"macros"`
	Micros            Micros        `json:"micros"`
	CategoryTags      []Tag         `json:"category_tags"`
	FunctionalityTags []Tag         `json:"functionality_tags"`
	ImageURL          *string       `json:"image_url,omitempty"`
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`
}

type ConvertedFoodItem struct {
	*FoodItem
	ConvertedMacros Macros         `json:"converted_macros"`
	UnitPreference  UnitPreference `json:"unit_preference"`
	DisplayWeight   float64        `json:"display_weight"`
}

type ScaledFoodItem struct {
	*ConvertedFoodItem
	OriginalQuantity float64 `json:"original_quantity"`
	ScaledQuantity   float64 `json:"scaled_quantity"`
	ScaledMacros     Macros  `json:"scaled_macros"`
}

type ValidationError struct {
	Field   string      `json:"field"`
	Message string      `json:"message"`
	Value   interface{} `json:"value,omitempty"`
}

type FoodItemState string

const (
	FoodItemStateNew        FoodItemState = "new"
	FoodItemStateActive     FoodItemState = "active"
	FoodItemStateDeprecated FoodItemState = "deprecated"
	FoodItemStateDeleted    FoodItemState = "deleted"
)
