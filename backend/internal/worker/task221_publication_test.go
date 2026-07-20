package worker

// Implements DESIGN-004 JobStatusTracker Task 221 authoritative result persistence verification.

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/optimization"
)

func TestTask221RedisStoreValidatesSimilarityScoreAtPublicationAndDecode(t *testing.T) {
	client := openWorkerIntegrationRedis(t)
	ctx := context.Background()
	invalidScores := []float64{-0.0001, 1.0001, 0.12345}

	for _, score := range invalidScores {
		jobID := uuid.New()
		store := NewRedisOptimizationJobStore(client)
		if err := store.Save(ctx, OptimizationJob{JobID: jobID, UserID: uuid.New(), DailyDietID: uuid.New(), Status: OptimizationJobQueued}); err != nil {
			t.Fatalf("Save() error = %v", err)
		}
		if _, err := store.MarkProcessing(ctx, jobID, time.Now()); err != nil {
			t.Fatalf("MarkProcessing() error = %v", err)
		}
		if err := store.PublishCompleted(ctx, jobID, []optimization.DietAlternative{task221Alternative(score)}, time.Now()); err == nil {
			t.Fatalf("PublishCompleted(similarityScore=%v) succeeded", score)
		}

		job := OptimizationJob{
			JobID: jobID, UserID: uuid.New(), DailyDietID: uuid.New(), Status: OptimizationJobCompleted,
			CreatedAt: time.Now().UTC(), Alternatives: []optimization.DietAlternative{task221Alternative(score)},
		}
		payload, err := json.Marshal(job)
		if err != nil {
			t.Fatalf("json.Marshal() error = %v", err)
		}
		if err := client.Set(ctx, optimizationJobKey(jobID), payload, time.Minute).Err(); err != nil {
			t.Fatalf("inject malformed completed job: %v", err)
		}
		if _, err := store.Load(ctx, jobID); err == nil {
			t.Fatalf("Load(similarityScore=%v) succeeded", score)
		}
	}

	jobID := uuid.New()
	store := NewRedisOptimizationJobStore(client)
	if err := store.Save(ctx, OptimizationJob{JobID: jobID, UserID: uuid.New(), DailyDietID: uuid.New(), Status: OptimizationJobQueued}); err != nil {
		t.Fatalf("Save(valid) error = %v", err)
	}
	if _, err := store.MarkProcessing(ctx, jobID, time.Now()); err != nil {
		t.Fatalf("MarkProcessing(valid) error = %v", err)
	}
	if err := store.PublishCompleted(ctx, jobID, []optimization.DietAlternative{task221Alternative(0.1234)}, time.Now()); err != nil {
		t.Fatalf("PublishCompleted(valid) error = %v", err)
	}
	job, err := store.Load(ctx, jobID)
	if err != nil {
		t.Fatalf("Load(valid) error = %v", err)
	}
	if job.Status != OptimizationJobCompleted || len(job.Alternatives) != 1 {
		t.Fatalf("round-trip job = %+v, want one completed alternative", job)
	}
	if got := job.Alternatives[0].SimilarityScore; got != 0.1234 {
		t.Fatalf("round-trip similarityScore = %v, want 0.1234", got)
	}
}

// Implements DESIGN-004 JobStatusTracker presence-aware persisted result decoding.
func TestTask221RedisStoreRejectsMalformedRawSimilarityScore(t *testing.T) {
	client := openWorkerIntegrationRedis(t)
	ctx := context.Background()
	store := NewRedisOptimizationJobStore(client)
	tests := []struct {
		name      string
		score     any
		omitScore bool
		wantErr   bool
	}{
		{name: "omitted", omitScore: true, wantErr: true},
		{name: "null", score: nil, wantErr: true},
		{name: "string", score: "0", wantErr: true},
		{name: "zero", score: float64(0)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jobID := uuid.New()
			payload := task221RawCompletedJob(t, jobID, tt.score, tt.omitScore)
			if err := client.Set(ctx, optimizationJobKey(jobID), payload, time.Minute).Err(); err != nil {
				t.Fatalf("inject raw completed job: %v", err)
			}
			job, err := store.Load(ctx, jobID)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Load() error = %v, wantErr %v; job=%+v", err, tt.wantErr, job)
			}
			if !tt.wantErr && job.Alternatives[0].SimilarityScore != 0 {
				t.Fatalf("Load() similarityScore = %v, want zero", job.Alternatives[0].SimilarityScore)
			}
		})
	}
}

func task221RawCompletedJob(t *testing.T, jobID uuid.UUID, score any, omitScore bool) []byte {
	t.Helper()
	job := OptimizationJob{
		JobID: jobID, UserID: uuid.New(), DailyDietID: uuid.New(), Status: OptimizationJobCompleted,
		CreatedAt: time.Now().UTC(), Alternatives: []optimization.DietAlternative{task221Alternative(0)},
	}
	payload, err := json.Marshal(job)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(payload, &raw); err != nil {
		t.Fatalf("decode raw fixture: %v", err)
	}
	alternative := raw["alternatives"].([]any)[0].(map[string]any)
	if omitScore {
		delete(alternative, "similarityScore")
	} else {
		alternative["similarityScore"] = score
	}
	payload, err = json.Marshal(raw)
	if err != nil {
		t.Fatalf("encode raw fixture: %v", err)
	}
	return payload
}

func task221Alternative(score float64) optimization.DietAlternative {
	return optimization.DietAlternative{
		Meals:  []optimization.MealQuantity{{MealID: uuid.New(), Quantity: 100, Unit: "g", Position: 0}},
		Macros: optimization.MacroTarget{Protein: 20, Carbohydrates: 30, Fat: 10}, Calories: 290, SimilarityScore: score,
	}
}
