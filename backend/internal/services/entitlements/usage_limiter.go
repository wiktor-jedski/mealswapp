package entitlements

import (
	"context"
	"sync"
	"time"

	searchsvc "mealswapp/backend/internal/services/search"

	"github.com/google/uuid"
)

const anonymousUsageKey = "anonymous"

type TokenResolver interface {
	UserIDFromAccessToken(ctx context.Context, accessToken string) (uuid.UUID, bool, error)
}

type UsageLimiter struct {
	manager  Manager
	resolver TokenResolver
	store    *MemoryUsageStore
	now      func() time.Time
}

type UsageWindow struct {
	Key         string
	StartedAt   time.Time
	SearchCount int
}

func NewUsageLimiter(manager Manager, resolver TokenResolver, store *MemoryUsageStore) UsageLimiter {
	return NewUsageLimiterWithClock(manager, resolver, store, time.Now)
}

func NewUsageLimiterWithClock(manager Manager, resolver TokenResolver, store *MemoryUsageStore, now func() time.Time) UsageLimiter {
	if store == nil {
		store = NewMemoryUsageStore()
	}
	return UsageLimiter{manager: manager, resolver: resolver, store: store, now: now}
}

func (limiter UsageLimiter) CheckAndRecord(ctx context.Context, accessToken string, mode searchsvc.Mode) (Decision, error) {
	userID, key, err := limiter.resolveUser(ctx, accessToken)
	if err != nil {
		return Decision{}, err
	}
	window := limiter.store.Get(key, limiter.now())
	decision, err := limiter.manager.CheckMode(ctx, userID, mode, window.SearchCount)
	if err != nil {
		return Decision{}, err
	}
	if !decision.Allowed {
		return decision, nil
	}
	limiter.store.Increment(key, limiter.now())
	return decision, nil
}

func (limiter UsageLimiter) Window(ctx context.Context, accessToken string) (UsageWindow, error) {
	_, key, err := limiter.resolveUser(ctx, accessToken)
	if err != nil {
		return UsageWindow{}, err
	}
	return limiter.store.Get(key, limiter.now()), nil
}

func (limiter UsageLimiter) resolveUser(ctx context.Context, accessToken string) (*uuid.UUID, string, error) {
	if accessToken == "" || limiter.resolver == nil {
		return nil, anonymousUsageKey, nil
	}
	userID, ok, err := limiter.resolver.UserIDFromAccessToken(ctx, accessToken)
	if err != nil {
		return nil, "", err
	}
	if !ok || userID == uuid.Nil {
		return nil, anonymousUsageKey, nil
	}
	return &userID, userID.String(), nil
}

type MemoryUsageStore struct {
	mu      sync.Mutex
	windows map[string]UsageWindow
}

func NewMemoryUsageStore() *MemoryUsageStore {
	return &MemoryUsageStore{windows: map[string]UsageWindow{}}
}

func (store *MemoryUsageStore) Get(key string, now time.Time) UsageWindow {
	store.mu.Lock()
	defer store.mu.Unlock()

	window, ok := store.windows[key]
	if !ok || !now.Before(window.StartedAt.Add(24*time.Hour)) {
		window = UsageWindow{Key: key, StartedAt: now.UTC()}
		store.windows[key] = window
	}
	return window
}

func (store *MemoryUsageStore) Increment(key string, now time.Time) UsageWindow {
	store.mu.Lock()
	defer store.mu.Unlock()

	window, ok := store.windows[key]
	if !ok || !now.Before(window.StartedAt.Add(24*time.Hour)) {
		window = UsageWindow{Key: key, StartedAt: now.UTC()}
	}
	window.SearchCount++
	store.windows[key] = window
	return window
}
