package repositories

import (
	"context"
	"time"

	"mealswapp/backend/internal/domain/food"
	"mealswapp/backend/internal/domain/meal"
	"mealswapp/backend/internal/domain/micronutrient"
	"mealswapp/backend/internal/domain/recipe"
	"mealswapp/backend/internal/domain/tag"

	"github.com/google/uuid"
)

type RepositoryContext struct {
	UserID         *uuid.UUID
	UnitSystem     string
	IncludeDeleted bool
}

type FoodItemQuery struct {
	Text          string
	IncludeTagIDs []uuid.UUID
	ExcludeTagIDs []uuid.UUID
	Limit         int
	Offset        int
}

type FoodItemRepository interface {
	GetByID(ctx context.Context, id uuid.UUID, rc RepositoryContext) (food.FoodItemEntity, error)
	Search(ctx context.Context, q FoodItemQuery) ([]food.FoodItemEntity, int, error)
	Create(ctx context.Context, item food.FoodItemEntity) (uuid.UUID, error)
	Update(ctx context.Context, item food.FoodItemEntity) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type MealRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (meal.MealEntity, error)
	Create(ctx context.Context, meal meal.MealEntity) (uuid.UUID, error)
	Update(ctx context.Context, meal meal.MealEntity) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type RecipeRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (recipe.RecipeEntity, error)
	Create(ctx context.Context, recipe recipe.RecipeEntity) (uuid.UUID, error)
	Update(ctx context.Context, recipe recipe.RecipeEntity) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type TagRepository interface {
	List(ctx context.Context, kind tag.Kind) ([]tag.TagEntity, error)
	Upsert(ctx context.Context, tag tag.TagEntity) (uuid.UUID, error)
	AttachToFoodItem(ctx context.Context, foodItemID uuid.UUID, tagID uuid.UUID) error
	RemoveFromFoodItem(ctx context.Context, foodItemID uuid.UUID, tagID uuid.UUID) error
	QueryFoodItemIDs(ctx context.Context, filter FoodItemTagFilter) ([]uuid.UUID, error)
}

type FoodItemTagFilter struct {
	IncludeTagIDs []uuid.UUID
	ExcludeTagIDs []uuid.UUID
}

type MicronutrientVocabularyRepository interface {
	ListActive(ctx context.Context) ([]micronutrient.Entry, error)
	IsAllowed(ctx context.Context, key string) (bool, error)
	Upsert(ctx context.Context, entry micronutrient.Entry) error
}

type UserEntity struct {
	ID           uuid.UUID
	Email        string
	DisplayName  string
	PasswordHash string
	Role         string
	Disabled     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type UserRepository interface {
	Create(ctx context.Context, user UserEntity) (uuid.UUID, error)
	GetByID(ctx context.Context, id uuid.UUID) (UserEntity, error)
	Update(ctx context.Context, user UserEntity) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type PreferenceEntity struct {
	UserID            uuid.UUID
	Theme             string
	DefaultSearchMode string
	EnabledMacros     map[string]bool
	ExcludedTagIDs    []uuid.UUID
	DietaryFilterIDs  []uuid.UUID
	UpdatedAt         time.Time
}

type PreferenceRepository interface {
	Upsert(ctx context.Context, preference PreferenceEntity) error
	GetByUserID(ctx context.Context, userID uuid.UUID) (PreferenceEntity, error)
	Delete(ctx context.Context, userID uuid.UUID) error
}

type EntitlementEntity struct {
	UserID    uuid.UUID
	Plan      string
	Status    string
	ExpiresAt *time.Time
	UpdatedAt time.Time
}

type EntitlementRepository interface {
	Upsert(ctx context.Context, entitlement EntitlementEntity) error
	GetByUserID(ctx context.Context, userID uuid.UUID) (EntitlementEntity, error)
	Delete(ctx context.Context, userID uuid.UUID) error
}

type SavedDataEntity struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Kind      string
	Label     string
	Payload   []byte
	CreatedAt time.Time
}

type SavedDataRepository interface {
	Create(ctx context.Context, saved SavedDataEntity) (uuid.UUID, error)
	GetByID(ctx context.Context, id uuid.UUID) (SavedDataEntity, error)
	Update(ctx context.Context, saved SavedDataEntity) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type AuditLogEntity struct {
	ID        uuid.UUID
	ActorID   *uuid.UUID
	Action    string
	Target    string
	Metadata  []byte
	CreatedAt time.Time
}

type AuditLogRepository interface {
	Create(ctx context.Context, entry AuditLogEntity) (uuid.UUID, error)
	GetByID(ctx context.Context, id uuid.UUID) (AuditLogEntity, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type ImportEntity struct {
	ID         uuid.UUID
	Provider   string
	ExternalID string
	Status     string
	Payload    []byte
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type ImportRepository interface {
	Create(ctx context.Context, importRecord ImportEntity) (uuid.UUID, error)
	GetByID(ctx context.Context, id uuid.UUID) (ImportEntity, error)
	Update(ctx context.Context, importRecord ImportEntity) error
	Delete(ctx context.Context, id uuid.UUID) error
}
