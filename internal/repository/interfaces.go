// Phase: phase-01 | Task: 6 | Architecture: ARCH-005 | Design: FoodItemEntity
package repository

import (
	"context"

	"github.com/google/uuid"
	"mealswapp/internal/models"
)

type FoodItemRepository interface {
	Create(ctx context.Context, item *models.FoodItem) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.FoodItem, error)
	List(ctx context.Context, query models.FoodItemQuery) ([]*models.FoodItem, int64, error)
	Update(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error
	Delete(ctx context.Context, id uuid.UUID) error
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
	GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*models.FoodItem, error)
	Count(ctx context.Context, query models.FoodItemQuery) (int64, error)
}

type TagRepository interface {
	Create(ctx context.Context, input models.TagCreateInput) (*models.Tag, error)
	Update(ctx context.Context, id string, input models.TagUpdateInput) (*models.Tag, error)
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*models.Tag, error)
	GetBySlug(ctx context.Context, slug string) (*models.Tag, error)
	GetByIDs(ctx context.Context, ids []string) ([]models.Tag, error)
	GetByType(ctx context.Context, tagType models.TagType) ([]models.Tag, error)
	GetCategoryTags(ctx context.Context) ([]models.Tag, error)
	GetFunctionalityTags(ctx context.Context) ([]models.Tag, error)
	List(ctx context.Context, filter models.TagFilter) (*models.TagListResult, error)
	Exists(ctx context.Context, id string) (bool, error)
	ExistsBySlug(ctx context.Context, slug string, excludeID string) (bool, error)
	CountByType(ctx context.Context, tagType models.TagType) (int, error)
	GetTagsForFoodItem(ctx context.Context, foodItemID string) ([]models.Tag, error)
	GetTagsForMeal(ctx context.Context, mealID string) ([]models.Tag, error)
	AssignTagsToFoodItem(ctx context.Context, foodItemID string, tagIDs []string) error
	RemoveTagsFromFoodItem(ctx context.Context, foodItemID string, tagIDs []string) error
	AssignTagsToMeal(ctx context.Context, mealID string, tagIDs []string) error
	RemoveTagsFromMeal(ctx context.Context, mealID string, tagIDs []string) error
}
