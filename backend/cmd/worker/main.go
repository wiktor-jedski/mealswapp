package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/wiktor-jedski/mealswapp/backend/internal/cache"
	"github.com/wiktor-jedski/mealswapp/backend/internal/config"
	"github.com/wiktor-jedski/mealswapp/backend/internal/database"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
	"github.com/wiktor-jedski/mealswapp/backend/internal/optimization"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/worker"
)

// main starts the background worker process.
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

	pg, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("open postgres: %v", err)
	}
	defer pg.Close()

	telemetrySink := observability.JSONSink{Writer: os.Stdout}
	telemetry := observability.NewOptimizationTelemetry(telemetrySink, telemetrySink, 1)
	store := worker.NewRedisOptimizationJobStore(redisClient).WithTelemetry(telemetry)
	mealRepository := repository.NewPostgresMealRepository(pg)
	dietRepository := repository.NewPostgresSavedDataRepository(pg)
	inputs := worker.NewRepositoryOptimizationInputLoader(optimization.NewConstraintBuilder(mealRepository, dietRepository))
	solver := optimization.NewLPSolverWrapper(optimization.CLPConfig{
		Executable:      cfg.CLPExecutable,
		ExpectedVersion: cfg.CLPVersion,
	})
	processor := worker.NewOptimizationProcessor(store, inputs, solver).WithTelemetry(telemetry)
	// Compose the complete processor at the dedicated worker boundary; the API
	// process never runs optimization synchronously.
	if err := worker.RunWithProcessor(ctx, cfg, redisClient, processor.ProcessOptimizationJob, processor.Terminal); err != nil {
		log.Fatalf("worker stopped: %v", err)
	}
}
