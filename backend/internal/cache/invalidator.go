package cache

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
)

type InvalidationReason string

const (
	ReasonFoodUpdated   InvalidationReason = "food_updated"
	ReasonRecipeUpdated InvalidationReason = "recipe_updated"
	ReasonTagUpdated    InvalidationReason = "tag_updated"
	ReasonImportChanged InvalidationReason = "import_changed"
)

type InvalidationEvent struct {
	ItemIDs   []uuid.UUID
	TagIDs    []uuid.UUID
	UserID    *uuid.UUID
	Reason    InvalidationReason
	CreatedAt time.Time
}

type TaggedCacheStore interface {
	DeleteByTags(ctx context.Context, tags []string) (int, error)
}

type Invalidator struct {
	store TaggedCacheStore
}

type InvalidationResult struct {
	TagsDeleted []string
	KeysDeleted int
}

func NewInvalidator(store TaggedCacheStore) Invalidator {
	return Invalidator{store: store}
}

func (invalidator Invalidator) Invalidate(ctx context.Context, event InvalidationEvent) (InvalidationResult, error) {
	tags := InvalidationTags(event)
	if len(tags) == 0 {
		return InvalidationResult{TagsDeleted: []string{}}, nil
	}

	deleted, err := invalidator.store.DeleteByTags(ctx, tags)
	if err != nil {
		return InvalidationResult{}, err
	}
	return InvalidationResult{TagsDeleted: tags, KeysDeleted: deleted}, nil
}

func InvalidationTags(event InvalidationEvent) []string {
	tags := make([]string, 0)
	switch event.Reason {
	case ReasonFoodUpdated, ReasonRecipeUpdated, ReasonImportChanged:
		tags = appendUniqueTag(tags, "namespace:search")
		tags = appendUniqueTag(tags, "namespace:similarity")
	case ReasonTagUpdated:
		tags = appendUniqueTag(tags, "namespace:search")
	}

	for _, itemID := range event.ItemIDs {
		if itemID == uuid.Nil {
			continue
		}
		tags = appendUniqueTag(tags, "item:"+itemID.String())
	}
	for _, tagID := range event.TagIDs {
		if tagID == uuid.Nil {
			continue
		}
		tags = appendUniqueTag(tags, "tag:"+tagID.String())
	}
	if event.UserID != nil && *event.UserID != uuid.Nil {
		tags = appendUniqueTag(tags, "user:"+event.UserID.String())
	}
	return tags
}

func appendUniqueTag(tags []string, tag string) []string {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return tags
	}
	for _, existing := range tags {
		if existing == tag {
			return tags
		}
	}
	return append(tags, tag)
}
