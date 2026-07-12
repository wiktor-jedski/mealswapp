package worker

import (
	"context"
	"errors"
	"log"

	"github.com/redis/go-redis/v9"
	"github.com/wiktor-jedski/mealswapp/backend/internal/config"
	"github.com/wiktor-jedski/mealswapp/backend/internal/optimization"
	"github.com/wiktor-jedski/mealswapp/backend/internal/queue"
)

// Run starts the worker bootstrap and consumes optimization jobs through the
// production processor.
// Implements DESIGN-004 JobQueueManager worker process bootstrap.
func Run(ctx context.Context, cfg config.Config, redisClient *redis.Client) error {
	return RunWithProcessor(ctx, cfg, redisClient, ProcessOptimizationJob)
}

// RunWithProcessor starts the Redis Streams consumer after Redis and CLP
// readiness checks. Only this dedicated worker process invokes the processor;
// the Fiber API has no synchronous solving fallback.
// Implements DESIGN-004 JobQueueManager and LPSolverWrapper worker boundary.
func RunWithProcessor(ctx context.Context, cfg config.Config, redisClient *redis.Client, processor queue.Processor, terminalHandlers ...queue.TerminalHandler) error {
	if redisClient == nil {
		return errors.New("worker Redis client is required")
	}
	if processor == nil {
		return errors.New("worker processor is required")
	}
	if err := redisClient.Ping(ctx).Err(); err != nil {
		return err
	}
	// The worker owns solver readiness; the Fiber API never invokes CLP.
	if err := optimization.NewLPSolverWrapper(optimization.CLPConfig{
		Executable:      cfg.CLPExecutable,
		ExpectedVersion: cfg.CLPVersion,
	}).StartupCheck(ctx); err != nil {
		return err
	}
	if len(terminalHandlers) > 1 {
		return errors.New("only one worker terminal handler is supported")
	}
	queueConfig := queue.Config{}
	if len(terminalHandlers) == 1 {
		queueConfig.TerminalHandler = terminalHandlers[0]
	}
	manager := queue.NewJobQueueManager(redisClient, queueConfig)
	if err := manager.Bootstrap(ctx); err != nil {
		return err
	}
	stopHeartbeat, err := startWorkerHeartbeat(ctx, redisClient, manager.Config().Consumer)
	if err != nil {
		return err
	}
	defer stopHeartbeat()
	return manager.Run(ctx, processor)
}

// ProcessOptimizationJob is a fail-closed compatibility seam. Production
// startup must compose NewOptimizationProcessor and pass its method so a nil
// dependency cannot turn a queue delivery into false success.
// Implements DESIGN-004 JobQueueManager worker execution.
func ProcessOptimizationJob(ctx context.Context, job queue.Job) error {
	if ctx == nil {
		return errors.New("worker processor context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if job.ID == "" {
		return queue.ErrInvalidJob
	}
	return errors.New("optimization processor dependencies are required")
}

// runAfterPing keeps the legacy testable lifecycle seam for bootstrap checks.
// Implements DESIGN-004 JobQueueManager worker lifecycle.
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
