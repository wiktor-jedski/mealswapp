package http

import (
	"context"
	"net/http"
	"testing"
	"time"

	"mealswapp/backend/internal/config"
	"mealswapp/backend/internal/services/optimization"

	"github.com/google/uuid"
)

func TestOptimizationControllerReturnsAcceptedJobResponse(t *testing.T) {
	userID := uuid.New()
	jobID := uuid.MustParse("00000000-0000-0000-0000-000000000069")
	service := &fakeOptimizationService{
		submitResult: optimization.SubmitResult{
			JobID:   jobID,
			PollURL: "/api/v1/optimization/jobs/" + jobID.String(),
			Status:  optimization.JobStatusQueued,
		},
	}
	app := NewRouter(ServiceDependencies{
		Config:                   config.Config{Environment: "test"},
		OptimizationService:      service,
		OptimizationUserResolver: fakeOptimizationUserResolver{userID: userID},
	})

	res := performJSONRequest(t, app, http.MethodPost, "/api/v1/optimization/jobs", validOptimizationJSON(), "access-token", false)
	defer res.Body.Close()
	if res.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 accepted, got %d", res.StatusCode)
	}
	if service.lastUserID != userID || service.lastRequest.TargetMacros.Protein != 100 {
		t.Fatalf("unexpected service call: user=%s request=%#v", service.lastUserID, service.lastRequest)
	}
	data := dataMap(t, decodeEnvelope(t, res).Data)
	if data["jobId"] != jobID.String() || data["pollUrl"] != "/api/v1/optimization/jobs/"+jobID.String() || data["status"] != "queued" {
		t.Fatalf("unexpected accepted payload: %#v", data)
	}
}

func TestOptimizationControllerRejectsInvalidPayload(t *testing.T) {
	manager := optimization.NewQueueManagerWithClock(optimization.NewMemoryQueueStore(), fixedQueueNowHTTP, uuid.New)
	app := NewRouter(ServiceDependencies{
		Config:                   config.Config{Environment: "test"},
		OptimizationService:      manager,
		OptimizationUserResolver: fakeOptimizationUserResolver{userID: uuid.New()},
	})

	res := performJSONRequest(t, app, http.MethodPost, "/api/v1/optimization/jobs", `{}`, "access-token", false)
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected invalid payload 400, got %d", res.StatusCode)
	}
	payload := decodeEnvelope(t, res)
	if payload.Error == nil || payload.Error.Code != "validation_error" {
		t.Fatalf("expected validation envelope, got %#v", payload)
	}
}

func TestOptimizationControllerRequiresAuthAndReturnsJobStatus(t *testing.T) {
	jobID := uuid.New()
	service := &fakeOptimizationService{
		job: optimization.OptimizationJob{JobID: jobID, Status: optimization.JobStatusQueued},
		ok:  true,
	}
	app := NewRouter(ServiceDependencies{
		Config:              config.Config{Environment: "test"},
		OptimizationService: service,
	})

	unauthorized := performJSONRequest(t, app, http.MethodPost, "/api/v1/optimization/jobs", validOptimizationJSON(), "", false)
	defer unauthorized.Body.Close()
	if unauthorized.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized submit 401, got %d", unauthorized.StatusCode)
	}

	status := performRequest(t, app, http.MethodGet, "/api/v1/optimization/jobs/"+jobID.String())
	defer status.Body.Close()
	if status.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status.StatusCode)
	}
	data := dataMap(t, decodeEnvelope(t, status).Data)
	if data["jobId"] != jobID.String() || data["status"] != "queued" {
		t.Fatalf("unexpected job status payload: %#v", data)
	}
}

type fakeOptimizationService struct {
	submitResult optimization.SubmitResult
	lastUserID   uuid.UUID
	lastRequest  optimization.DietOptimizationRequest
	job          optimization.OptimizationJob
	ok           bool
}

func (service *fakeOptimizationService) Submit(ctx context.Context, userID uuid.UUID, request optimization.DietOptimizationRequest) (optimization.SubmitResult, error) {
	service.lastUserID = userID
	service.lastRequest = request
	return service.submitResult, nil
}

func (service *fakeOptimizationService) Get(ctx context.Context, jobID uuid.UUID) (optimization.OptimizationJob, bool, error) {
	return service.job, service.ok, nil
}

type fakeOptimizationUserResolver struct {
	userID uuid.UUID
}

func (resolver fakeOptimizationUserResolver) UserIDFromAccessToken(ctx context.Context, accessToken string) (uuid.UUID, bool, error) {
	return resolver.userID, resolver.userID != uuid.Nil, nil
}

func validOptimizationJSON() string {
	return `{
		"originalMeals":[{"id":"meal-1","name":"Breakfast","quantity":1}],
		"targetMacros":{"protein":100,"carbs":150,"fat":60},
		"excludedIds":["meal-9"],
		"tolerancePercent":10
	}`
}

func fixedQueueNowHTTP() time.Time {
	return time.Date(2026, 5, 20, 13, 30, 0, 0, time.UTC)
}
