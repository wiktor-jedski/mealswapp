package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wiktor-jedski/mealswapp/backend/internal/app"
	"github.com/wiktor-jedski/mealswapp/backend/internal/cache"
	"github.com/wiktor-jedski/mealswapp/backend/internal/config"
	"github.com/wiktor-jedski/mealswapp/backend/internal/database"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
)

// main starts the HTTP API process.
// Implements DESIGN-010 RouteHandler API process bootstrap.
func main() {
	// loading env variables using internal config module
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// creating root context to rule them all
	// no values, no deadline, cannot be cancelled
	ctx := context.Background()

	// postgres init
	pg, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("open postgres: %v", err)
	}
	defer pg.Close()

	// redis init
	redisClient, err := cache.Open(cfg.RedisURL)
	if err != nil {
		log.Fatalf("open redis: %v", err)
	}
	defer redisClient.Close()

	// initiating the app that can use PostgresPing
	// and RedisPing to check for readiness
	observabilitySink := observability.JSONSink{Writer: os.Stdout}
	server, err := app.NewProduction(cfg, pg, redisClient, observabilitySink)
	if err != nil {
		log.Fatalf("build server: %v", err)
	}

	// error channel
	// if server breaks, then it passes the error
	errs := make(chan error, 1)
	go func() {
		errs <- server.Listen(":" + cfg.HTTPPort)
	}()

	// signal handling
	// can be stopped gracefully by these signals
	// app still can be crashed by SIGKILL
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// running and waiting for halting
	select {
	case err := <-errs:
		if err != nil {
			log.Fatalf("api server stopped: %v", err)
		}
		log.Print("api server stopped gracefully")
	case <-stop:
		// 5 secs for shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		// most possible errors: past 5 seconds, listener closing error, server not running
		if err := server.ShutdownWithContext(ctx); err != nil {
			log.Fatalf("shutdown api server: %v", err)
		}
	}
}
