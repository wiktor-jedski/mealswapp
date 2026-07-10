package entitlement

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// Implements DESIGN-007 UsageLimiter rolling free-tier search accounting.
const (
	// Implements DESIGN-007 UsageLimiter rolling 24-hour free-tier window.
	freeUsageWindowDuration = 24 * time.Hour
	// UsageFeatureSearch identifies counted search usage in usage-window persistence.
	// Implements DESIGN-007 UsageLimiter repository-backed counted search feature.
	UsageFeatureSearch = "search"
)

// UsageDenyReason identifies why a usage decision was blocked.
// Implements DESIGN-007 UsageLimiter.
type UsageDenyReason string

// Implements DESIGN-007 UsageLimiter decision states.
const (
	// UsageDenyReasonNone means usage limiting did not block the request.
	UsageDenyReasonNone UsageDenyReason = ""
	// UsageDenyReasonEntitlement means entitlement policy blocked the request.
	UsageDenyReasonEntitlement UsageDenyReason = "entitlement_denied"
	// UsageDenyReasonFreeLimitReached means the free-tier rolling limit is exhausted.
	UsageDenyReasonFreeLimitReached UsageDenyReason = "free_limit_reached"
)

// UsageDecision reports whether a search may proceed before dispatch.
// Implements DESIGN-007 UsageLimiter.
type UsageDecision struct {
	UserID             *uuid.UUID
	Feature            Feature
	Allowed            bool
	CountUsageOnFinish bool
	Limit              int
	Used               int
	Remaining          int
	Tier               string
	Status             string
	DenyReason         UsageDenyReason
	EntitlementReason  DenyReason
	WindowStartedAt    time.Time
}

// UsageRequest carries server-side identity and the requested feature.
// Completed searches should call RecordCompletedSearch only when the returned
// decision has CountUsageOnFinish set.
// Implements DESIGN-007 UsageLimiter.
type UsageRequest struct {
	UserID  *uuid.UUID
	Feature Feature
}

// UsageLimiter enforces free-tier rolling usage before paid-mode dispatch.
// Implements DESIGN-007 UsageLimiter.
type UsageLimiter struct {
	entitlements *EntitlementManager
	usage        repository.UsageRepository
	now          func() time.Time
	locks        keyedLocks
}

// NewUsageLimiter creates a usage limiter with a real clock.
// Implements DESIGN-007 UsageLimiter.
func NewUsageLimiter(entitlements *EntitlementManager, usage repository.UsageRepository) *UsageLimiter {
	return NewUsageLimiterWithClock(entitlements, usage, time.Now)
}

// NewUsageLimiterWithClock creates a usage limiter with an injectable clock.
// Implements DESIGN-007 UsageLimiter.
func NewUsageLimiterWithClock(entitlements *EntitlementManager, usage repository.UsageRepository, now func() time.Time) *UsageLimiter {
	if now == nil {
		now = time.Now
	}
	return &UsageLimiter{entitlements: entitlements, usage: usage, now: now}
}

// CheckSearchAllowed decides whether the search may dispatch.
// Implements DESIGN-007 UsageLimiter and DESIGN-018 AuthenticatedActionGuard.
func (l *UsageLimiter) CheckSearchAllowed(ctx context.Context, req UsageRequest) (UsageDecision, error) {
	if err := l.validate(); err != nil {
		return UsageDecision{}, err
	}
	if !validFeature(req.Feature) {
		return UsageDecision{Feature: req.Feature, DenyReason: UsageDenyReasonEntitlement, EntitlementReason: DenyReasonInvalidFeature}, nil
	}
	if req.UserID == nil {
		return anonymousDecision(req.Feature), nil
	}
	if *req.UserID == uuid.Nil {
		return UsageDecision{}, repository.NewError(repository.ErrorKindValidation, "authenticated user id is required", nil)
	}

	userID := *req.UserID
	unlock := l.locks.lock(userID, UsageFeatureSearch)
	defer unlock()

	entitlementDecision, err := l.entitlements.CheckEntitlement(ctx, userID, req.Feature)
	if err != nil {
		return UsageDecision{}, err
	}
	decision := usageDecisionFromEntitlement(entitlementDecision)
	if !entitlementDecision.Allowed {
		return decision, nil
	}
	if !effectiveFreeUsageScope(entitlementDecision) {
		decision.Allowed = true
		return decision, nil
	}

	limit := freeSearchLimitPer24h
	since := l.now().UTC().Add(-freeUsageWindowDuration)
	window, err := l.usage.GetUsageSince(ctx, userID, UsageFeatureSearch, since)
	if err != nil {
		return UsageDecision{}, err
	}

	decision.Limit = limit
	decision.Used = window.SearchCount
	decision.Remaining = max(limit-window.SearchCount, 0)
	decision.WindowStartedAt = since
	if window.SearchCount >= limit {
		decision.Allowed = false
		decision.CountUsageOnFinish = false
		decision.DenyReason = UsageDenyReasonFreeLimitReached
		return decision, nil
	}

	decision.Allowed = true
	decision.CountUsageOnFinish = true
	return decision, nil
}

// RecordCompletedSearch records one allowed counted search after successful completion.
// Implements DESIGN-007 UsageLimiter.
func (l *UsageLimiter) RecordCompletedSearch(ctx context.Context, decision UsageDecision) (UsageDecision, repository.UsageWindow, error) {
	if err := l.validate(); err != nil {
		return UsageDecision{}, repository.UsageWindow{}, err
	}
	if !decision.CountUsageOnFinish || decision.UserID == nil {
		return decision, repository.UsageWindow{}, nil
	}
	if *decision.UserID == uuid.Nil {
		return UsageDecision{}, repository.UsageWindow{}, repository.NewError(repository.ErrorKindValidation, "authenticated user id is required", nil)
	}

	userID := *decision.UserID
	unlock := l.locks.lock(userID, UsageFeatureSearch)
	defer unlock()

	now := l.now().UTC()
	since := now.Add(-freeUsageWindowDuration)
	recordedWindow, recorded, err := l.usage.RecordUsageWithinLimit(ctx, userID, UsageFeatureSearch, now, since, decision.Limit)
	if err != nil {
		return UsageDecision{}, repository.UsageWindow{}, err
	}
	decision.Used = recordedWindow.SearchCount
	decision.Remaining = max(decision.Limit-recordedWindow.SearchCount, 0)
	decision.WindowStartedAt = since
	if !recorded {
		decision.Allowed = false
		decision.CountUsageOnFinish = false
		decision.DenyReason = UsageDenyReasonFreeLimitReached
		return decision, repository.UsageWindow{}, nil
	}

	decision.CountUsageOnFinish = false
	return decision, recordedWindow, nil
}

// validate checks UsageLimiter dependencies before request handling.
// Implements DESIGN-007 UsageLimiter.
func (l *UsageLimiter) validate() error {
	if l == nil || l.entitlements == nil || l.usage == nil {
		return repository.NewError(repository.ErrorKindValidation, "usage limiter dependencies are required", nil)
	}
	return nil
}

// anonymousDecision resolves unauthenticated access without usage persistence.
// Implements DESIGN-007 UsageLimiter and DESIGN-018 AuthenticatedActionGuard.
func anonymousDecision(feature Feature) UsageDecision {
	decision := UsageDecision{Feature: feature}
	if feature == FeatureCatalog {
		decision.Allowed = true
		return decision
	}
	decision.DenyReason = UsageDenyReasonEntitlement
	decision.EntitlementReason = DenyReasonFreeTierScope
	return decision
}

// usageDecisionFromEntitlement maps entitlement state to usage-limiter state.
// Implements DESIGN-007 UsageLimiter.
func usageDecisionFromEntitlement(decision Decision) UsageDecision {
	userID := decision.UserID
	result := UsageDecision{
		UserID:            &userID,
		Feature:           decision.Feature,
		Tier:              decision.Tier,
		Status:            decision.Status,
		EntitlementReason: decision.DenyReason,
	}
	if !decision.Allowed {
		result.DenyReason = UsageDenyReasonEntitlement
	}
	return result
}

// effectiveFreeUsageScope reports whether allowed free-scope searches use the free usage cap.
// Implements DESIGN-007 UsageLimiter.
func effectiveFreeUsageScope(decision Decision) bool {
	if !decision.Allowed || !freeFeature(decision.Feature) {
		return false
	}
	if decision.Tier == "free" {
		return decision.Status == "active"
	}
	return decision.Status != "active"
}

// IsUsageLimitError reports deterministic free-tier limit denials.
// Implements DESIGN-007 UsageLimiter.
func IsUsageLimitError(decision UsageDecision) bool {
	return !decision.Allowed && decision.DenyReason == UsageDenyReasonFreeLimitReached
}

// keyedLocks serializes in-process same-user checks as a fast local guard.
// Implements DESIGN-007 UsageLimiter.
type keyedLocks struct {
	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

// lock returns an unlock function for one user and usage feature.
// Implements DESIGN-007 UsageLimiter.
func (l *keyedLocks) lock(userID uuid.UUID, feature string) func() {
	key := userID.String() + ":" + feature
	l.mu.Lock()
	if l.locks == nil {
		l.locks = map[string]*sync.Mutex{}
	}
	lock, ok := l.locks[key]
	if !ok {
		lock = &sync.Mutex{}
		l.locks[key] = lock
	}
	l.mu.Unlock()

	lock.Lock()
	return func() {
		lock.Unlock()
	}
}

// IsUsageValidationError reports validation failures from this service boundary.
// Implements DESIGN-007 UsageLimiter.
func IsUsageValidationError(err error) bool {
	var repoErr *repository.Error
	return errors.As(err, &repoErr) && repoErr.Kind == repository.ErrorKindValidation
}
