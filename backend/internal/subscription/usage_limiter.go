package subscription

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// UsageLimiter enforces the 24-hour rolling usage limit for free-tier users.
// Implements DESIGN-007 UsageLimiter.
type UsageLimiter struct {
	usageRepo repository.UsageRepository
	limit     int
	mu        sync.Mutex
	inFlight  map[uuid.UUID]int
}

// NewUsageLimiter creates a new UsageLimiter with the specified daily limit.
// Implements DESIGN-007 UsageLimiter.
func NewUsageLimiter(usageRepo repository.UsageRepository, limit int) *UsageLimiter {
	return &UsageLimiter{
		usageRepo: usageRepo,
		limit:     limit,
		inFlight:  make(map[uuid.UUID]int),
	}
}

// CheckAccess returns an error if the user has reached their 24-hour limit for the given feature
// or if the feature is not allowed. It increments an in-flight counter for free users.
// Implements DESIGN-007 UsageLimiter.
func (l *UsageLimiter) CheckAccess(ctx context.Context, ent *repository.Entitlement, feature string, now time.Time) error {
	// Anonymous user
	if ent == nil {
		if feature == "catalog" {
			return nil
		}
		return ErrFeatureNotAllowed
	}

	// Active trial or paid users are unlimited
	if (ent.Tier == "trial" || ent.Tier == "paid") && ent.Status == "active" {
		return nil
	}

	// Free or inactive users
	if feature != "catalog" && feature != "single" {
		return ErrFeatureNotAllowed
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	windowStart := now.Add(-24 * time.Hour)
	window, err := l.usageRepo.GetUsageSince(ctx, ent.UserID, feature, windowStart)
	if err != nil {
		return err
	}

	if window.SearchCount+l.inFlight[ent.UserID] >= l.limit {
		return ErrUsageLimitExceeded
	}

	l.inFlight[ent.UserID]++
	return nil
}

// RecordUsage records usage in the repository if the user is bound by limits and the search was successful.
// It decrements the in-flight counter.
// Implements DESIGN-007 UsageLimiter.
func (l *UsageLimiter) RecordUsage(ctx context.Context, ent *repository.Entitlement, feature string, now time.Time, success bool) error {
	if ent == nil {
		return nil
	}
	if (ent.Tier == "trial" || ent.Tier == "paid") && ent.Status == "active" {
		return nil
	}

	l.mu.Lock()
	l.inFlight[ent.UserID]--
	if l.inFlight[ent.UserID] < 0 {
		l.inFlight[ent.UserID] = 0
	}
	l.mu.Unlock()

	if success {
		_, err := l.usageRepo.RecordUsage(ctx, ent.UserID, feature, now)
		return err
	}
	return nil
}
