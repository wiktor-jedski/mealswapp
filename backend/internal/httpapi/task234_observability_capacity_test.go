package httpapi

// Implements DESIGN-014 MetricsCollector Task 234 concurrent submission and endpoint-capacity gate.

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
	"github.com/wiktor-jedski/mealswapp/backend/internal/worker"
)

// TestTask234ConcurrentSubmissionsReplayCleanupAndPollResponsiveness verifies
// IT-ARCH-004-007, ARCH-004, DESIGN-004 JobStatusTracker,
// DESIGN-014 MetricsCollector, and SW-REQ-080/SW-REQ-082 under concurrent,
// replay, cleanup-failure, active-worker polling, and degraded collaboration.
func TestTask234ConcurrentSubmissionsReplayCleanupAndPollResponsiveness(t *testing.T) {
	const submissions = 8
	userID, dietID := uuid.New(), uuid.New()
	store, queue := newOptimizationHTTPJobStore(), &optimizationHTTPQueue{}
	sink := &observability.MemorySink{}
	controller := NewOptimizationController(store, queue, &optimizationHTTPDiets{dietID: dietID, ownerID: userID}, &optimizationHTTPEntitlements{allowed: true}, newOptimizationHTTPIdempotencyStore(), &optimizationHTTPAdmission{}).
		WithTelemetry(observability.NewOptimizationTelemetry(sink, sink, 2))
	app, cookies, csrf := optimizationHTTPTestApp(t, controller, userID)
	body := optimizationHTTPBody(dietID, 20)

	responses := make([]*optimizationHTTPEnvelopeResponse, submissions)
	started := time.Now()
	runTask234Concurrently(submissions, func(index int) {
		responses[index] = optimizationHTTPSubmit(t, app, body, cookies, csrf, fmt.Sprintf("task-234-unrelated-%02d", index))
	})
	assertTask234CapacityWindow(t, "unrelated submission", started)
	jobIDs := make([]uuid.UUID, submissions)
	for index, response := range responses {
		if response.StatusCode != fiber.StatusAccepted {
			t.Fatalf("unrelated submission %d status = %d, want 202", index, response.StatusCode)
		}
		jobIDs[index] = optimizationHTTPJobID(t, response)
	}

	started = time.Now()
	runTask234Concurrently(submissions, func(index int) {
		replay := optimizationHTTPSubmit(t, app, body, cookies, csrf, fmt.Sprintf("task-234-unrelated-%02d", index))
		if replay.StatusCode != fiber.StatusAccepted || optimizationHTTPJobID(t, replay) != jobIDs[index] {
			t.Errorf("same-key replay %d = status %d job %v, want original acknowledgement", index, replay.StatusCode, replay.Data["jobId"])
		}
	})
	assertTask234CapacityWindow(t, "same-key replay", started)
	if queue.calls != submissions {
		t.Fatalf("queue publications = %d, want %d unrelated jobs and no replay publication", queue.calls, submissions)
	}

	workerRelease := make(chan struct{})
	workerStarted := make(chan struct{})
	go func() {
		for _, jobID := range jobIDs {
			job, _ := store.Load(context.Background(), jobID)
			job.Status = worker.OptimizationJobProcessing
			store.setJob(job)
		}
		close(workerStarted)
		<-workerRelease
	}()
	<-workerStarted
	started = time.Now()
	runTask234Concurrently(submissions, func(index int) {
		poll := optimizationHTTPPoll(t, app, jobIDs[index], cookies)
		if poll.StatusCode != fiber.StatusOK || poll.Data["status"] != string(worker.OptimizationJobProcessing) {
			t.Errorf("poll %d = status %d data %+v, want responsive processing state", index, poll.StatusCode, poll.Data)
		}
	})
	close(workerRelease)
	assertTask234CapacityWindow(t, "poll", started)

	cleanupSink := &observability.MemorySink{}
	cleanupAdmission := &task223Admission{releaseErr: errors.New("private admission release diagnostic")}
	cleanupController := NewOptimizationController(newOptimizationHTTPJobStore(), &optimizationHTTPQueue{err: errors.New("private queue diagnostic")}, &optimizationHTTPDiets{dietID: dietID, ownerID: userID}, &optimizationHTTPEntitlements{allowed: true}, newOptimizationHTTPIdempotencyStore(), cleanupAdmission).
		WithTelemetry(observability.NewOptimizationTelemetry(cleanupSink, cleanupSink, 1))
	cleanupApp, cleanupCookies, cleanupCSRF := optimizationHTTPTestApp(t, cleanupController, userID)
	cleanupStarted := time.Now()
	cleanup := optimizationHTTPSubmit(t, cleanupApp, body, cleanupCookies, cleanupCSRF, "task-234-private-key")
	if cleanup.StatusCode != fiber.StatusServiceUnavailable || cleanup.Error == nil || cleanup.Error.Code != "queue_unavailable" {
		t.Fatalf("release failure replaced final queue response: status=%d error=%+v", cleanup.StatusCode, cleanup.Error)
	}
	if elapsed := time.Since(cleanupStarted); elapsed >= 500*time.Millisecond {
		t.Fatalf("release failure held endpoint for %s", elapsed)
	}
	waitForTask223CleanupTelemetry(t, cleanupSink)
	assertTask223TelemetryIsBounded(t, cleanupSink, userID, dietID, "task-234-private-key", body)

	metrics, _ := sink.Snapshot()
	outcomes := map[string]int{}
	for _, point := range metrics {
		if point.Name == observability.MetricOptimizationSubmissionTotal {
			outcomes[point.Labels["outcome"]]++
		}
	}
	if outcomes[string(observability.OptimizationSubmissionAccepted)] != submissions || outcomes[string(observability.OptimizationSubmissionReplayed)] != submissions || len(outcomes) != 2 {
		t.Fatalf("submission outcomes = %#v, want %d accepted and %d replayed only", outcomes, submissions, submissions)
	}
}

func assertTask234CapacityWindow(t *testing.T, endpoint string, started time.Time) {
	t.Helper()
	if elapsed := time.Since(started); elapsed >= 2*time.Second {
		t.Fatalf("%s endpoints took %s while workers ran, want P95 critical boundary below 2s", endpoint, elapsed)
	}
}

func runTask234Concurrently(count int, run func(int)) {
	var group sync.WaitGroup
	for index := range count {
		group.Add(1)
		go func() {
			defer group.Done()
			run(index)
		}()
	}
	group.Wait()
}
