package cache

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// classificationGenerationKey stores the global invalidation generation.
// Implements DESIGN-011 CacheInvalidator shared invalidation.
const classificationGenerationKey = "classification:cache-generation:v1"

// ClassificationGeneration coordinates classification-derived caches across API instances.
// Implements DESIGN-009 TagManager and DESIGN-011 CacheInvalidator shared invalidation.
type ClassificationGeneration struct {
	client classificationGenerationClient
}

// classificationGenerationClient is the Redis command subset used for generation coordination.
// Implements DESIGN-011 RedisCache shared generation versioning.
type classificationGenerationClient interface {
	redis.Scripter
	Get(context.Context, string) *redis.StringCmd
	Incr(context.Context, string) *redis.IntCmd
}

// NewClassificationGeneration creates a Redis-backed shared cache generation.
// Implements DESIGN-011 CacheInvalidator shared invalidation.
func NewClassificationGeneration(client *redis.Client) ClassificationGeneration {
	if client == nil {
		return ClassificationGeneration{}
	}
	return ClassificationGeneration{client: client}
}

// Current returns the shared classification generation, treating an absent key as generation zero.
// Implements DESIGN-011 RedisCache versioned lookup.
func (g ClassificationGeneration) Current(ctx context.Context) (uint64, error) {
	if g.client == nil {
		return 0, nil
	}
	raw, err := g.client.Get(ctx, classificationGenerationKey).Result()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(raw, 10, 64)
}

// Advance atomically invalidates every older classification-derived cache generation.
// Implements DESIGN-009 TagManager post-commit invalidation.
func (g ClassificationGeneration) Advance(ctx context.Context) (uint64, error) {
	if g.client == nil {
		return 0, nil
	}
	return g.client.Incr(ctx, classificationGenerationKey).Uint64()
}

// SetIfCurrent writes only when no classification invalidation occurred since the cache miss.
// Implements DESIGN-011 RedisCache guarded cache-miss persistence.
func (g ClassificationGeneration) SetIfCurrent(ctx context.Context, generation uint64, key, value string, ttl time.Duration) (bool, error) {
	if g.client == nil || ttl <= 0 {
		return false, nil
	}
	result, err := setIfClassificationGenerationScript.Run(ctx, g.client, []string{classificationGenerationKey, key}, strconv.FormatUint(generation, 10), value, strconv.FormatInt(ttl.Milliseconds(), 10)).Int64()
	return result == 1, err
}
