package cache

import (
	"github.com/redis/go-redis/v9"
)

// Open creates a Redis client from the configured Redis URL.
//
// Implements DESIGN-011 RedisCache connection factory.
func Open(redisURL string) (*redis.Client, error) {
	options, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	return redis.NewClient(options), nil
}
