// Implements DESIGN-007 UsageLimiter PostgreSQL integration verification.
package entitlement

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wiktor-jedski/mealswapp/backend/internal/migrations"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// Implements DESIGN-007 UsageLimiter PostgreSQL integration fixture.
const usageLimiterTestDatabaseURL = "postgres://mealswapp:mealswapp@localhost:5432/mealswapp?sslmode=disable"

// TestUsageLimiterPostgresConcurrentSeparateInstancesCannotExceedPersistedLimit verifies atomic persisted free-tier enforcement.
// Implements DESIGN-007 UsageLimiter.
func TestUsageLimiterPostgresConcurrentSeparateInstancesCannotExceedPersistedLimit(t *testing.T) {
	// Verifies IT-ARCH-007-001.
	// Verifies ARCH-007.
	// Verifies ARCH-002.
	// Verifies ARCH-005.
	// Traces SW-REQ-042, SW-REQ-052, and SW-REQ-053.
	db := openUsageLimiterTestDB(t)
	ctx := context.Background()

	userID := createUsageLimiterUser(t, ctx, db, "task-159-concurrency-"+uuid.NewString())
	entitlements := repository.NewPostgresEntitlementRepository(db)
	if err := entitlements.AppendEntitlement(ctx, repository.Entitlement{
		UserID:            userID,
		Tier:              "free",
		Status:            "active",
		SearchLimitPer24h: freeSearchLimitPer24h,
		AllowedModes:      []string{"catalog", "substitution"},
	}); err != nil {
		t.Fatalf("AppendEntitlement() error = %v", err)
	}

	usage := repository.NewPostgresEntitlementRepository(db)
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	decisions := make([]UsageDecision, 8)
	for i := range decisions {
		limiter := NewUsageLimiterWithClock(
			NewEntitlementManager(repository.NewPostgresEntitlementRepository(db)),
			repository.NewPostgresEntitlementRepository(db),
			func() time.Time { return now },
		)
		decision, err := limiter.CheckSearchAllowed(ctx, UsageRequest{UserID: &userID, Feature: FeatureCatalog})
		if err != nil {
			t.Fatalf("CheckSearchAllowed(%d) error = %v", i, err)
		}
		if !decision.Allowed || !decision.CountUsageOnFinish {
			t.Fatalf("CheckSearchAllowed(%d) decision = %+v, want stale allowed decision", i, decision)
		}
		decisions[i] = decision
	}

	var wg sync.WaitGroup
	recorded := make(chan bool, len(decisions))
	for i, decision := range decisions {
		wg.Add(1)
		go func(i int, decision UsageDecision) {
			defer wg.Done()
			limiter := NewUsageLimiterWithClock(
				NewEntitlementManager(repository.NewPostgresEntitlementRepository(db)),
				repository.NewPostgresEntitlementRepository(db),
				func() time.Time { return now.Add(time.Duration(i) * time.Millisecond) },
			)
			updated, _, err := limiter.RecordCompletedSearch(ctx, decision)
			if err != nil {
				t.Errorf("RecordCompletedSearch(%d) error = %v", i, err)
				return
			}
			recorded <- updated.Allowed
		}(i, decision)
	}
	wg.Wait()
	close(recorded)

	allowedCompletions := 0
	for allowed := range recorded {
		if allowed {
			allowedCompletions++
		}
	}
	if allowedCompletions != freeSearchLimitPer24h {
		t.Fatalf("allowed completions = %d, want %d", allowedCompletions, freeSearchLimitPer24h)
	}

	window, err := usage.GetUsageSince(ctx, userID, UsageFeatureSearch, now.Add(-freeUsageWindowDuration))
	if err != nil {
		t.Fatalf("GetUsageSince() error = %v", err)
	}
	if window.SearchCount != freeSearchLimitPer24h {
		t.Fatalf("persisted usage count = %d, want %d", window.SearchCount, freeSearchLimitPer24h)
	}
}

// openUsageLimiterTestDB resets the PostgreSQL integration database for UsageLimiter tests.
// Implements DESIGN-007 UsageLimiter.
func openUsageLimiterTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	databaseURL := os.Getenv("MEALSWAPP_DATABASE_URL")
	if databaseURL == "" {
		databaseURL = usageLimiterTestDatabaseURL
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("postgres unavailable: %v", err)
	}
	if _, err := pool.Exec(ctx, "SELECT pg_advisory_lock(9010101)"); err != nil {
		pool.Close()
		t.Fatalf("acquire usage limiter test database lock: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "SELECT pg_advisory_unlock(9010101)")
	})

	migrationDir, err := filepath.Abs("../../../database/migrations")
	if err != nil {
		pool.Close()
		t.Fatalf("resolve migration dir: %v", err)
	}
	if err := migrations.Run(ctx, pool, "up", migrationDir); err != nil {
		pool.Close()
		t.Fatalf("apply migrations up: %v", err)
	}

	t.Cleanup(pool.Close)
	return pool
}

// createUsageLimiterUser inserts one encrypted user for UsageLimiter integration tests.
// Implements DESIGN-007 UsageLimiter.
func createUsageLimiterUser(t *testing.T, ctx context.Context, db *pgxpool.Pool, digest string) uuid.UUID {
	t.Helper()

	userID, err := repository.NewPostgresEncryptedIdentityRepository(db).CreateUser(ctx, repository.EncryptedAuthUser{
		Email:                 repository.EncryptedField{KeyVersion: "pii-v1", Nonce: []byte("email-nonce-" + digest), Ciphertext: []byte("email-ciphertext-" + digest)},
		NormalizedEmailDigest: repository.LookupDigest{KeyVersion: "lookup-v1", Value: "email-digest-" + digest},
		Role:                  repository.UserRoleUser,
	})
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	return userID
}
