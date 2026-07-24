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
	"github.com/wiktor-jedski/mealswapp/backend/internal/deletionworker"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
	"github.com/wiktor-jedski/mealswapp/backend/internal/optimization"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/userdata"
	"github.com/wiktor-jedski/mealswapp/backend/internal/worker"
	"golang.org/x/sync/errgroup"
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
	foodRepository := repository.NewPostgresFoodItemRepository(pg)
	dietRepository := repository.NewPostgresSavedDataRepository(pg)
	inputs := worker.NewRepositoryOptimizationInputLoader(optimization.NewConstraintBuilder(mealRepository, dietRepository, foodRepository))
	solver := optimization.NewLPSolverWrapper(optimization.CLPConfig{
		Executable:      cfg.CLPExecutable,
		ExpectedVersion: cfg.CLPVersion,
	})
	processor := worker.NewOptimizationProcessor(store, inputs, solver).WithTelemetry(telemetry)
	processor.WithAdmissionGate(worker.NewRedisOptimizationAdmissionGate(redisClient, worker.OptimizationAdmissionConfig{}))
	deletionService := userdata.NewAccountDeletionService(
		repository.NewPostgresComplianceRepository(pg),
		repository.NewPostgresSessionRepository(pg),
		repository.NewPostgresEncryptedIdentityRepository(pg),
		cache.NewUserPurger(redisClient),
	)
	// Compose the complete processor at the dedicated worker boundary; the API
	// process never runs optimization synchronously.
	group, workerCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		return worker.RunWithProcessorAndTelemetry(workerCtx, cfg, redisClient, processor.ProcessOptimizationJob, telemetry, processor.Terminal)
	})
	group.Go(func() error {
		return deletionworker.RunAccountDeletionProcessor(workerCtx, deletionService, 0, 0, telemetrySink)
	})
	if err := group.Wait(); err != nil {
		log.Fatalf("worker stopped: %v", err)
	}
}
