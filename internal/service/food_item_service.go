// Phase: phase-01 | Task: 7 | Architecture: ARCH-005 | Design: FoodItemEntity

package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"mealswapp/internal/models"
	"mealswapp/internal/repository"
)

var (
	ErrFoodItemNotFound          = fmt.Errorf("food item not found")
	ErrInvalidCategoryTagID      = fmt.Errorf("invalid category tag ID")
	ErrInvalidFunctionalityTagID = fmt.Errorf("invalid functionality tag ID")
	ErrFoodItemInUse             = fmt.Errorf("food item is in use by recipes")
	ErrInvalidUnitPreference     = fmt.Errorf("invalid unit preference")
	ErrInvalidPhysicalState      = fmt.Errorf("invalid physical state")
	ErrNegativeMacroValue        = fmt.Errorf("negative macro value not allowed")
	ErrQuantityOutOfRange        = fmt.Errorf("quantity must be greater than 0")
)

type FoodItemService interface {
	CreateFoodItem(ctx context.Context, input models.FoodItemCreate) (*models.FoodItem, error)
	GetFoodItem(ctx context.Context, id uuid.UUID, unitPref models.UnitPreference) (*models.ConvertedFoodItem, error)
	ListFoodItems(ctx context.Context, query models.FoodItemQuery) ([]models.ConvertedFoodItem, int64, error)
	UpdateFoodItem(ctx context.Context, id uuid.UUID, input models.FoodItemUpdate) (*models.FoodItem, error)
	DeleteFoodItem(ctx context.Context, id uuid.UUID) error
	ScaleFoodItem(ctx context.Context, id uuid.UUID, quantity float64, unitPref models.UnitPreference) (*models.ScaledFoodItem, error)
	ValidateFoodItem(ctx context.Context, input models.FoodItemCreate) []models.ValidationError
}

type foodItemService struct {
	foodItemRepo repository.FoodItemRepository
	tagRepo      repository.TagRepository
}

func NewFoodItemService(foodItemRepo repository.FoodItemRepository, tagRepo repository.TagRepository) FoodItemService {
	return &foodItemService{
		foodItemRepo: foodItemRepo,
		tagRepo:      tagRepo,
	}
}

func (s *foodItemService) CreateFoodItem(ctx context.Context, input models.FoodItemCreate) (*models.FoodItem, error) {
	validationErrors := s.ValidateFoodItem(ctx, input)
	if len(validationErrors) > 0 {
		return nil, fmt.Errorf("validation failed")
	}

	uuid := uuid.New()
	now := time.Now().UTC()

	item := &models.FoodItem{
		ID:                uuid,
		Name:              input.Name,
		PhysicalState:     input.PhysicalState,
		PrepTime:          input.PrepTime,
		AverageUnitWeight: input.AverageUnitWeight,
		Macros:            input.Macros,
		Micros:            input.Micros,
		ImageURL:          input.ImageURL,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if len(input.CategoryTagIDs) > 0 {
		tagIDStrings := make([]string, len(input.CategoryTagIDs))
		for i, id := range input.CategoryTagIDs {
			tagIDStrings[i] = id.String()
		}
		tags, err := s.tagRepo.GetByIDs(ctx, tagIDStrings)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch category tags: %w", err)
		}
		if len(tags) != len(input.CategoryTagIDs) {
			return nil, ErrInvalidCategoryTagID
		}
		for _, tag := range tags {
			tagUUID, err := uuid.Parse(tag.ID)
			if err != nil {
				return nil, ErrInvalidCategoryTagID
			}
			item.CategoryTags = append(item.CategoryTags, models.Tag{
				ID:      tagUUID,
				Name:    tag.Name,
				TagType: tag.TagType,
			})
		}
	}

	if len(input.FunctionalityTagIDs) > 0 {
		tagIDStrings := make([]string, len(input.FunctionalityTagIDs))
		for i, id := range input.FunctionalityTagIDs {
			tagIDStrings[i] = id.String()
		}
		tags, err := s.tagRepo.GetByIDs(ctx, tagIDStrings)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch functionality tags: %w", err)
		}
		if len(tags) != len(input.FunctionalityTagIDs) {
			return nil, ErrInvalidFunctionalityTagID
		}
		for _, tag := range tags {
			tagUUID, err := uuid.Parse(tag.ID)
			if err != nil {
				return nil, ErrInvalidFunctionalityTagID
			}
			item.FunctionalityTags = append(item.FunctionalityTags, models.Tag{
				ID:      tagUUID,
				Name:    tag.Name,
				TagType: tag.TagType,
			})
		}
	}

	if err := s.foodItemRepo.Create(ctx, item); err != nil {
		return nil, fmt.Errorf("failed to create food item: %w", err)
	}

	return s.foodItemRepo.GetByID(ctx, uuid)
}

func (s *foodItemService) GetFoodItem(ctx context.Context, id uuid.UUID, unitPref models.UnitPreference) (*models.ConvertedFoodItem, error) {
	if err := validateUnitPreference(unitPref); err != nil {
		return nil, err
	}

	item, err := s.foodItemRepo.GetByID(ctx, id)
	if err != nil {
		if err.Error() == "food item not found" {
			return nil, ErrFoodItemNotFound
		}
		return nil, fmt.Errorf("failed to get food item: %w", err)
	}

	return s.convertFoodItem(item, unitPref), nil
}

func (s *foodItemService) ListFoodItems(ctx context.Context, query models.FoodItemQuery) ([]models.ConvertedFoodItem, int64, error) {
	items, total, err := s.foodItemRepo.List(ctx, query)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list food items: %w", err)
	}

	convertedItems := make([]models.ConvertedFoodItem, len(items))
	for i, item := range items {
		convertedItems[i] = *s.convertFoodItem(item, models.UnitPreferenceMetric)
	}

	return convertedItems, total, nil
}

func (s *foodItemService) UpdateFoodItem(ctx context.Context, id uuid.UUID, input models.FoodItemUpdate) (*models.FoodItem, error) {
	exists, err := s.foodItemRepo.Exists(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to check existence: %w", err)
	}
	if !exists {
		return nil, ErrFoodItemNotFound
	}

	updates := make(map[string]interface{})

	if input.Name != nil {
		updates["name"] = *input.Name
	}
	if input.PhysicalState != nil {
		updates["physical_state"] = string(*input.PhysicalState)
	}
	if input.PrepTime != nil {
		updates["prep_time"] = *input.PrepTime
	}
	if input.AverageUnitWeight != nil {
		updates["average_unit_weight"] = *input.AverageUnitWeight
	}
	if input.Macros != nil {
		updates["macros"] = *input.Macros
	}
	if input.Micros != nil {
		updates["micros"] = *input.Micros
	}
	if input.ImageURL != nil {
		updates["image_url"] = input.ImageURL
	}
	if len(input.CategoryTagIDs) > 0 {
		updates["category_tag_ids"] = input.CategoryTagIDs
	}
	if len(input.FunctionalityTagIDs) > 0 {
		updates["functionality_tag_ids"] = input.FunctionalityTagIDs
	}

	if len(updates) > 0 {
		if err := s.foodItemRepo.Update(ctx, id, updates); err != nil {
			return nil, fmt.Errorf("failed to update food item: %w", err)
		}
	}

	return s.foodItemRepo.GetByID(ctx, id)
}

func (s *foodItemService) DeleteFoodItem(ctx context.Context, id uuid.UUID) error {
	exists, err := s.foodItemRepo.Exists(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to check existence: %w", err)
	}
	if !exists {
		return ErrFoodItemNotFound
	}

	if err := s.foodItemRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete food item: %w", err)
	}

	return nil
}

func (s *foodItemService) ScaleFoodItem(ctx context.Context, id uuid.UUID, quantity float64, unitPref models.UnitPreference) (*models.ScaledFoodItem, error) {
	if quantity <= 0 {
		return nil, ErrQuantityOutOfRange
	}

	if err := validateUnitPreference(unitPref); err != nil {
		return nil, err
	}

	item, err := s.foodItemRepo.GetByID(ctx, id)
	if err != nil {
		if err.Error() == "food item not found" {
			return nil, ErrFoodItemNotFound
		}
		return nil, fmt.Errorf("failed to get food item: %w", err)
	}

	converted := s.convertFoodItem(item, unitPref)

	scaleFactor := quantity / 100.0
	scaledMacros := models.Macros{
		Protein: converted.ConvertedMacros.Protein * scaleFactor,
		Carbs:   converted.ConvertedMacros.Carbs * scaleFactor,
		Fat:     converted.ConvertedMacros.Fat * scaleFactor,
	}

	scaledQuantity := quantity
	if unitPref == models.UnitPreferenceImperial {
		if item.PhysicalState == models.PhysicalStateSolid {
			scaledQuantity = quantity / 28.3495
		} else {
			scaledQuantity = quantity / 29.5735
		}
	}

	return &models.ScaledFoodItem{
		ConvertedFoodItem: converted,
		OriginalQuantity:  100,
		ScaledQuantity:    scaledQuantity,
		ScaledMacros:      scaledMacros,
	}, nil
}

func (s *foodItemService) ValidateFoodItem(ctx context.Context, input models.FoodItemCreate) []models.ValidationError {
	var errors []models.ValidationError

	if input.Name == "" {
		errors = append(errors, models.ValidationError{Field: "name", Message: "name is required"})
	} else if len(input.Name) > 255 {
		errors = append(errors, models.ValidationError{Field: "name", Message: "name must be at most 255 characters"})
	}

	if input.PhysicalState != models.PhysicalStateSolid && input.PhysicalState != models.PhysicalStateLiquid {
		errors = append(errors, models.ValidationError{Field: "physical_state", Message: "physical_state must be 'solid' or 'liquid'"})
	}

	if input.PrepTime < 0 {
		errors = append(errors, models.ValidationError{Field: "prep_time", Message: "prep_time must be non-negative"})
	}

	if input.AverageUnitWeight < 0 {
		errors = append(errors, models.ValidationError{Field: "average_unit_weight", Message: "average_unit_weight must be non-negative"})
	}

	if input.Macros.Protein < 0 || input.Macros.Carbs < 0 || input.Macros.Fat < 0 {
		errors = append(errors, models.ValidationError{Field: "macros", Message: "macros values must be non-negative"})
	}

	return errors
}

func (s *foodItemService) convertFoodItem(item *models.FoodItem, unitPref models.UnitPreference) *models.ConvertedFoodItem {
	convertedMacros := item.Macros
	displayWeight := item.AverageUnitWeight

	if unitPref == models.UnitPreferenceImperial {
		if item.PhysicalState == models.PhysicalStateSolid {
			displayWeight = item.AverageUnitWeight / 28.3495
			convertedMacros = models.Macros{
				Protein: item.Macros.Protein * 0.0283495,
				Carbs:   item.Macros.Carbs * 0.0283495,
				Fat:     item.Macros.Fat * 0.0283495,
			}
		} else {
			displayWeight = item.AverageUnitWeight / 29.5735
			convertedMacros = models.Macros{
				Protein: item.Macros.Protein * 0.0295735,
				Carbs:   item.Macros.Carbs * 0.0295735,
				Fat:     item.Macros.Fat * 0.0295735,
			}
		}
	}

	return &models.ConvertedFoodItem{
		FoodItem:        item,
		ConvertedMacros: convertedMacros,
		UnitPreference:  unitPref,
		DisplayWeight:   displayWeight,
	}
}

func validateUnitPreference(unitPref models.UnitPreference) error {
	if unitPref != models.UnitPreferenceMetric && unitPref != models.UnitPreferenceImperial {
		return ErrInvalidUnitPreference
	}
	return nil
}
