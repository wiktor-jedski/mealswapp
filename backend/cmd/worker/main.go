package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/wiktor-jedski/mealswapp/backend/internal/cache"
	"github.com/wiktor-jedski/mealswapp/backend/internal/config"
	"github.com/wiktor-jedski/mealswapp/backend/internal/worker"
)

// Implements DESIGN-004 JobQueueManager worker process bootstrap.
func main() {
	// load env
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// create cache
	redisClient, err := cache.Open(cfg.RedisURL)
	if err != nil {
		log.Fatalf("open redis: %v", err)
	}
	defer redisClient.Close()

	// create context that can be passed to stop the func
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// run worker using internal func
	// can error if pinging redis fails
	if err := worker.Run(ctx, cfg, redisClient); err != nil {
		log.Fatalf("worker stopped: %v", err)
	}
}
