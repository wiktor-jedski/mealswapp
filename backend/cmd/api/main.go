package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mealswapp/mealswapp/backend/internal/app"
	"github.com/mealswapp/mealswapp/backend/internal/cache"
	"github.com/mealswapp/mealswapp/backend/internal/config"
	"github.com/mealswapp/mealswapp/backend/internal/database"
	"github.com/mealswapp/mealswapp/backend/internal/httpapi"
)

func main() {
	// Implements DESIGN-010 RouteHandler API process bootstrap.
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx := context.Background()
	pg, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("open postgres: %v", err)
	}
	defer pg.Close()

	redisClient, err := cache.Open(cfg.RedisURL)
	if err != nil {
		log.Fatalf("open redis: %v", err)
	}
	defer redisClient.Close()

	deps := httpapi.Dependencies{
		Config: cfg,
		PostgresPing: func(ctx context.Context) error {
			return pg.Ping(ctx)
		},
		RedisPing: func(ctx context.Context) error {
			return redisClient.Ping(ctx).Err()
		},
	}
	server := app.New(deps)

	errs := make(chan error, 1)
	go func() {
		errs <- server.Listen(":" + cfg.HTTPPort)
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errs:
		log.Fatalf("api server stopped: %v", err)
	case <-stop:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.ShutdownWithContext(ctx); err != nil {
			log.Fatalf("shutdown api server: %v", err)
		}
	}
}
