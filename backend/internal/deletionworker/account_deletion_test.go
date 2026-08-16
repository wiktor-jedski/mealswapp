package deletionworker

// Implements DESIGN-008 AccountDeleter production scheduler verification.

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

type accountDeletionProcessorStub struct {
	mu     sync.Mutex
	calls  int
	cancel context.CancelFunc
}

type accountDeletionProcessorFunc func(context.Context, time.Time, int) ([]repository.DataDeletionRequest, error)

type doneAfterErrContext struct {
	context.Context
	done chan struct{}
	once sync.Once
}

func (c *doneAfterErrContext) Done() <-chan struct{} { return c.done }

func (c *doneAfterErrContext) Err() error {
	c.once.Do(func() { close(c.done) })
	return nil
}

func (f accountDeletionProcessorFunc) ProcessDueDeletionRequests(ctx context.Context, now time.Time, limit int) ([]repository.DataDeletionRequest, error) {
	return f(ctx, now, limit)
}

func (p *accountDeletionProcessorStub) ProcessDueDeletionRequests(context.Context, time.Time, int) ([]repository.DataDeletionRequest, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.calls++
	if p.calls == 1 {
		return nil, errors.New("temporary database failure")
	}
	p.cancel()
	return []repository.DataDeletionRequest{{ID: uuid.New()}}, nil
}

func TestRunAccountDeletionProcessorRetriesAndReportsBoundedMetrics(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	processor := &accountDeletionProcessorStub{cancel: cancel}
	metrics := &observability.MemorySink{}
	if err := RunAccountDeletionProcessor(ctx, processor, time.Millisecond, 2, metrics); err != nil {
		t.Fatalf("RunAccountDeletionProcessor() error = %v", err)
	}
	processor.mu.Lock()
	calls := processor.calls
	processor.mu.Unlock()
	points, _ := metrics.Snapshot()
	if calls != 2 || len(points) != 3 {
		t.Fatalf("calls=%d metrics=%+v", calls, points)
	}
	if points[0].Name != MetricAccountDeletionCycles || points[0].Labels["status"] != "failed" || points[1].Labels["status"] != "completed" || points[2].Name != MetricAccountDeletionRequests || len(points[2].Labels) != 0 {
		t.Fatalf("account deletion metrics = %+v", points)
	}
}

func TestRunAccountDeletionProcessorRejectsMissingDependencies(t *testing.T) {
	if err := RunAccountDeletionProcessor(nil, &accountDeletionProcessorStub{}, time.Second, 1, nil); err == nil {
		t.Fatal("nil context accepted")
	}
	if err := RunAccountDeletionProcessor(context.Background(), nil, time.Second, 1, nil); err == nil {
		t.Fatal("nil processor accepted")
	}
}

func TestRunAccountDeletionProcessorDefaultsAndStopsAfterCanceledFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	processor := accountDeletionProcessorFunc(func(_ context.Context, _ time.Time, limit int) ([]repository.DataDeletionRequest, error) {
		if limit != defaultAccountDeletionBatch {
			t.Fatalf("default limit=%d, want %d", limit, defaultAccountDeletionBatch)
		}
		cancel()
		return nil, errors.New("database unavailable during shutdown")
	})
	if err := RunAccountDeletionProcessor(ctx, processor, 0, 0, nil); err != nil {
		t.Fatalf("RunAccountDeletionProcessor() error = %v", err)
	}
}

func TestRunAccountDeletionProcessorStopsAfterTickerFailureAndCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	processor := accountDeletionProcessorFunc(func(context.Context, time.Time, int) ([]repository.DataDeletionRequest, error) {
		calls++
		if calls == 1 {
			return nil, nil
		}
		cancel()
		return nil, errors.New("database unavailable during shutdown")
	})
	if err := RunAccountDeletionProcessor(ctx, processor, time.Millisecond, 1, &observability.MemorySink{}); err != nil {
		t.Fatalf("RunAccountDeletionProcessor() error = %v", err)
	}
	if calls != 2 {
		t.Fatalf("processor calls=%d, want 2", calls)
	}
}

func TestRunAccountDeletionProcessorStopsWhenContextCompletesBeforeFirstTick(t *testing.T) {
	ctx := &doneAfterErrContext{Context: context.Background(), done: make(chan struct{})}
	processor := accountDeletionProcessorFunc(func(context.Context, time.Time, int) ([]repository.DataDeletionRequest, error) {
		return nil, nil
	})
	if err := RunAccountDeletionProcessor(ctx, processor, time.Hour, 1, nil); err != nil {
		t.Fatalf("RunAccountDeletionProcessor() error = %v", err)
	}
}
