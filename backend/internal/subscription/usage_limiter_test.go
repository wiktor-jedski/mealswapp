package subscription

import (
	"context"
	"sync"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

type mockUsageRepo struct {
	mu                sync.Mutex
	searchCount       int
	getUsageSinceFunc func(ctx context.Context, userID uuid.UUID, feature string, since time.Time) (repository.UsageWindow, error)
	recordUsageFunc   func(ctx context.Context, userID uuid.UUID, feature string, occurredAt time.Time) (repository.UsageWindow, error)
}

func (m *mockUsageRepo) GetUsageSince(ctx context.Context, userID uuid.UUID, feature string, since time.Time) (repository.UsageWindow, error) {
	if m.getUsageSinceFunc != nil {
		return m.getUsageSinceFunc(ctx, userID, feature, since)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return repository.UsageWindow{SearchCount: m.searchCount}, nil
}

func (m *mockUsageRepo) RecordUsage(ctx context.Context, userID uuid.UUID, feature string, occurredAt time.Time) (repository.UsageWindow, error) {
	if m.recordUsageFunc != nil {
		return m.recordUsageFunc(ctx, userID, feature, occurredAt)
	}
	m.mu.Lock()
	m.searchCount++
	m.mu.Unlock()
	return repository.UsageWindow{}, nil
}

func TestUsageLimiter_CheckAccess(t *testing.T) {
	userID := uuid.New()
	now := time.Now()

	freeEntitlement := &repository.Entitlement{UserID: userID, Tier: "free", Status: "active"}
	paidEntitlement := &repository.Entitlement{UserID: userID, Tier: "paid", Status: "active"}
	trialEntitlement := &repository.Entitlement{UserID: userID, Tier: "trial", Status: "active"}

	tests := []struct {
		name        string
		ent         *repository.Entitlement
		feature     string
		usageCount  int
		repoErr     error
		expectedErr error
	}{
		{
			name:        "anonymous catalog search allowed",
			ent:         nil,
			feature:     "catalog",
			usageCount:  0,
			expectedErr: nil,
		},
		{
			name:        "anonymous non-catalog denied",
			ent:         nil,
			feature:     "single",
			usageCount:  0,
			expectedErr: ErrFeatureNotAllowed,
		},
		{
			name:        "paid user unlimited",
			ent:         paidEntitlement,
			feature:     "multi",
			usageCount:  10,
			expectedErr: nil,
		},
		{
			name:        "trial user unlimited",
			ent:         trialEntitlement,
			feature:     "multi",
			usageCount:  10,
			expectedErr: nil,
		},
		{
			name:        "free user under limit",
			ent:         freeEntitlement,
			feature:     "catalog",
			usageCount:  2,
			expectedErr: nil,
		},
		{
			name:        "free user at limit",
			ent:         freeEntitlement,
			feature:     "catalog",
			usageCount:  3,
			expectedErr: ErrUsageLimitExceeded,
		},
		{
			name:        "free user restricted feature",
			ent:         freeEntitlement,
			feature:     "multi",
			usageCount:  0,
			expectedErr: ErrFeatureNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockUsageRepo{searchCount: tt.usageCount}
			if tt.repoErr != nil {
				mockRepo.getUsageSinceFunc = func(ctx context.Context, u uuid.UUID, f string, since time.Time) (repository.UsageWindow, error) {
					return repository.UsageWindow{}, tt.repoErr
				}
			}
			limiter := NewUsageLimiter(mockRepo, 3)
			err := limiter.CheckAccess(context.Background(), tt.ent, tt.feature, now)
			
			if tt.expectedErr != nil {
				if err == nil || err.Error() != tt.expectedErr.Error() {
					t.Errorf("expected error %v, got %v", tt.expectedErr, err)
				}
			} else if err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}

func TestUsageLimiter_Concurrency(t *testing.T) {
	userID := uuid.New()
	freeEntitlement := &repository.Entitlement{UserID: userID, Tier: "free", Status: "active"}
	mockRepo := &mockUsageRepo{}
	limiter := NewUsageLimiter(mockRepo, 3)

	var wg sync.WaitGroup
	var allowedCount int
	var mu sync.Mutex

	// Fire 10 concurrent requests
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := limiter.CheckAccess(context.Background(), freeEntitlement, "catalog", time.Now())
			if err == nil {
				mu.Lock()
				allowedCount++
				mu.Unlock()
				// Simulate successful search delay
				time.Sleep(10 * time.Millisecond)
				limiter.RecordUsage(context.Background(), freeEntitlement, "catalog", time.Now(), true)
			}
		}()
	}

	wg.Wait()

	if allowedCount != 3 {
		t.Errorf("expected exactly 3 concurrent requests to be allowed, got %d", allowedCount)
	}
	if mockRepo.searchCount != 3 {
		t.Errorf("expected mockRepo searchCount to be 3, got %d", mockRepo.searchCount)
	}
}

func TestUsageLimiter_DeniedAttempts(t *testing.T) {
	userID := uuid.New()
	freeEntitlement := &repository.Entitlement{UserID: userID, Tier: "free", Status: "active"}
	mockRepo := &mockUsageRepo{}
	limiter := NewUsageLimiter(mockRepo, 3)

	// Check access and it should be allowed
	err := limiter.CheckAccess(context.Background(), freeEntitlement, "catalog", time.Now())
	if err != nil {
		t.Errorf("expected access to be allowed, got %v", err)
	}

	// But record as failed (e.g. validation error during search)
	err = limiter.RecordUsage(context.Background(), freeEntitlement, "catalog", time.Now(), false)
	if err != nil {
		t.Errorf("expected record to succeed, got %v", err)
	}

	// searchCount should still be 0
	if mockRepo.searchCount != 0 {
		t.Errorf("expected searchCount to remain 0 after failed search, got %d", mockRepo.searchCount)
	}

	// Try again, should still be allowed
	err = limiter.CheckAccess(context.Background(), freeEntitlement, "catalog", time.Now())
	if err != nil {
		t.Errorf("expected access to be allowed, got %v", err)
	}
}

func TestUsageLimiter_MissingCoverage(t *testing.T) {
	userID := uuid.New()
	freeEntitlement := &repository.Entitlement{UserID: userID, Tier: "free", Status: "active"}
	paidEntitlement := &repository.Entitlement{UserID: userID, Tier: "paid", Status: "active"}
	
	// Test CheckAccess db error
	mockRepoErr := &mockUsageRepo{
		getUsageSinceFunc: func(ctx context.Context, u uuid.UUID, f string, since time.Time) (repository.UsageWindow, error) {
			return repository.UsageWindow{}, errors.New("db err")
		},
	}
	limiterErr := NewUsageLimiter(mockRepoErr, 3)
	err := limiterErr.CheckAccess(context.Background(), freeEntitlement, "catalog", time.Now())
	if err == nil || err.Error() != "db err" {
		t.Errorf("expected db err, got %v", err)
	}

	// Test RecordUsage for anonymous user
	mockRepo2 := &mockUsageRepo{}
	limiter2 := NewUsageLimiter(mockRepo2, 3)
	err = limiter2.RecordUsage(context.Background(), nil, "catalog", time.Now(), true)
	if err != nil {
		t.Errorf("expected nil error for anonymous RecordUsage, got %v", err)
	}

	// Test RecordUsage for paid user
	err = limiter2.RecordUsage(context.Background(), paidEntitlement, "catalog", time.Now(), true)
	if err != nil {
		t.Errorf("expected nil error for paid RecordUsage, got %v", err)
	}

	// Test RecordUsage with negative flight (force decrement below 0)
	err = limiter2.RecordUsage(context.Background(), freeEntitlement, "catalog", time.Now(), true)
	if err != nil {
		t.Errorf("expected nil error for negative flight RecordUsage, got %v", err)
	}
}
