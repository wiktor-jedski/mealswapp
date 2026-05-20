package externaldata

import (
	"context"
	"errors"
	"strings"
	"time"

	"mealswapp/backend/internal/cache"
	"mealswapp/backend/internal/domain/tag"

	"github.com/google/uuid"
)

var ErrTagMergeInvalid = errors.New("tag merge requires distinct source and target ids")

type TagManagerStore interface {
	List(ctx context.Context, kind tag.Kind) ([]tag.TagEntity, error)
	Upsert(ctx context.Context, tag tag.TagEntity) (uuid.UUID, error)
	AttachToFoodItem(ctx context.Context, foodItemID uuid.UUID, tagID uuid.UUID) error
	RemoveFromFoodItem(ctx context.Context, foodItemID uuid.UUID, tagID uuid.UUID) error
	Deactivate(ctx context.Context, id uuid.UUID) error
	Merge(ctx context.Context, sourceID uuid.UUID, targetID uuid.UUID) error
}

type TagManager struct {
	tags        TagManagerStore
	invalidator CacheInvalidator
	now         func() time.Time
}

type TagAssignment struct {
	FoodItemID uuid.UUID `json:"foodItemId"`
	TagID      uuid.UUID `json:"tagId"`
}

type TagMergeRequest struct {
	SourceID uuid.UUID `json:"sourceId"`
	TargetID uuid.UUID `json:"targetId"`
}

func NewTagManager(tags TagManagerStore, invalidator CacheInvalidator) TagManager {
	return TagManager{tags: tags, invalidator: invalidator, now: time.Now}
}

func (manager TagManager) List(ctx context.Context, kind tag.Kind) ([]tag.TagEntity, error) {
	if !kind.Valid() {
		return nil, tag.ErrInvalidKind
	}
	return manager.tags.List(ctx, kind)
}

func (manager TagManager) Upsert(ctx context.Context, entity tag.TagEntity) (tag.TagEntity, error) {
	entity.Name = strings.TrimSpace(entity.Name)
	if err := entity.Validate(); err != nil {
		return tag.TagEntity{}, err
	}
	if !entity.Active {
		entity.Active = true
	}
	id, err := manager.tags.Upsert(ctx, entity)
	if err != nil {
		return tag.TagEntity{}, err
	}
	entity.ID = id
	if err := manager.invalidate(ctx, nil, []uuid.UUID{id}); err != nil {
		return tag.TagEntity{}, err
	}
	return entity, nil
}

func (manager TagManager) Assign(ctx context.Context, foodItemID uuid.UUID, tagID uuid.UUID) error {
	if foodItemID == uuid.Nil || tagID == uuid.Nil {
		return ErrTagMergeInvalid
	}
	if err := manager.tags.AttachToFoodItem(ctx, foodItemID, tagID); err != nil {
		return err
	}
	return manager.invalidate(ctx, []uuid.UUID{foodItemID}, []uuid.UUID{tagID})
}

func (manager TagManager) Remove(ctx context.Context, foodItemID uuid.UUID, tagID uuid.UUID) error {
	if foodItemID == uuid.Nil || tagID == uuid.Nil {
		return ErrTagMergeInvalid
	}
	if err := manager.tags.RemoveFromFoodItem(ctx, foodItemID, tagID); err != nil {
		return err
	}
	return manager.invalidate(ctx, []uuid.UUID{foodItemID}, []uuid.UUID{tagID})
}

func (manager TagManager) Deactivate(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return ErrTagMergeInvalid
	}
	if err := manager.tags.Deactivate(ctx, id); err != nil {
		return err
	}
	return manager.invalidate(ctx, nil, []uuid.UUID{id})
}

func (manager TagManager) Merge(ctx context.Context, sourceID uuid.UUID, targetID uuid.UUID) error {
	if sourceID == uuid.Nil || targetID == uuid.Nil || sourceID == targetID {
		return ErrTagMergeInvalid
	}
	if err := manager.tags.Merge(ctx, sourceID, targetID); err != nil {
		return err
	}
	return manager.invalidate(ctx, nil, []uuid.UUID{sourceID, targetID})
}

func (manager TagManager) invalidate(ctx context.Context, itemIDs []uuid.UUID, tagIDs []uuid.UUID) error {
	if manager.invalidator == nil {
		return nil
	}
	_, err := manager.invalidator.Invalidate(ctx, cache.InvalidationEvent{ItemIDs: itemIDs, TagIDs: tagIDs, Reason: cache.ReasonTagUpdated, CreatedAt: manager.now().UTC()})
	return err
}
