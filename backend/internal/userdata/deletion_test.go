package userdata

// Implements DESIGN-008 AccountDeleter verification.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

type memoryDeletionRepository struct {
	request     repository.DataDeletionRequest
	claimed     []repository.DataDeletionRequest
	failures    []string
	nextRetry   *time.Time
	completed   uuid.UUID
	requestErr  error
	claimErr    error
	failureErr  error
	completeErr error
}

func (r *memoryDeletionRepository) RequestDeletion(context.Context, uuid.UUID) (repository.DataDeletionRequest, error) {
	return r.request, r.requestErr
}
func (r *memoryDeletionRepository) UpdateDeletionStatus(context.Context, uuid.UUID, string, string) error {
	return nil
}
func (r *memoryDeletionRepository) ListDeletionAudit(context.Context, uuid.UUID) ([]repository.DataDeletionAuditEntry, error) {
	return nil, nil
}
func (r *memoryDeletionRepository) ClaimDeletionRequests(context.Context, time.Time, int) ([]repository.DataDeletionRequest, error) {
	return r.claimed, r.claimErr
}
func (r *memoryDeletionRepository) RecordDeletionFailure(_ context.Context, _ uuid.UUID, _ time.Time, category string, note string, nextAttemptAt *time.Time) error {
	r.failures = append(r.failures, category+":"+note)
	r.nextRetry = nextAttemptAt
	return r.failureErr
}
func (r *memoryDeletionRepository) CompleteDeletionRequest(_ context.Context, _ uuid.UUID, _ time.Time, receiptID uuid.UUID, _ time.Time) error {
	r.completed = receiptID
	return r.completeErr
}

type memoryDeletionSessions struct {
	revoked uuid.UUID
	err     error
}

func (s *memoryDeletionSessions) CreateSession(context.Context, repository.UserSession) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (s *memoryDeletionSessions) GetSessionByRefreshTokenHash(context.Context, string) (repository.UserSession, error) {
	return repository.UserSession{}, nil
}
func (s *memoryDeletionSessions) RevokeSession(context.Context, uuid.UUID) error       { return nil }
func (s *memoryDeletionSessions) RevokeSessionFamily(context.Context, uuid.UUID) error { return nil }
func (s *memoryDeletionSessions) RevokeUserSessions(_ context.Context, userID uuid.UUID) error {
	s.revoked = userID
	return s.err
}

type memoryAccountDeletionRepository struct {
	deleted uuid.UUID
	err     error
}

func (r *memoryAccountDeletionRepository) DeleteUserAccount(_ context.Context, userID uuid.UUID) error {
	if r.err != nil {
		return r.err
	}
	r.deleted = userID
	return nil
}

type memoryCachePurger struct {
	err error
}

func (p memoryCachePurger) PurgeUser(context.Context, uuid.UUID) error { return p.err }

type memoryCanceledCachePurger struct{}

func (memoryCanceledCachePurger) PurgeUser(context.Context, uuid.UUID) error {
	return context.Canceled
}

// TestAccountDeletionService verifies DESIGN-008 AccountDeleter service behavior.
func TestAccountDeletionService(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	requestID := uuid.New()
	receiptID := uuid.New()
	requests := &memoryDeletionRepository{request: repository.DataDeletionRequest{ID: requestID, UserID: userID, Status: "pending", RequestedAt: time.Now()}}
	sessions := &memoryDeletionSessions{}
	accounts := &memoryAccountDeletionRepository{}
	service := NewAccountDeletionService(requests, sessions, accounts, nil)
	request, err := service.RequestDeletion(ctx, userID)
	if err != nil || request.ID != requestID || sessions.revoked != userID {
		t.Fatalf("RequestDeletion() request=%#v err=%v revoked=%s", request, err, sessions.revoked)
	}
	leaseExpiresAt := time.Now().Add(time.Minute)
	request.NextAttemptAt = &leaseExpiresAt
	if err := service.ExecuteDeletion(ctx, request, receiptID, time.Now()); err != nil {
		t.Fatalf("ExecuteDeletion() error = %v", err)
	}
	if accounts.deleted != userID || requests.completed != receiptID {
		t.Fatalf("execution deleted=%s completed=%s", accounts.deleted, requests.completed)
	}
	missingLease := request
	missingLease.NextAttemptAt = nil
	if err := service.ExecuteDeletion(ctx, missingLease, uuid.New(), time.Now()); !repository.IsKind(err, repository.ErrorKindValidation) {
		t.Fatalf("ExecuteDeletion() missing lease error = %v", err)
	}
	expiredLease := time.Now().Add(-time.Second)
	expired := request
	expired.NextAttemptAt = &expiredLease
	if err := service.ExecuteDeletion(ctx, expired, uuid.New(), time.Now()); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("ExecuteDeletion() expired lease error = %v", err)
	}
	failingCache := NewAccountDeletionService(requests, sessions, accounts, memoryCachePurger{err: errors.New("cache down")})
	if err := failingCache.ExecuteDeletion(ctx, request, uuid.New(), time.Now()); !repository.IsKind(err, repository.ErrorKindRetryable) {
		t.Fatalf("ExecuteDeletion() cache failure path error = %v, want retryable", err)
	}
	canceledCache := NewAccountDeletionService(requests, sessions, accounts, memoryCanceledCachePurger{})
	if err := canceledCache.ExecuteDeletion(ctx, request, uuid.New(), time.Now()); !errors.Is(err, context.Canceled) {
		t.Fatalf("ExecuteDeletion() canceled cache error = %v", err)
	}
}

// TestAccountDeletionServiceProcessesClaimedWork verifies DESIGN-015 retry orchestration.
func TestAccountDeletionServiceProcessesClaimedWork(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 7, 12, 0, 0, 0, time.UTC)
	leaseExpiresAt := time.Now().Add(time.Minute)
	userID := uuid.New()
	request := repository.DataDeletionRequest{ID: uuid.New(), UserID: userID, Status: "processing", RetryCount: 1, NextAttemptAt: &leaseExpiresAt}
	requests := &memoryDeletionRepository{claimed: []repository.DataDeletionRequest{request}}
	accounts := &memoryAccountDeletionRepository{err: repository.NewError(repository.ErrorKindRetryable, "temporary", nil)}
	service := NewAccountDeletionService(requests, &memoryDeletionSessions{}, accounts, nil)
	claimed, err := service.ProcessDueDeletionRequests(ctx, now, 10)
	if err != nil || len(claimed) != 1 {
		t.Fatalf("ProcessDueDeletionRequests() claimed=%#v err=%v", claimed, err)
	}
	if len(requests.failures) != 1 || requests.failures[0] != "transient:dependency_unavailable" {
		t.Fatalf("failure metadata = %#v", requests.failures)
	}
	if requests.nextRetry == nil || !requests.nextRetry.Equal(now.Add(2*time.Minute)) {
		t.Fatalf("next retry = %v", requests.nextRetry)
	}

	requests.claimed = []repository.DataDeletionRequest{{ID: uuid.New(), Status: "processing", NextAttemptAt: &leaseExpiresAt}}
	requests.failures = nil
	requests.nextRetry = nil
	accounts.err = nil
	if _, err := service.ProcessDueDeletionRequests(ctx, now, 1); err != nil {
		t.Fatalf("missing user processing error = %v", err)
	}
	if len(requests.failures) != 1 || requests.failures[0] != "permanent:missing_user_id" || requests.nextRetry != nil {
		t.Fatalf("missing user metadata = %#v retry=%v", requests.failures, requests.nextRetry)
	}
}

func TestAccountDeletionServicePropagatesFailuresAndClassifiesRetries(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC)
	leaseExpiresAt := time.Now().Add(time.Minute)
	userID := uuid.New()
	request := repository.DataDeletionRequest{ID: uuid.New(), UserID: userID, RetryCount: 2, NextAttemptAt: &leaseExpiresAt}
	wantErr := errors.New("failed")

	service := NewAccountDeletionService(&memoryDeletionRepository{requestErr: wantErr}, &memoryDeletionSessions{}, &memoryAccountDeletionRepository{}, nil)
	if _, err := service.RequestDeletion(ctx, userID); !errors.Is(err, wantErr) {
		t.Fatalf("request error = %v", err)
	}
	service = NewAccountDeletionService(&memoryDeletionRepository{request: request}, &memoryDeletionSessions{err: wantErr}, &memoryAccountDeletionRepository{}, nil)
	if _, err := service.RequestDeletion(ctx, userID); !errors.Is(err, wantErr) {
		t.Fatalf("session revoke request error = %v", err)
	}
	if err := service.ExecuteDeletion(ctx, request, uuid.New(), now); !errors.Is(err, wantErr) {
		t.Fatalf("session revoke execution error = %v", err)
	}

	requests := &memoryDeletionRepository{claimErr: wantErr}
	service = NewAccountDeletionService(requests, &memoryDeletionSessions{}, &memoryAccountDeletionRepository{}, nil)
	if _, err := service.ProcessDueDeletionRequests(ctx, time.Time{}, 1); !errors.Is(err, wantErr) {
		t.Fatalf("claim error = %v", err)
	}
	requests = &memoryDeletionRepository{claimed: []repository.DataDeletionRequest{{ID: uuid.New()}}}
	service = NewAccountDeletionService(requests, &memoryDeletionSessions{}, &memoryAccountDeletionRepository{}, nil)
	if _, err := service.ProcessDueDeletionRequests(ctx, now, 1); !repository.IsKind(err, repository.ErrorKindValidation) {
		t.Fatalf("missing processing lease error = %v", err)
	}

	requests = &memoryDeletionRepository{claimed: []repository.DataDeletionRequest{{ID: uuid.New(), NextAttemptAt: &leaseExpiresAt}}, failureErr: wantErr}
	service = NewAccountDeletionService(requests, &memoryDeletionSessions{}, &memoryAccountDeletionRepository{}, nil)
	if _, err := service.ProcessDueDeletionRequests(ctx, now, 1); !errors.Is(err, wantErr) {
		t.Fatalf("missing-user record error = %v", err)
	}
	requests = &memoryDeletionRepository{claimed: []repository.DataDeletionRequest{request}, failureErr: wantErr}
	service = NewAccountDeletionService(requests, &memoryDeletionSessions{}, &memoryAccountDeletionRepository{err: wantErr}, nil)
	if _, err := service.ProcessDueDeletionRequests(ctx, now, 1); !errors.Is(err, wantErr) {
		t.Fatalf("execution failure record error = %v", err)
	}

	for _, tc := range []struct {
		err      error
		category string
		note     string
	}{
		{context.Canceled, "transient", "deadline_or_cancellation"},
		{repository.NewError(repository.ErrorKindValidation, "bad", nil), "permanent", "invalid_deletion_state"},
		{repository.NewError(repository.ErrorKindInternal, "bad", nil), "unknown", "repository_failure"},
		{wantErr, "unknown", "deletion_failed"},
	} {
		category, note := classifyDeletionFailure(tc.err)
		if category != tc.category || note != tc.note {
			t.Fatalf("classifyDeletionFailure(%v) = %s:%s", tc.err, category, note)
		}
	}
	if nextDeletionAttempt(now, 2, "transient") != nil || nextDeletionAttempt(now, 0, "permanent") != nil {
		t.Fatal("non-retryable deletion scheduled")
	}
}
