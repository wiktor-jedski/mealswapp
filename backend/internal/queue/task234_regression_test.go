package queue

// Implements DESIGN-004 JobQueueManager and DESIGN-014 MetricsCollector Task 234 load/failure gate.

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
)

// TestTask234MixedAgesRetriesAndStreamRecoveryMatchFinalState verifies
// IT-ARCH-004-007, ARCH-004, DESIGN-004 JobQueueManager,
// DESIGN-014 MetricsCollector, and SW-REQ-080/SW-REQ-082 against real Redis.
func TestTask234MixedAgesRetriesAndStreamRecoveryMatchFinalState(t *testing.T) {
	client := openQueueIntegrationRedis(t)
	manager := newIntegrationQueue(t, client)
	sink := &observability.MemorySink{}
	manager.WithTelemetry(observability.NewOptimizationTelemetry(sink, sink, 1))
	ctx := context.Background()
	retryID, waitingID := uuid.NewString(), uuid.NewString()
	if _, err := manager.Enqueue(ctx, retryID); err != nil {
		t.Fatalf("enqueue retry fixture: %v", err)
	}
	if _, err := manager.Enqueue(ctx, waitingID); err != nil {
		t.Fatalf("enqueue waiting fixture: %v", err)
	}
	pending, err := manager.Reserve(ctx)
	if err != nil {
		t.Fatalf("reserve pending fixture: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	mixed, err := manager.Stats(ctx)
	if err != nil {
		t.Fatalf("mixed Stats(): %v", err)
	}
	if mixed.QueueDepth != 2 || mixed.PendingDepth != 1 || mixed.OldestQueuedAge <= 0 || mixed.OldestPendingAge <= 0 {
		t.Fatalf("mixed stats = %+v, want one waiting, one pending, and positive authoritative ages", mixed)
	}

	manager.config.TerminalHandler = func(context.Context, Job, error) (TerminalPublication, error) { return PublishedFailed, nil }
	for attempt := 1; attempt <= manager.config.MaxAttempts; attempt++ {
		if err := manager.Process(ctx, pending, func(context.Context, Job) (TerminalPublication, error) { return "", errProcessingFixture }); err != nil {
			t.Fatalf("process retry attempt %d: %v", attempt, err)
		}
		if attempt < manager.config.MaxAttempts {
			time.Sleep(10 * time.Millisecond)
			reclaimed, reclaimErr := manager.Reclaim(ctx, time.Millisecond)
			if reclaimErr != nil || len(reclaimed) != 1 {
				t.Fatalf("reclaim attempt %d = %+v, %v", attempt+1, reclaimed, reclaimErr)
			}
			pending = reclaimed[0]
		}
	}
	if got := client.Get(ctx, manager.doneKey(retryID)).Val(); got != failedValue {
		t.Fatalf("retry final marker = %q, want %q", got, failedValue)
	}

	if destroyed := client.XGroupDestroy(ctx, manager.config.Stream, manager.config.Group).Val(); destroyed != 1 {
		t.Fatal("stream recovery fixture did not destroy consumer group")
	}
	recovered, err := manager.Reserve(ctx)
	if err != nil || recovered.ID != waitingID {
		t.Fatalf("Reserve() after group recovery = %+v, %v, want waiting job", recovered, err)
	}
	if err := manager.Process(ctx, recovered, func(context.Context, Job) (TerminalPublication, error) { return PublishedCompleted, nil }); err != nil {
		t.Fatalf("complete recovered job: %v", err)
	}
	if got := client.Get(ctx, manager.doneKey(waitingID)).Val(); got != completedValue {
		t.Fatalf("recovered final marker = %q, want %q", got, completedValue)
	}
	final, err := manager.Stats(ctx)
	if err != nil || final.QueueDepth != 0 || final.PendingDepth != 0 || final.OldestQueuedAge != 0 || final.OldestPendingAge != 0 {
		t.Fatalf("final stats = %+v, %v, want empty zero-age queue", final, err)
	}

	metrics, logs := sink.Snapshot()
	retryOutcomes := []string{}
	queueMetrics := []observability.MetricPoint{}
	for _, point := range metrics {
		if point.Name == observability.MetricOptimizationRetryTotal {
			retryOutcomes = append(retryOutcomes, point.Labels["outcome"])
		}
		if len(point.Labels) > 1 {
			t.Fatalf("unbounded metric labels: %+v", point)
		}
		if point.Name == observability.MetricOptimizationQueueDepth || point.Name == observability.MetricOptimizationQueueAgeSeconds {
			queueMetrics = append(queueMetrics, point)
		}
	}
	if len(retryOutcomes) != 3 || retryOutcomes[0] != "retry" || retryOutcomes[1] != "retry" || retryOutcomes[2] != "exhausted" {
		t.Fatalf("retry outcomes = %v, want [retry retry exhausted]", retryOutcomes)
	}
	wantQueueValues := []float64{2, mixed.OldestQueuedAge.Seconds(), mixed.OldestPendingAge.Seconds(), 0, 0, 0}
	if len(queueMetrics) != len(wantQueueValues) {
		t.Fatalf("queue metrics = %+v, want mixed and final depth/age triplets", queueMetrics)
	}
	for index, want := range wantQueueValues {
		if queueMetrics[index].Value != want {
			t.Fatalf("queue metric %d value = %v, want final-state value %v", index, queueMetrics[index].Value, want)
		}
	}
	payload, marshalErr := json.Marshal(struct{ Metrics, Logs any }{metrics, logs})
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	for _, forbidden := range []string{retryID, waitingID, errProcessingFixture.Error(), "job_id", "entry_id"} {
		if bytes.Contains(payload, []byte(forbidden)) {
			t.Fatalf("queue telemetry leaked %q: %s", forbidden, payload)
		}
	}
	if pendingCount := client.XPending(ctx, manager.config.Stream, manager.config.Group).Val().Count; pendingCount != 0 {
		t.Fatalf("final pending count = %d, want zero", pendingCount)
	}
	if entries := client.XLen(ctx, manager.config.Stream).Val(); entries != 0 {
		t.Fatalf("final stream length = %d, want zero", entries)
	}
}
