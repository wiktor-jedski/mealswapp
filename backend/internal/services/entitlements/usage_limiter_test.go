package entitlements

import (
	"context"
	"testing"
	"time"

	"mealswapp/backend/internal/repositories"
	searchsvc "mealswapp/backend/internal/services/search"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestUsageLimiterCountsFreeSearchesAndBlocksFourth(t *testing.T) {
	now := fixedNow()
	manager := NewManagerWithClock(fakeRepository{err: pgx.ErrNoRows}, func() time.Time { return now })
	limiter := NewUsageLimiterWithClock(manager, nil, NewMemoryUsageStore(), func() time.Time { return now })

	for i := 0; i < 3; i++ {
		decision, err := limiter.CheckAndRecord(context.Background(), "", searchsvc.ModeSingle)
		if err != nil {
			t.Fatal(err)
		}
		if !decision.Allowed {
			t.Fatalf("expected search %d to be allowed, got %#v", i+1, decision)
		}
	}
	decision, err := limiter.CheckAndRecord(context.Background(), "", searchsvc.ModeSingle)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Allowed || decision.Code != "search_limit_reached" {
		t.Fatalf("expected fourth search blocked, got %#v", decision)
	}
}

func TestUsageLimiterResetsWindowAfter24Hours(t *testing.T) {
	now := fixedNow()
	manager := NewManagerWithClock(fakeRepository{err: pgx.ErrNoRows}, func() time.Time { return now })
	limiter := NewUsageLimiterWithClock(manager, nil, NewMemoryUsageStore(), func() time.Time { return now })

	for i := 0; i < 3; i++ {
		if decision, err := limiter.CheckAndRecord(context.Background(), "", searchsvc.ModeSingle); err != nil || !decision.Allowed {
			t.Fatalf("expected initial search %d allowed, decision=%#v err=%v", i+1, decision, err)
		}
	}
	now = now.Add(24*time.Hour + time.Second)
	decision, err := limiter.CheckAndRecord(context.Background(), "", searchsvc.ModeSingle)
	if err != nil {
		t.Fatal(err)
	}
	if !decision.Allowed {
		t.Fatalf("expected search allowed after reset, got %#v", decision)
	}
	window, err := limiter.Window(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if window.SearchCount != 1 {
		t.Fatalf("expected reset window count 1, got %#v", window)
	}
}

func TestUsageLimiterBlocksFreePaidModesBeforeCounting(t *testing.T) {
	now := fixedNow()
	manager := NewManagerWithClock(fakeRepository{err: pgx.ErrNoRows}, func() time.Time { return now })
	limiter := NewUsageLimiterWithClock(manager, nil, NewMemoryUsageStore(), func() time.Time { return now })

	decision, err := limiter.CheckAndRecord(context.Background(), "", searchsvc.ModeDiet)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Allowed || decision.Code != "mode_not_allowed" {
		t.Fatalf("expected diet blocked for free user, got %#v", decision)
	}
	window, err := limiter.Window(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if window.SearchCount != 0 {
		t.Fatalf("blocked mode should not count usage, got %#v", window)
	}
}

func TestUsageLimiterUsesAuthenticatedPaidEntitlement(t *testing.T) {
	userID := uuid.New()
	now := fixedNow()
	manager := NewManagerWithClock(fakeRepository{entity: repositories.EntitlementEntity{
		UserID: userID,
		Plan:   "paid",
		Status: "active",
	}}, func() time.Time { return now })
	limiter := NewUsageLimiterWithClock(manager, fakeTokenResolver{userID: userID}, NewMemoryUsageStore(), func() time.Time { return now })

	decision, err := limiter.CheckAndRecord(context.Background(), "access-token", searchsvc.ModeDiet)
	if err != nil {
		t.Fatal(err)
	}
	if !decision.Allowed || decision.Entitlement.Tier != TierPaid {
		t.Fatalf("expected paid diet search allowed, got %#v", decision)
	}
}

type fakeTokenResolver struct {
	userID uuid.UUID
	ok     bool
	err    error
}

func (resolver fakeTokenResolver) UserIDFromAccessToken(ctx context.Context, accessToken string) (uuid.UUID, bool, error) {
	if resolver.err != nil {
		return uuid.Nil, false, resolver.err
	}
	if resolver.ok || resolver.userID != uuid.Nil {
		return resolver.userID, true, nil
	}
	return uuid.Nil, false, nil
}
