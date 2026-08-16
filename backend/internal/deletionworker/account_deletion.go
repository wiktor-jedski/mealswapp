package deletionworker

import (
	"context"
	"errors"
	"time"

	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// Implements DESIGN-008 AccountDeleter scheduler defaults and DESIGN-014 MetricsCollector names.
const (
	defaultAccountDeletionInterval = 30 * time.Second
	defaultAccountDeletionBatch    = 10

	// MetricAccountDeletionCycles counts sanitized scheduler outcomes.
	// Implements DESIGN-014 MetricsCollector and DESIGN-008 AccountDeleter.
	MetricAccountDeletionCycles = "account_deletion_processor_cycles_total"
	// MetricAccountDeletionRequests counts requests claimed by successful cycles.
	// Implements DESIGN-014 MetricsCollector and DESIGN-008 AccountDeleter.
	MetricAccountDeletionRequests = "account_deletion_processor_requests_total"
)

// AccountDeletionProcessor claims and executes due account erasure requests.
// Implements DESIGN-008 AccountDeleter production worker boundary.
type AccountDeletionProcessor interface {
	ProcessDueDeletionRequests(context.Context, time.Time, int) ([]repository.DataDeletionRequest, error)
}

// RunAccountDeletionProcessor polls immediately and then periodically until cancellation.
// Operational failures are observed with bounded labels and retried on the next cycle.
// Implements DESIGN-008 AccountDeleter production worker execution.
func RunAccountDeletionProcessor(ctx context.Context, processor AccountDeletionProcessor, interval time.Duration, limit int, metrics observability.MetricsCollector) error {
	if ctx == nil {
		return errors.New("account deletion processor context is required")
	}
	if processor == nil {
		return errors.New("account deletion processor is required")
	}
	if interval <= 0 {
		interval = defaultAccountDeletionInterval
	}
	if limit <= 0 {
		limit = defaultAccountDeletionBatch
	}

	run := func() bool {
		claimed, err := processor.ProcessDueDeletionRequests(ctx, time.Now().UTC(), limit)
		if err != nil {
			recordAccountDeletionMetric(ctx, metrics, MetricAccountDeletionCycles, 1, "cycles", map[string]string{"status": "failed"})
			return ctx.Err() == nil
		}
		recordAccountDeletionMetric(ctx, metrics, MetricAccountDeletionCycles, 1, "cycles", map[string]string{"status": "completed"})
		if len(claimed) > 0 {
			recordAccountDeletionMetric(ctx, metrics, MetricAccountDeletionRequests, float64(len(claimed)), "requests", nil)
		}
		return ctx.Err() == nil
	}
	if !run() {
		return nil
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if !run() {
				return nil
			}
		}
	}
}

// recordAccountDeletionMetric emits only fixed metric names, units, and labels.
// Implements DESIGN-014 MetricsCollector and DESIGN-008 AccountDeleter.
func recordAccountDeletionMetric(ctx context.Context, metrics observability.MetricsCollector, name string, value float64, unit string, labels map[string]string) {
	if metrics == nil {
		return
	}
	_ = metrics.RecordMetric(ctx, observability.MetricPoint{Name: name, Value: value, Unit: unit, Labels: labels, ObservedAt: time.Now().UTC()})
}
