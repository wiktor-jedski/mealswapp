// Implements DESIGN-007 UsageLimiter verification.
package entitlement

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

type usageRepositoryStub struct {
	mu           sync.Mutex
	records      map[uuid.UUID][]time.Time
	getCalls     int
	recordCalls  int
	recordWaiter chan struct{}
	getErr       error
	recordErr    error
}

func (r *usageRepositoryStub) RecordUsage(_ context.Context, userID uuid.UUID, feature string, occurredAt time.Time) (repository.UsageWindow, error) {
	if r.recordWaiter != nil {
		<-r.recordWaiter
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	r.recordCalls++
	if r.recordErr != nil {
		return repository.UsageWindow{}, r.recordErr
	}
	if r.records == nil {
		r.records = map[uuid.UUID][]time.Time{}
	}
	r.records[userID] = append(r.records[userID], occurredAt)
	return repository.UsageWindow{
		UserID:      userID,
		Feature:     feature,
		StartedAt:   occurredAt,
		SearchCount: len(r.records[userID]),
		CreatedAt:   occurredAt,
		UpdatedAt:   occurredAt,
	}, nil
}

func (r *usageRepositoryStub) RecordUsageWithinLimit(_ context.Context, userID uuid.UUID, feature string, occurredAt time.Time, since time.Time, limit int) (repository.UsageWindow, bool, error) {
	if r.recordWaiter != nil {
		<-r.recordWaiter
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	r.recordCalls++
	if r.recordErr != nil {
		return repository.UsageWindow{}, false, r.recordErr
	}
	count := 0
	for _, existing := range r.records[userID] {
		if !existing.Before(since) {
			count++
		}
	}
	if count >= limit {
		return repository.UsageWindow{UserID: userID, Feature: feature, StartedAt: since, SearchCount: count, CreatedAt: since, UpdatedAt: since}, false, nil
	}
	if r.records == nil {
		r.records = map[uuid.UUID][]time.Time{}
	}
	r.records[userID] = append(r.records[userID], occurredAt)
	return repository.UsageWindow{UserID: userID, Feature: feature, StartedAt: since, SearchCount: count + 1, CreatedAt: since, UpdatedAt: occurredAt}, true, nil
}

func (r *usageRepositoryStub) GetUsageSince(_ context.Context, userID uuid.UUID, feature string, since time.Time) (repository.UsageWindow, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.getCalls++
	if r.getErr != nil {
		return repository.UsageWindow{}, r.getErr
	}
	count := 0
	for _, occurredAt := range r.records[userID] {
		if !occurredAt.Before(since) {
			count++
		}
	}
	return repository.UsageWindow{
		UserID:      userID,
		Feature:     feature,
		StartedAt:   since,
		SearchCount: count,
		CreatedAt:   since,
		UpdatedAt:   since,
	}, nil
}

func (r *usageRepositoryStub) count(userID uuid.UUID) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.records[userID])
}

func TestUsageLimiterCapsFreeUsersAtThreeCountedSearchesPerRolling24Hours(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	userID := uuid.New()
	usage := &usageRepositoryStub{records: map[uuid.UUID][]time.Time{
		userID: {
			now.Add(-25 * time.Hour),
			now.Add(-23 * time.Hour),
			now.Add(-2 * time.Hour),
		},
	}}
	limiter := newUsageLimiterFixture(userID, freeEntitlement(userID), usage, now)

	decision, err := limiter.CheckSearchAllowed(context.Background(), UsageRequest{UserID: &userID, Feature: FeatureCatalog})
	if err != nil {
		t.Fatalf("CheckSearchAllowed() error = %v", err)
	}
	if !decision.Allowed || !decision.CountUsageOnFinish || decision.Used != 2 || decision.Remaining != 1 {
		t.Fatalf("decision with two rolling records = %+v, want one remaining", decision)
	}
	decision, _, err = limiter.RecordCompletedSearch(context.Background(), decision)
	if err != nil {
		t.Fatalf("RecordCompletedSearch() error = %v", err)
	}
	if !decision.Allowed || decision.Used != 3 || decision.Remaining != 0 {
		t.Fatalf("recorded third search decision = %+v, want exhausted allow", decision)
	}

	decision, err = limiter.CheckSearchAllowed(context.Background(), UsageRequest{UserID: &userID, Feature: FeatureSingleSubstitution})
	if err != nil {
		t.Fatalf("CheckSearchAllowed() exhausted error = %v", err)
	}
	if decision.Allowed || decision.CountUsageOnFinish || decision.DenyReason != UsageDenyReasonFreeLimitReached || !IsUsageLimitError(decision) {
		t.Fatalf("exhausted decision = %+v, want deterministic free limit denial", decision)
	}
	if usage.count(userID) != 4 {
		t.Fatalf("persisted records = %d, want old outside-window plus three counted records", usage.count(userID))
	}
}

func TestUsageLimiterDoesNotCountDeniedAttemptsOrBeforeCompletion(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	userID := uuid.New()
	usage := &usageRepositoryStub{records: map[uuid.UUID][]time.Time{
		userID: {now.Add(-time.Hour), now.Add(-30 * time.Minute), now.Add(-time.Minute)},
	}}
	limiter := newUsageLimiterFixture(userID, freeEntitlement(userID), usage, now)

	denied, err := limiter.CheckSearchAllowed(context.Background(), UsageRequest{UserID: &userID, Feature: FeatureCatalog})
	if err != nil {
		t.Fatalf("CheckSearchAllowed() error = %v", err)
	}
	if denied.Allowed || denied.CountUsageOnFinish {
		t.Fatalf("denied decision = %+v, want no completion record", denied)
	}
	if _, _, err := limiter.RecordCompletedSearch(context.Background(), denied); err != nil {
		t.Fatalf("RecordCompletedSearch() denied error = %v", err)
	}
	if usage.count(userID) != 3 {
		t.Fatalf("denied attempt records = %d, want unchanged", usage.count(userID))
	}

	otherUserID := uuid.New()
	otherUsage := &usageRepositoryStub{}
	otherLimiter := newUsageLimiterFixture(otherUserID, freeEntitlement(otherUserID), otherUsage, now)
	allowed, err := otherLimiter.CheckSearchAllowed(context.Background(), UsageRequest{UserID: &otherUserID, Feature: FeatureCatalog})
	if err != nil {
		t.Fatalf("CheckSearchAllowed() allowed error = %v", err)
	}
	if !allowed.CountUsageOnFinish || otherUsage.recordCalls != 0 {
		t.Fatalf("allowed before completion decision=%+v recordCalls=%d, want no write until completion", allowed, otherUsage.recordCalls)
	}
}

func TestUsageLimiterDoesNotCapTrialOrPaidActiveUsers(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	for _, tier := range []string{"trial", "paid"} {
		t.Run(tier, func(t *testing.T) {
			userID := uuid.New()
			usage := &usageRepositoryStub{records: map[uuid.UUID][]time.Time{
				userID: {now.Add(-time.Hour), now.Add(-50 * time.Minute), now.Add(-40 * time.Minute), now.Add(-30 * time.Minute)},
			}}
			limiter := newUsageLimiterFixture(userID, paidEntitlement(userID, tier, "active"), usage, now)

			decision, err := limiter.CheckSearchAllowed(context.Background(), UsageRequest{UserID: &userID, Feature: FeatureDailyDietAlternative})
			if err != nil {
				t.Fatalf("CheckSearchAllowed() error = %v", err)
			}
			if !decision.Allowed || decision.CountUsageOnFinish || usage.getCalls != 0 || usage.recordCalls != 0 {
				t.Fatalf("paid-scope decision=%+v getCalls=%d recordCalls=%d, want unmetered allow", decision, usage.getCalls, usage.recordCalls)
			}
		})
	}
}

func TestUsageLimiterAllowsAnonymousCatalogWithoutUsageWrites(t *testing.T) {
	usage := &usageRepositoryStub{}
	limiter := newUsageLimiterFixture(uuid.New(), freeEntitlement(uuid.New()), usage, time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC))

	decision, err := limiter.CheckSearchAllowed(context.Background(), UsageRequest{Feature: FeatureCatalog})
	if err != nil {
		t.Fatalf("CheckSearchAllowed() error = %v", err)
	}
	if !decision.Allowed || decision.CountUsageOnFinish || decision.UserID != nil || usage.getCalls != 0 || usage.recordCalls != 0 {
		t.Fatalf("anonymous catalog decision=%+v getCalls=%d recordCalls=%d, want unmetered allow", decision, usage.getCalls, usage.recordCalls)
	}

	decision, err = limiter.CheckSearchAllowed(context.Background(), UsageRequest{Feature: FeatureSingleSubstitution})
	if err != nil {
		t.Fatalf("CheckSearchAllowed() anonymous substitution error = %v", err)
	}
	if decision.Allowed || decision.DenyReason != UsageDenyReasonEntitlement {
		t.Fatalf("anonymous substitution decision=%+v, want entitlement denial", decision)
	}
}

func TestUsageLimiterBlocksPaidModesBeforeUsageDispatchForFreeUsers(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	userID := uuid.New()
	usage := &usageRepositoryStub{}
	limiter := newUsageLimiterFixture(userID, freeEntitlement(userID), usage, now)

	decision, err := limiter.CheckSearchAllowed(context.Background(), UsageRequest{UserID: &userID, Feature: FeatureDailyDietAlternative})
	if err != nil {
		t.Fatalf("CheckSearchAllowed() error = %v", err)
	}
	if decision.Allowed || decision.DenyReason != UsageDenyReasonEntitlement || decision.EntitlementReason != DenyReasonFreeTierScope {
		t.Fatalf("paid-mode decision = %+v, want entitlement denial", decision)
	}
	if usage.getCalls != 0 || usage.recordCalls != 0 {
		t.Fatalf("usage calls get=%d record=%d, want no usage persistence before paid-mode dispatch", usage.getCalls, usage.recordCalls)
	}
}

func TestUsageLimiterConcurrentSameUserCompletionsCannotExceedPersistedLimit(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	userID := uuid.New()
	usage := &usageRepositoryStub{}
	limiter := newUsageLimiterFixture(userID, freeEntitlement(userID), usage, now)

	start := make(chan struct{})
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			decision := UsageDecision{
				UserID:             &userID,
				Feature:            FeatureCatalog,
				Allowed:            true,
				CountUsageOnFinish: true,
				Limit:              freeSearchLimitPer24h,
				Tier:               "free",
				Status:             "active",
			}
			if _, _, err := limiter.RecordCompletedSearch(context.Background(), decision); err != nil {
				t.Errorf("RecordCompletedSearch() error = %v", err)
			}
		}()
	}
	close(start)
	wg.Wait()

	if usage.count(userID) != freeSearchLimitPer24h {
		t.Fatalf("persisted usage count = %d, want %d", usage.count(userID), freeSearchLimitPer24h)
	}
}

func TestUsageLimiterValidationErrors(t *testing.T) {
	if _, err := (*UsageLimiter)(nil).CheckSearchAllowed(context.Background(), UsageRequest{Feature: FeatureCatalog}); !IsUsageValidationError(err) {
		t.Fatalf("nil limiter error = %v, want validation", err)
	}

	if _, err := NewUsageLimiter(nil, &usageRepositoryStub{}).CheckSearchAllowed(context.Background(), UsageRequest{Feature: FeatureCatalog}); !IsUsageValidationError(err) {
		t.Fatalf("missing entitlement dependency error = %v, want validation", err)
	}

	userID := uuid.Nil
	limiter := newUsageLimiterFixture(uuid.New(), freeEntitlement(uuid.New()), &usageRepositoryStub{}, time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC))
	if _, err := limiter.CheckSearchAllowed(context.Background(), UsageRequest{UserID: &userID, Feature: FeatureCatalog}); !IsUsageValidationError(err) {
		t.Fatalf("nil user error = %v, want validation", err)
	}

	invalid, err := limiter.CheckSearchAllowed(context.Background(), UsageRequest{Feature: Feature("unknown")})
	if err != nil || invalid.Allowed || invalid.EntitlementReason != DenyReasonInvalidFeature {
		t.Fatalf("invalid feature decision=%+v err=%v, want invalid feature denial", invalid, err)
	}
}

func TestUsageLimiterPropagatesUsageRepositoryErrors(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	userID := uuid.New()
	wantErr := errors.New("usage unavailable")
	limiter := newUsageLimiterFixture(userID, freeEntitlement(userID), &usageRepositoryStub{getErr: wantErr}, now)

	if _, err := limiter.CheckSearchAllowed(context.Background(), UsageRequest{UserID: &userID, Feature: FeatureCatalog}); !errors.Is(err, wantErr) {
		t.Fatalf("CheckSearchAllowed() error = %v, want %v", err, wantErr)
	}

	limiter = newUsageLimiterFixture(userID, freeEntitlement(userID), &usageRepositoryStub{recordErr: wantErr}, now)
	decision := UsageDecision{UserID: &userID, Feature: FeatureCatalog, Allowed: true, CountUsageOnFinish: true, Limit: freeSearchLimitPer24h, Tier: "free", Status: "active"}
	if _, _, err := limiter.RecordCompletedSearch(context.Background(), decision); !errors.Is(err, wantErr) {
		t.Fatalf("RecordCompletedSearch() error = %v, want %v", err, wantErr)
	}
}

func TestUsageLimiterDefaultClockConstructor(t *testing.T) {
	userID := uuid.New()
	usage := &usageRepositoryStub{}
	entitlementRepo := &entitlementRepositoryStub{entitlements: map[uuid.UUID]repository.Entitlement{
		userID: freeEntitlement(userID),
	}}
	limiter := NewUsageLimiterWithClock(NewEntitlementManager(entitlementRepo), usage, nil)

	decision, err := limiter.CheckSearchAllowed(context.Background(), UsageRequest{UserID: &userID, Feature: FeatureCatalog})
	if err != nil {
		t.Fatalf("CheckSearchAllowed() error = %v", err)
	}
	if !decision.Allowed || decision.WindowStartedAt.IsZero() {
		t.Fatalf("default clock decision=%+v, want allowed with window", decision)
	}
}

func newUsageLimiterFixture(userID uuid.UUID, entitlement repository.Entitlement, usage *usageRepositoryStub, now time.Time) *UsageLimiter {
	entitlementRepo := &entitlementRepositoryStub{entitlements: map[uuid.UUID]repository.Entitlement{
		userID: entitlement,
	}}
	return NewUsageLimiterWithClock(NewEntitlementManager(entitlementRepo), usage, func() time.Time {
		return now
	})
}
