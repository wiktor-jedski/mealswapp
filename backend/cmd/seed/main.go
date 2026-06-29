package main

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"github.com/wiktor-jedski/mealswapp/backend/internal/cache"
	"github.com/wiktor-jedski/mealswapp/backend/internal/config"
	"github.com/wiktor-jedski/mealswapp/backend/internal/seed"
)

// main loads configuration, connects to PostgreSQL, and runs development seeding.
// Implements DESIGN-005 MicronutrientVocabulary.
func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer conn.Close(ctx)

	if err := seed.Run(ctx, conn); err != nil {
		log.Fatalf("run seed: %v", err)
	}
	if err := clearDevelopmentSearchCaches(ctx, cfg.RedisURL); err != nil {
		log.Printf("clear development search caches: %v", err)
	}
}

// clearDevelopmentSearchCaches removes search result caches after development fixture changes.
// Implements DESIGN-011 RedisCache development cache invalidation for DESIGN-005 seed data.
func clearDevelopmentSearchCaches(ctx context.Context, redisURL string) error {
	client, err := cache.Open(redisURL)
	if err != nil {
		return err
	}
	defer client.Close()

	for _, pattern := range []string{"search:*", "autocomplete:*", "similarity:*"} {
		if err := deletePattern(ctx, client, pattern); err != nil {
			return err
		}
	}
	return nil
}

// deletePattern scans and deletes matching Redis keys without blocking on a full keyspace command.
// Implements DESIGN-011 RedisCache namespace-scoped development invalidation.
func deletePattern(ctx context.Context, client *redis.Client, pattern string) error {
	var cursor uint64
	for {
		keys, nextCursor, err := client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			if err := client.Del(ctx, keys...).Err(); err != nil {
				return err
			}
		}
		if nextCursor == 0 {
			return nil
		}
		cursor = nextCursor
	}
}
