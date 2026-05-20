package worker

import (
	"context"
	"testing"

	"mealswapp/backend/internal/services/optimization"

	"github.com/google/uuid"
)

func TestRunOneOptimizationJobReservesAndProcessesQueuedJob(t *testing.T) {
	queue := &fakeOptimizationQueue{
		job: optimization.OptimizationJob{JobID: uuid.New(), Status: optimization.JobStatusProcessing},
		ok:  true,
	}
	processor := &fakeOptimizationProcessor{}

	processed, err := RunOneOptimizationJob(context.Background(), queue, processor)
	if err != nil {
		t.Fatal(err)
	}
	if !processed || processor.job.JobID != queue.job.JobID {
		t.Fatalf("expected reserved job processed, processed=%v job=%#v", processed, processor.job)
	}
}

func TestRunOneOptimizationJobReturnsFalseWhenQueueEmpty(t *testing.T) {
	processed, err := RunOneOptimizationJob(context.Background(), &fakeOptimizationQueue{}, &fakeOptimizationProcessor{})
	if err != nil {
		t.Fatal(err)
	}
	if processed {
		t.Fatal("expected empty queue to return processed=false")
	}
}

type fakeOptimizationQueue struct {
	job optimization.OptimizationJob
	ok  bool
}

func (queue *fakeOptimizationQueue) Reserve(ctx context.Context) (optimization.OptimizationJob, bool, error) {
	return queue.job, queue.ok, nil
}

type fakeOptimizationProcessor struct {
	job optimization.OptimizationJob
}

func (processor *fakeOptimizationProcessor) ProcessOptimizationJob(ctx context.Context, job optimization.OptimizationJob) error {
	processor.job = job
	return nil
}
