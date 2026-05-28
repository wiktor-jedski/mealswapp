package worker

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
	"github.com/wiktor-jedski/mealswapp/backend/internal/config"
)

// Run starts the Phase 00 worker lifecycle and blocks until the context is canceled.
//
// Implements DESIGN-004 JobQueueManager worker lifecycle placeholder.
func Run(ctx context.Context, cfg config.Config, redisClient *redis.Client) error {
	return runAfterPing(ctx, cfg, func(ctx context.Context) error {
		// ping redis
		return redisClient.Ping(ctx).Err()
	})
}

func runAfterPing(ctx context.Context, cfg config.Config, ping func(context.Context) error) error {
	// ping context (here redis)
	log.Printf("worker started env=%s", cfg.Environment)
	if err := ping(ctx); err != nil {
		return err
	}
	// block until context dies
	<-ctx.Done()
	log.Printf("worker stopped: %v", ctx.Err())
	return nil
}
