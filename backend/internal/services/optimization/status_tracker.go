package optimization

import (
	"context"
	"time"

	"github.com/google/uuid"
)

const (
	DefaultJobResultTTL = time.Hour
	DefaultJobTimeout   = 30 * time.Second
)

type StatusStore interface {
	Get(ctx context.Context, jobID uuid.UUID) (OptimizationJob, bool, error)
	Update(ctx context.Context, job OptimizationJob) error
}

type JobStatusView struct {
	JobID      uuid.UUID         `json:"jobId"`
	Status     JobStatus         `json:"status"`
	Progress   int               `json:"progress"`
	Result     []DietAlternative `json:"result,omitempty"`
	Error      string            `json:"error,omitempty"`
	CreatedAt  time.Time         `json:"createdAt"`
	StartedAt  *time.Time        `json:"startedAt,omitempty"`
	FinishedAt *time.Time        `json:"finishedAt,omitempty"`
}

type StatusTracker struct {
	store     StatusStore
	now       func() time.Time
	resultTTL time.Duration
	timeout   time.Duration
}

func NewStatusTracker(store StatusStore) StatusTracker {
	return NewStatusTrackerWithClock(store, time.Now)
}

func NewStatusTrackerWithClock(store StatusStore, now func() time.Time) StatusTracker {
	return StatusTracker{store: store, now: now, resultTTL: DefaultJobResultTTL, timeout: DefaultJobTimeout}
}

func (tracker StatusTracker) MarkProcessing(ctx context.Context, jobID uuid.UUID) (OptimizationJob, error) {
	job, ok, err := tracker.store.Get(ctx, jobID)
	if err != nil || !ok {
		return job, err
	}
	startedAt := tracker.now().UTC()
	job.Status = JobStatusProcessing
	job.StartedAt = &startedAt
	job.Metadata = setProgress(job.Metadata, 25)
	return job, tracker.store.Update(ctx, job)
}

func (tracker StatusTracker) MarkCompleted(ctx context.Context, jobID uuid.UUID, alternatives []DietAlternative) (OptimizationJob, error) {
	job, ok, err := tracker.store.Get(ctx, jobID)
	if err != nil || !ok {
		return job, err
	}
	finishedAt := tracker.now().UTC()
	job.Status = JobStatusCompleted
	job.FinishedAt = &finishedAt
	job.Error = ""
	job.Result = alternativesToMaps(alternatives)
	job.Metadata = setProgress(job.Metadata, 100)
	job.Metadata["resultExpiresAt"] = finishedAt.Add(tracker.resultTTL).Format(time.RFC3339)
	return job, tracker.store.Update(ctx, job)
}

func (tracker StatusTracker) MarkFailed(ctx context.Context, jobID uuid.UUID, message string, partial []DietAlternative) (OptimizationJob, error) {
	job, ok, err := tracker.store.Get(ctx, jobID)
	if err != nil || !ok {
		return job, err
	}
	finishedAt := tracker.now().UTC()
	job.Status = JobStatusFailed
	job.FinishedAt = &finishedAt
	job.Error = message
	job.Result = alternativesToMaps(partial)
	job.Metadata = setProgress(job.Metadata, 100)
	return job, tracker.store.Update(ctx, job)
}

func (tracker StatusTracker) ApplyTimeout(ctx context.Context, jobID uuid.UUID, partial []DietAlternative) (bool, error) {
	job, ok, err := tracker.store.Get(ctx, jobID)
	if err != nil || !ok {
		return false, err
	}
	if job.Status != JobStatusProcessing || job.StartedAt == nil || tracker.now().Sub(*job.StartedAt) < tracker.timeout {
		return false, nil
	}
	_, err = tracker.MarkFailed(ctx, jobID, "Optimization timed out", partial)
	return true, err
}

func (tracker StatusTracker) View(ctx context.Context, jobID uuid.UUID) (JobStatusView, bool, error) {
	job, ok, err := tracker.store.Get(ctx, jobID)
	if err != nil || !ok {
		return JobStatusView{}, ok, err
	}
	view := JobStatusView{
		JobID:      job.JobID,
		Status:     job.Status,
		Progress:   progress(job),
		Error:      job.Error,
		CreatedAt:  job.CreatedAt,
		StartedAt:  job.StartedAt,
		FinishedAt: job.FinishedAt,
	}
	if job.Status == JobStatusCompleted && !tracker.resultExpired(job) {
		view.Result = mapsToAlternatives(job.Result)
	}
	if job.Status == JobStatusFailed {
		view.Result = mapsToAlternatives(job.Result)
	}
	return view, true, nil
}

func (tracker StatusTracker) resultExpired(job OptimizationJob) bool {
	if job.FinishedAt == nil {
		return false
	}
	return !tracker.now().Before(job.FinishedAt.Add(tracker.resultTTL))
}

func setProgress(metadata map[string]any, value int) map[string]any {
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["progress"] = value
	return metadata
}

func progress(job OptimizationJob) int {
	if job.Metadata != nil {
		if value, ok := job.Metadata["progress"].(int); ok {
			return value
		}
	}
	switch job.Status {
	case JobStatusQueued:
		return 0
	case JobStatusProcessing:
		return 50
	case JobStatusCompleted, JobStatusFailed, JobStatusCancelled:
		return 100
	default:
		return 0
	}
}

func alternativesToMaps(alternatives []DietAlternative) []map[string]any {
	if alternatives == nil {
		return nil
	}
	items := make([]map[string]any, len(alternatives))
	for i, alternative := range alternatives {
		items[i] = map[string]any{
			"meals":           alternative.Meals,
			"macros":          alternative.Macros,
			"calories":        alternative.Calories,
			"similarityScore": alternative.SimilarityScore,
		}
	}
	return items
}

func mapsToAlternatives(items []map[string]any) []DietAlternative {
	alternatives := make([]DietAlternative, 0, len(items))
	for _, item := range items {
		alternative := DietAlternative{}
		if calories, ok := item["calories"].(float64); ok {
			alternative.Calories = calories
		}
		if score, ok := item["similarityScore"].(float64); ok {
			alternative.SimilarityScore = score
		}
		if macros, ok := item["macros"].(MacroTarget); ok {
			alternative.Macros = macros
		}
		if meals, ok := item["meals"].([]MealQuantity); ok {
			alternative.Meals = meals
		}
		alternatives = append(alternatives, alternative)
	}
	return alternatives
}
