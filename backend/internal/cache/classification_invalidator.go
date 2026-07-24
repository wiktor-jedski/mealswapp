package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// FilterOptionInvalidator discards the process-local classification projection.
// Implements DESIGN-009 TagManager cache invalidation.
type FilterOptionInvalidator interface {
	Invalidate()
}

// ClassificationInvalidator clears filter options and cached search responses after commit.
// Implements DESIGN-009 TagManager cache invalidation.
type ClassificationInvalidator struct {
	filter     FilterOptionInvalidator
	redis      classificationRedisInvalidator
	generation ClassificationGeneration
}

// classificationRedisInvalidator is the Redis subset needed for bounded namespace invalidation.
// Implements DESIGN-009 TagManager cache invalidation.
type classificationRedisInvalidator interface {
	Scan(context.Context, uint64, string, int64) *redis.ScanCmd
	Del(context.Context, ...string) *redis.IntCmd
}

// NewClassificationInvalidator creates post-commit classification-derived cache invalidation.
// Implements DESIGN-009 TagManager cache invalidation.
func NewClassificationInvalidator(filter FilterOptionInvalidator, redisClient *redis.Client) ClassificationInvalidator {
	invalidator := ClassificationInvalidator{filter: filter}
	if redisClient != nil {
		invalidator.redis = redisClient
		invalidator.generation = NewClassificationGeneration(redisClient)
	}
	return invalidator
}

// Invalidate synchronously drops local options and best-effort deletes bounded Redis search batches.
// Implements DESIGN-009 TagManager cache invalidation.
func (i ClassificationInvalidator) Invalidate() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if i.generation.client != nil {
		if _, err := i.generation.Advance(ctx); err != nil {
			if i.filter != nil {
				i.filter.Invalidate()
			}
			return
		}
	}
	if i.filter != nil {
		i.filter.Invalidate()
	}
	if i.redis == nil {
		return
	}
	var cursor uint64
	for page := 0; page < 1000; page++ {
		keys, next, err := i.redis.Scan(ctx, cursor, string(RedisNamespaceSearch)+":"+SearchSchemaVersion+"*:*", 100).Result()
		if err != nil {
			return
		}
		if len(keys) > 0 && i.redis.Del(ctx, keys...).Err() != nil {
			return
		}
		cursor = next
		if cursor == 0 {
			return
		}
	}
}
