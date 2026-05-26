package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mealswapp/mealswapp/backend/internal/cache"
	"github.com/mealswapp/mealswapp/backend/internal/config"
	"github.com/mealswapp/mealswapp/backend/internal/worker"
)

func main() {
	// Implements DESIGN-004 JobQueueManager worker process bootstrap.
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	redisClient, err := cache.Open(cfg.RedisURL)
	if err != nil {
		log.Fatalf("open redis: %v", err)
	}
	defer redisClient.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := worker.Run(ctx, cfg, redisClient); err != nil {
		log.Fatalf("worker stopped: %v", err)
	}
}
