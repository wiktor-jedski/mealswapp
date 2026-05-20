package optimization

import (
	"context"
	"testing"
	"time"

	"mealswapp/backend/internal/http/apperrors"

	"github.com/google/uuid"
)

func TestQueueManagerSubmitsQueuedJobWithPollURL(t *testing.T) {
	jobID := uuid.MustParse("00000000-0000-0000-0000-000000000069")
	userID := uuid.New()
	store := NewMemoryQueueStore()
	manager := NewQueueManagerWithClock(store, fixedQueueNow, func() uuid.UUID { return jobID })

	result, err := manager.Submit(context.Background(), userID, validRequest())
	if err != nil {
		t.Fatal(err)
	}
	if result.JobID != jobID || result.Status != JobStatusQueued || result.PollURL != "/api/v1/optimization/jobs/"+jobID.String() {
		t.Fatalf("unexpected submit result: %#v", result)
	}
	job, ok, err := store.Get(context.Background(), jobID)
	if err != nil || !ok {
		t.Fatalf("expected stored job, ok=%v err=%v", ok, err)
	}
	if job.UserID != userID || job.Status != JobStatusQueued || !job.CreatedAt.Equal(fixedQueueNow()) {
		t.Fatalf("unexpected stored job: %#v", job)
	}
}

func TestQueueManagerRejectsInvalidPayloadBeforeEnqueue(t *testing.T) {
	manager := NewQueueManagerWithClock(NewMemoryQueueStore(), fixedQueueNow, uuid.New)

	_, err := manager.Submit(context.Background(), uuid.New(), DietOptimizationRequest{})
	appErr, ok := apperrors.As(err)
	if !ok || appErr.Code != "validation_error" {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestQueueManagerWorkerReservesQueuedJobAsProcessing(t *testing.T) {
	jobID := uuid.New()
	store := NewMemoryQueueStore()
	manager := NewQueueManagerWithClock(store, fixedQueueNow, func() uuid.UUID { return jobID })
	if _, err := manager.Submit(context.Background(), uuid.New(), validRequest()); err != nil {
		t.Fatal(err)
	}

	job, ok, err := manager.Reserve(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !ok || job.JobID != jobID || job.Status != JobStatusProcessing || job.StartedAt == nil || !job.StartedAt.Equal(fixedQueueNow()) {
		t.Fatalf("expected processing reservation, ok=%v job=%#v", ok, job)
	}
	stored, ok, err := store.Get(context.Background(), jobID)
	if err != nil || !ok || stored.Status != JobStatusProcessing {
		t.Fatalf("expected stored processing job, ok=%v job=%#v err=%v", ok, stored, err)
	}
}

func validRequest() DietOptimizationRequest {
	return DietOptimizationRequest{
		OriginalMeals:    []MealInput{{ID: "meal-1", Name: "Breakfast", Quantity: 1}},
		TargetMacros:     MacroTarget{Protein: 100, Carbs: 150, Fat: 60},
		ExcludedIDs:      []string{"meal-9"},
		TolerancePercent: 10,
	}
}

func fixedQueueNow() time.Time {
	return time.Date(2026, 5, 20, 13, 0, 0, 0, time.UTC)
}
