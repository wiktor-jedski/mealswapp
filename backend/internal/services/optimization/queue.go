package optimization

import (
	"context"
	"strconv"
	"sync"
	"time"

	"mealswapp/backend/internal/http/apperrors"

	"github.com/google/uuid"
)

type JobStatus string

const (
	JobStatusQueued     JobStatus = "queued"
	JobStatusProcessing JobStatus = "processing"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
	JobStatusCancelled  JobStatus = "cancelled"
)

type MacroTarget struct {
	Protein float64 `json:"protein"`
	Carbs   float64 `json:"carbs"`
	Fat     float64 `json:"fat"`
}

type MealInput struct {
	ID       string      `json:"id"`
	Name     string      `json:"name"`
	Quantity float64     `json:"quantity"`
	Macros   MacroTarget `json:"macros"`
	Calories float64     `json:"calories"`
}

type DietOptimizationRequest struct {
	OriginalMeals    []MealInput `json:"originalMeals"`
	TargetMacros     MacroTarget `json:"targetMacros"`
	ExcludedIDs      []string    `json:"excludedIds"`
	TolerancePercent float64     `json:"tolerancePercent"`
}

type OptimizationJob struct {
	JobID      uuid.UUID               `json:"jobId"`
	UserID     uuid.UUID               `json:"userId"`
	Request    DietOptimizationRequest `json:"request"`
	Status     JobStatus               `json:"status"`
	CreatedAt  time.Time               `json:"createdAt"`
	StartedAt  *time.Time              `json:"startedAt,omitempty"`
	FinishedAt *time.Time              `json:"finishedAt,omitempty"`
	Error      string                  `json:"error,omitempty"`
	Result     []map[string]any        `json:"result,omitempty"`
	Metadata   map[string]any          `json:"metadata,omitempty"`
}

type SubmitResult struct {
	JobID   uuid.UUID `json:"jobId"`
	PollURL string    `json:"pollUrl"`
	Status  JobStatus `json:"status"`
}

type QueueStore interface {
	Enqueue(ctx context.Context, job OptimizationJob) error
	Reserve(ctx context.Context) (OptimizationJob, bool, error)
	Get(ctx context.Context, jobID uuid.UUID) (OptimizationJob, bool, error)
	Update(ctx context.Context, job OptimizationJob) error
}

type QueueManager struct {
	store QueueStore
	now   func() time.Time
	newID func() uuid.UUID
}

func NewQueueManager(store QueueStore) QueueManager {
	return NewQueueManagerWithClock(store, time.Now, uuid.New)
}

func NewQueueManagerWithClock(store QueueStore, now func() time.Time, newID func() uuid.UUID) QueueManager {
	if store == nil {
		store = NewMemoryQueueStore()
	}
	return QueueManager{store: store, now: now, newID: newID}
}

func (manager QueueManager) Submit(ctx context.Context, userID uuid.UUID, request DietOptimizationRequest) (SubmitResult, error) {
	if err := ValidateRequest(request); err != nil {
		return SubmitResult{}, err
	}
	jobID := manager.newID()
	job := OptimizationJob{
		JobID:     jobID,
		UserID:    userID,
		Request:   request,
		Status:    JobStatusQueued,
		CreatedAt: manager.now().UTC(),
	}
	if err := manager.store.Enqueue(ctx, job); err != nil {
		return SubmitResult{}, apperrors.DependencyUnavailable("Optimization queue unavailable")
	}
	return SubmitResult{JobID: jobID, PollURL: "/api/v1/optimization/jobs/" + jobID.String(), Status: JobStatusQueued}, nil
}

func (manager QueueManager) Reserve(ctx context.Context) (OptimizationJob, bool, error) {
	job, ok, err := manager.store.Reserve(ctx)
	if err != nil || !ok {
		return job, ok, err
	}
	startedAt := manager.now().UTC()
	job.Status = JobStatusProcessing
	job.StartedAt = &startedAt
	if err := manager.store.Update(ctx, job); err != nil {
		return OptimizationJob{}, false, err
	}
	return job, true, nil
}

func (manager QueueManager) Get(ctx context.Context, jobID uuid.UUID) (OptimizationJob, bool, error) {
	return manager.store.Get(ctx, jobID)
}

func ValidateRequest(request DietOptimizationRequest) error {
	var fields []map[string]string
	if len(request.OriginalMeals) == 0 {
		fields = append(fields, map[string]string{"field": "originalMeals", "code": "required"})
	}
	if request.TargetMacros.Protein <= 0 {
		fields = append(fields, map[string]string{"field": "targetMacros.protein", "code": "positive"})
	}
	if request.TargetMacros.Carbs <= 0 {
		fields = append(fields, map[string]string{"field": "targetMacros.carbs", "code": "positive"})
	}
	if request.TargetMacros.Fat <= 0 {
		fields = append(fields, map[string]string{"field": "targetMacros.fat", "code": "positive"})
	}
	if request.TolerancePercent <= 0 || request.TolerancePercent > 100 {
		fields = append(fields, map[string]string{"field": "tolerancePercent", "code": "range"})
	}
	for i, meal := range request.OriginalMeals {
		if meal.ID == "" {
			fields = append(fields, map[string]string{"field": indexedField("originalMeals", i, "id"), "code": "required"})
		}
		if meal.Quantity <= 0 {
			fields = append(fields, map[string]string{"field": indexedField("originalMeals", i, "quantity"), "code": "positive"})
		}
	}
	if len(fields) > 0 {
		return apperrors.Validation("Optimization request validation failed", fields)
	}
	return nil
}

func indexedField(prefix string, index int, field string) string {
	return prefix + "." + strconv.Itoa(index) + "." + field
}

type MemoryQueueStore struct {
	mu    sync.Mutex
	order []uuid.UUID
	jobs  map[uuid.UUID]OptimizationJob
}

func NewMemoryQueueStore() *MemoryQueueStore {
	return &MemoryQueueStore{jobs: map[uuid.UUID]OptimizationJob{}}
}

func (store *MemoryQueueStore) Enqueue(ctx context.Context, job OptimizationJob) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	store.jobs[job.JobID] = job
	store.order = append(store.order, job.JobID)
	return nil
}

func (store *MemoryQueueStore) Reserve(ctx context.Context) (OptimizationJob, bool, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	for len(store.order) > 0 {
		jobID := store.order[0]
		store.order = store.order[1:]
		job, ok := store.jobs[jobID]
		if ok && job.Status == JobStatusQueued {
			return job, true, nil
		}
	}
	return OptimizationJob{}, false, nil
}

func (store *MemoryQueueStore) Get(ctx context.Context, jobID uuid.UUID) (OptimizationJob, bool, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	job, ok := store.jobs[jobID]
	return job, ok, nil
}

func (store *MemoryQueueStore) Update(ctx context.Context, job OptimizationJob) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	store.jobs[job.JobID] = job
	return nil
}
