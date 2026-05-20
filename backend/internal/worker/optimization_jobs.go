package worker

import (
	"context"

	"mealswapp/backend/internal/services/optimization"
)

type OptimizationQueue interface {
	Reserve(ctx context.Context) (optimization.OptimizationJob, bool, error)
}

type OptimizationProcessor interface {
	ProcessOptimizationJob(ctx context.Context, job optimization.OptimizationJob) error
}

func RunOneOptimizationJob(ctx context.Context, queue OptimizationQueue, processor OptimizationProcessor) (bool, error) {
	job, ok, err := queue.Reserve(ctx)
	if err != nil || !ok {
		return ok, err
	}
	if err := processor.ProcessOptimizationJob(ctx, job); err != nil {
		return true, err
	}
	return true, nil
}
