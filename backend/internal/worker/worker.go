package worker

import (
	"context"
	"log"

	"github.com/mealswapp/mealswapp/backend/internal/config"
	"github.com/redis/go-redis/v9"
)

// Run starts the Phase 00 worker lifecycle and blocks until the context is canceled.
//
// Implements DESIGN-004 JobQueueManager worker lifecycle placeholder.
func Run(ctx context.Context, cfg config.Config, redisClient *redis.Client) error {
	return runAfterPing(ctx, cfg, func(ctx context.Context) error {
		return redisClient.Ping(ctx).Err()
	})
}

func runAfterPing(ctx context.Context, cfg config.Config, ping func(context.Context) error) error {
	log.Printf("worker started env=%s", cfg.Environment)
	if err := ping(ctx); err != nil {
		return err
	}
	<-ctx.Done()
	log.Print("worker stopped")
	return nil
}
