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
	GetByIDs(ctx context.Context, ids []uuid.UUID) ([]models.Tag, error)
	GetByType(ctx context.Context, tagType models.TagType) ([]models.Tag, error)
	GetCategoryTags(ctx context.Context) ([]models.Tag, error)
	GetFunctionalityTags(ctx context.Context) ([]models.Tag, error)
	Create(ctx context.Context, tag *models.Tag) error
}
