package optimization

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestStatusTrackerTransitionsQueuedProcessingCompleted(t *testing.T) {
	now := fixedQueueNow()
	store := NewMemoryQueueStore()
	jobID := uuid.New()
	job := OptimizationJob{JobID: jobID, Status: JobStatusQueued, CreatedAt: now}
	if err := store.Enqueue(context.Background(), job); err != nil {
		t.Fatal(err)
	}
	tracker := NewStatusTrackerWithClock(store, func() time.Time { return now })

	processing, err := tracker.MarkProcessing(context.Background(), jobID)
	if err != nil {
		t.Fatal(err)
	}
	if processing.Status != JobStatusProcessing || processing.StartedAt == nil {
		t.Fatalf("expected processing job, got %#v", processing)
	}

	alternative := DietAlternative{Calories: 320, SimilarityScore: 1}
	completed, err := tracker.MarkCompleted(context.Background(), jobID, []DietAlternative{alternative})
	if err != nil {
		t.Fatal(err)
	}
	if completed.Status != JobStatusCompleted || completed.FinishedAt == nil || len(completed.Result) != 1 {
		t.Fatalf("expected completed job with result, got %#v", completed)
	}
	view, ok, err := tracker.View(context.Background(), jobID)
	if err != nil || !ok {
		t.Fatalf("expected status view, ok=%v err=%v", ok, err)
	}
	if view.Progress != 100 || len(view.Result) != 1 || view.Result[0].Calories != 320 {
		t.Fatalf("unexpected completed view: %#v", view)
	}
}

func TestStatusTrackerExpiresCompletedResultAfterTTL(t *testing.T) {
	now := fixedQueueNow()
	store := NewMemoryQueueStore()
	jobID := uuid.New()
	if err := store.Enqueue(context.Background(), OptimizationJob{JobID: jobID, Status: JobStatusQueued, CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	tracker := NewStatusTrackerWithClock(store, func() time.Time { return now })
	if _, err := tracker.MarkCompleted(context.Background(), jobID, []DietAlternative{{Calories: 100}}); err != nil {
		t.Fatal(err)
	}

	now = now.Add(DefaultJobResultTTL + time.Second)
	view, ok, err := tracker.View(context.Background(), jobID)
	if err != nil || !ok {
		t.Fatalf("expected expired status view, ok=%v err=%v", ok, err)
	}
	if len(view.Result) != 0 || view.Status != JobStatusCompleted {
		t.Fatalf("expected completed status with expired result omitted, got %#v", view)
	}
}

func TestStatusTrackerMarksTimeoutWithPartialResults(t *testing.T) {
	now := fixedQueueNow()
	startedAt := now.Add(-DefaultJobTimeout - time.Second)
	store := NewMemoryQueueStore()
	jobID := uuid.New()
	if err := store.Enqueue(context.Background(), OptimizationJob{
		JobID:     jobID,
		Status:    JobStatusProcessing,
		CreatedAt: now.Add(-time.Minute),
		StartedAt: &startedAt,
	}); err != nil {
		t.Fatal(err)
	}
	tracker := NewStatusTrackerWithClock(store, func() time.Time { return now })

	timedOut, err := tracker.ApplyTimeout(context.Background(), jobID, []DietAlternative{{Calories: 250}})
	if err != nil {
		t.Fatal(err)
	}
	if !timedOut {
		t.Fatal("expected timeout to be applied")
	}
	view, ok, err := tracker.View(context.Background(), jobID)
	if err != nil || !ok {
		t.Fatalf("expected timeout view, ok=%v err=%v", ok, err)
	}
	if view.Status != JobStatusFailed || view.Error != "Optimization timed out" || len(view.Result) != 1 {
		t.Fatalf("unexpected timeout view: %#v", view)
	}
}

func TestStatusTrackerDoesNotTimeoutRecentProcessingJob(t *testing.T) {
	now := fixedQueueNow()
	startedAt := now.Add(-time.Second)
	store := NewMemoryQueueStore()
	jobID := uuid.New()
	if err := store.Enqueue(context.Background(), OptimizationJob{JobID: jobID, Status: JobStatusProcessing, CreatedAt: now, StartedAt: &startedAt}); err != nil {
		t.Fatal(err)
	}
	tracker := NewStatusTrackerWithClock(store, func() time.Time { return now })

	timedOut, err := tracker.ApplyTimeout(context.Background(), jobID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if timedOut {
		t.Fatal("did not expect recent processing job to timeout")
	}
}
