## FILE: EntitlementManager.md
**Traceability:** ARCH-007

### 1. Data Structures & Types

```go
package entitlement

type Tier string

const (
    TierFree  Tier = "free"
    TierTrial Tier = "trial"
    TierPaid  Tier = "paid"
)

type Entitlement struct {
    UserID        string    `json:"user_id"`
    Tier          Tier      `json:"tier"`
    TrialEndAt    *time.Time `json:"trial_end_at,omitempty"`
    SubscriptionID string    `json:"subscription_id,omitempty"`
    LastSearchAt  *time.Time `json:"last_search_at,omitempty"`
    SearchCount24h int       `json:"search_count_24h"`
    CreatedAt     time.Time  `json:"created_at"`
    UpdatedAt     time.Time  `json:"updated_at"`
}

type CheckResult struct {
    Allowed       bool     `json:"allowed"`
    Tier          Tier     `json:"tier"`
    RemainingSearches int  `json:"remaining_searches,omitempty"`
    Feature       string   `json:"feature"`
    Error         string   `json:"error,omitempty"`
}

type Feature string

const (
    FeatureUnlimitedSearches Feature = "unlimited_searches"
    FeatureMultiItemSearch   Feature = "multi_item_search"
    FeatureAdvancedAnalytics Feature = "advanced_analytics"
    FeatureExportPDF         Feature = "export_pdf"
)

type SearchRequest struct {
    UserID    string    `json:"user_id"`
    ItemCount int       `json:"item_count"`
    Timestamp time.Time `json:"timestamp"`
}

type UsageRecord struct {
    UserID       string    `json:"user_id"`
    Feature      Feature   `json:"feature"`
    UsedAt       time.Time `json:"used_at"`
    SearchCount  int       `json:"search_count"`
}

type EntitlementManager interface {
    CheckEntitlement(ctx context.Context, userID string, feature Feature) (*CheckResult, error)
    CheckSearchLimit(ctx context.Context, userID string, itemCount int) (*CheckResult, error)
    RecordSearchUsage(ctx context.Context, userID string) error
    UpgradeToPaid(ctx context.Context, userID, subscriptionID string) error
    ActivateTrial(ctx context.Context, userID string, durationDays int) error
    DowngradeToFree(ctx context.Context, userID string) error
    SyncEntitlement(ctx context.Context, userID string) (*Entitlement, error)
}
```

### 2. Logic & Algorithms (Step-by-Step)

**CheckEntitlement(userID, feature) Algorithm:**

```
1. Fetch entitlement record from Redis cache
2. If not in cache, fetch from PostgreSQL via EntitlementRepository
3. Cache the entitlement with TTL of 5 minutes
4. If user has no entitlement record, return default free tier
5. If feature == "unlimited_searches" or "multi_item_search":
   a. If tier == "paid" OR (tier == "trial" AND trial not expired):
      - Return allowed: true, remaining_searches: unlimited
   b. If tier == "free":
      - Return allowed: false, remaining_searches: 0
6. For all other features (analytics, export):
   a. Only allow if tier == "paid" OR (tier == "trial" AND trial not expired)
7. Return CheckResult with allowed flag and tier info
```

**CheckSearchLimit(userID, itemCount) Algorithm:**

```
1. Fetch entitlement record from Redis cache
2. If not in cache, fetch from PostgreSQL
3. Check current tier:
   a. If tier == "paid":
      - Return allowed: true, no search limits
   b. If tier == "trial":
      - Check if trial_end_at > now()
      - If expired, call DowngradeToFree(), treat as free tier
      - If not expired, return allowed: true
   c. If tier == "free":
      - Get search count for last 24 hours from Redis (key: "search_count:{user_id}")
      - If search_count >= 3:
        - Return allowed: false, remaining_searches: 0
      - If itemCount > 1:
        - Return allowed: false, remaining_searches: remaining
      - Return allowed: true, remaining_searches: (3 - search_count)
```

**RecordSearchUsage(userID) Algorithm:**

```
1. Generate Redis key: "search_count:{user_id}"
2. Use INCR command to atomically increment search count
3. Set TTL of 24 hours on the key
4. Update last_search_at timestamp in entitlement record
5. Persist search count to PostgreSQL asynchronously via job queue
6. Return success
```

**ActivateTrial(userID, durationDays) Algorithm:**

```
1. Check if user already has active trial (prevent double activation)
2. Calculate trial_end_at = now() + durationDays
3. Create/update entitlement record with:
   - tier: "trial"
   - trial_end_at: calculated timestamp
4. Set Redis cache with TTL matching trial duration
5. Return success
```

**DowngradeToFree(userID) Algorithm:**

```
1. Fetch current entitlement record
2. If tier is already "free", return early
3. Update entitlement record:
   - tier: "free"
   - trial_end_at: nil
   - subscription_id: nil (preserve for reactivation)
4. Clear Redis cache for this user
5. Log downgrade event for analytics
6. Return success
```

**SyncEntitlement(userID) Algorithm:**

```
1. Fetch latest subscription status from Stripe API
2. Compare with local entitlement record
3. If Stripe status == "active" AND local tier != "paid":
   - Call UpgradeToPaid()
4. If Stripe status == "canceled" AND local tier == "paid":
   - Check if within grace period (e.g., 7 days)
   - If grace period expired: Call DowngradeToFree()
   - If within grace period: Update subscription_end_at, keep paid tier
5. Return synchronized entitlement
```

### 3. State Management & Error Handling

**Error States:**

| Error Condition | Error Code | Handling Strategy |
|----------------|------------|-------------------|
| Redis connection timeout | `ERR_REDIS_TIMEOUT` | Fallback to PostgreSQL, log alert, retry with exponential backoff |
| PostgreSQL connection failure | `ERR_DB_UNAVAILABLE` | Return cached data, queue sync job, alert on extended outage |
| Stripe API timeout | `ERR_STRIPE_TIMEOUT` | Retry 3 times with backoff, use cached entitlement as fallback |
| Stripe API rate limit | `ERR_STRIPE_RATE_LIMITED` | Queue sync job for later, use cached entitlement |
| Entitlement record not found | `ERR_NOT_FOUND` | Create default free tier entitlement, return default access |
| Invalid user ID | `ERR_INVALID_INPUT` | Return error immediately, reject request |
| Trial already active | `ERR_TRIAL_ACTIVE` | Return success, no-op (prevent double activation) |
| Webhook conflict | `ERR_CONCURRENT_MODIFICATION` | Retry transaction with fresh read |

**State Transitions:**

```
Initial State: No entitlement record
    |
    v
Default Free Tier (no trial, no subscription)
    |
    +---> ActivateTrial() ---> Trial State (7-day trial)
    |                           |
    |                           v
    |                   Expiry Check: trial_end_at > now?
    |                           |
    |               Yes ----+   No
    |                       |   |
    |                       v   v
    |               Paid Tier <--+ DowngradeToFree()
    |                   |         |
    |                   v         |
    |           Stripe Webhook    |
    |           (payment_succeeded)
    |                   |         |
    |                   v         |
    |           Subscription Active
    |                   |
    |                   v
    |           Stripe Webhook
    |           (subscription_deleted)
    |                   |
    |                   v
    |           Grace Period (7 days)
    |                   |
    |                   v
    +-------------------+ DowngradeToFree()

Cache State Machine:
    |
    v
Cache MISS -> Fetch DB -> Cache SET (5 min TTL)
    |
    |
    v
Cache HIT -> Return Data
    |
    |
    v
Write Operation -> Invalidate Cache -> Update DB -> Recache
```

**Recovery Procedures:**

1. **Cache Invalidation Storm:**
   - Use read-through cache pattern
   - Implement cache warming for active users
   - Set conservative TTL (5 minutes) on entitlement cache

2. **Stale Cache Data:**
   - Background reconciliation job runs hourly
   - Compares local entitlements with Stripe subscription status
   - Fixes discrepancies within 1 hour maximum

3. **Partial Payment Processing:**
   - Webhook handler wraps DB transaction
   - Dead-letter queue for failed transactions
   - Reconciliation job catches and fixes within 1 hour

### 4. Component Interfaces

```go
package entitlement

import (
    "context"
    "time"
)

// EntitlementRepository defines database operations for entitlements
type EntitlementRepository interface {
    GetByUserID(ctx context.Context, userID string) (*Entitlement, error)
    Create(ctx context.Context, entitlement *Entitlement) error
    Update(ctx context.Context, entitlement *Entitlement) error
    UpdateSearchCount(ctx context.Context, userID string, count int) error
    IncrementSearchCount(ctx context.Context, userID string) error
}

// EntitlementCache defines caching operations for entitlements
type EntitlementCache interface {
    Get(ctx context.Context, userID string) (*Entitlement, error)
    Set(ctx context.Context, entitlement *Entitlement, ttl time.Duration) error
    Delete(ctx context.Context, userID string) error
    GetSearchCount(ctx context.Context, userID string) (int, error)
    IncrementSearchCount(ctx context.Context, userID string) (int, error)
    SetSearchCount(ctx context.Context, userID string, count int, ttl time.Duration) error
}

// StripeClient defines Stripe API operations
type StripeClient interface {
    GetSubscription(subscriptionID string) (*StripeSubscription, error)
    GetCustomerSubscriptions(customerID string) ([]*StripeSubscription, error)
}

// EntitlementManagerImpl is the concrete implementation
type EntitlementManagerImpl struct {
    repo      EntitlementRepository
    cache     EntitlementCache
    stripe    StripeClient
    logger    *log.Logger
    queue     JobQueue
}

// NewEntitlementManager creates a new EntitlementManager instance
func NewEntitlementManager(
    repo EntitlementRepository,
    cache EntitlementCache,
    stripe StripeClient,
    logger *log.Logger,
    queue JobQueue,
) *EntitlementManagerImpl {
    return &EntitlementManagerImpl{
        repo:   repo,
        cache:  cache,
        stripe: stripe,
        logger: logger,
        queue:  queue,
    }
}

// CheckEntitlement verifies if user can access a specific feature
func (m *EntitlementManagerImpl) CheckEntitlement(ctx context.Context, userID string, feature Feature) (*CheckResult, error) {
    entitlement, err := m.getEntitlement(ctx, userID)
    if err != nil {
        return nil, err
    }

    result := &CheckResult{
        Tier:    entitlement.Tier,
        Feature: string(feature),
    }

    switch feature {
    case FeatureUnlimitedSearches, FeatureMultiItemSearch:
        result.Allowed = m.canAccessPaidFeature(entitlement)
        if !result.Allowed {
            remaining, _ := m.getRemainingSearches(ctx, userID)
            result.RemainingSearches = remaining
        }
    default:
        result.Allowed = m.canAccessPaidFeature(entitlement)
    }

    return result, nil
}

// CheckSearchLimit verifies if user can perform a search with given item count
func (m *EntitlementManagerImpl) CheckSearchLimit(ctx context.Context, userID string, itemCount int) (*CheckResult, error) {
    entitlement, err := m.getEntitlement(ctx, userID)
    if err != nil {
        return nil, err
    }

    result := &CheckResult{
        Tier:   entitlement.Tier,
        Feature: "search",
    }

    if entitlement.Tier == TierPaid {
        result.Allowed = true
        return result, nil
    }

    if entitlement.Tier == TierTrial {
        if m.isTrialExpired(entitlement) {
            _ = m.DowngradeToFree(ctx, userID)
            return m.CheckSearchLimit(ctx, userID, itemCount)
        }
        result.Allowed = true
        result.RemainingSearches = -1 // unlimited
        return result, nil
    }

    searchCount, err := m.cache.GetSearchCount(ctx, userID)
    if err != nil {
        searchCount = entitlement.SearchCount24h
    }

    remaining := 3 - searchCount
    if searchCount >= 3 {
        result.Allowed = false
        result.RemainingSearches = 0
        return result, nil
    }

    if itemCount > 1 {
        result.Allowed = false
        result.RemainingSearches = remaining
        return result, nil
    }

    result.Allowed = true
    result.RemainingSearches = remaining
    return result, nil
}

// RecordSearchUsage increments the search count for a user
func (m *EntitlementManagerImpl) RecordSearchUsage(ctx context.Context, userID string) error {
    count, err := m.cache.IncrementSearchCount(ctx, userID)
    if err != nil {
        m.logger.Warn("cache increment failed, falling back to db", "error", err)
        err = m.repo.IncrementSearchCount(ctx, userID)
        if err != nil {
            return err
        }
    }

    if count == 1 {
        m.cache.SetSearchCount(ctx, userID, count, 24*time.Hour)
    }

    entitlement, err := m.getEntitlement(ctx, userID)
    if err != nil {
        return err
    }

    now := time.Now()
    entitlement.LastSearchAt = &now
    entitlement.SearchCount24h = count

    go func() {
        if err := m.repo.UpdateSearchCount(ctx, userID, count); err != nil {
            m.logger.Error("failed to persist search count", "error", err)
        }
    }()

    return nil
}

// UpgradeToPaid transitions a user to paid tier
func (m *EntitlementManagerImpl) UpgradeToPaid(ctx context.Context, userID, subscriptionID string) error {
    entitlement, err := m.getEntitlement(ctx, userID)
    if err != nil {
        return err
    }

    entitlement.Tier = TierPaid
    entitlement.SubscriptionID = subscriptionID
    entitlement.UpdatedAt = time.Now()

    if err := m.repo.Update(ctx, entitlement); err != nil {
        return err
    }

    m.cache.Delete(ctx, userID)

    m.logger.Info("user upgraded to paid", "user_id", userID, "subscription_id", subscriptionID)
    return nil
}

// ActivateTrial starts a trial period for a user
func (m *EntitlementManagerImpl) ActivateTrial(ctx context.Context, userID string, durationDays int) error {
    existing, _ := m.getEntitlement(ctx, userID)
    if existing != nil && existing.Tier == TierTrial && !m.isTrialExpired(existing) {
        return nil // Trial already active
    }

    trialEnd := time.Now().AddDate(0, 0, durationDays)

    entitlement := &Entitlement{
        UserID:        userID,
        Tier:          TierTrial,
        TrialEndAt:    &trialEnd,
        SearchCount24h: 0,
        CreatedAt:     time.Now(),
        UpdatedAt:     time.Now(),
    }

    if existing == nil {
        if err := m.repo.Create(ctx, entitlement); err != nil {
            return err
        }
    } else {
        entitlement.CreatedAt = existing.CreatedAt
        entitlement.SubscriptionID = existing.SubscriptionID
        if err := m.repo.Update(ctx, entitlement); err != nil {
            return err
        }
    }

    m.cache.Delete(ctx, userID)

    m.logger.Info("trial activated", "user_id", userID, "duration_days", durationDays)
    return nil
}

// DowngradeToFree transitions a user to free tier
func (m *EntitlementManagerImpl) DowngradeToFree(ctx context.Context, userID string) error {
    entitlement, err := m.getEntitlement(ctx, userID)
    if err != nil {
        return err
    }

    if entitlement.Tier == TierFree {
        return nil
    }

    previousTier := entitlement.Tier
    entitlement.Tier = TierFree
    entitlement.TrialEndAt = nil
    entitlement.UpdatedAt = time.Now()

    if err := m.repo.Update(ctx, entitlement); err != nil {
        return err
    }

    m.cache.Delete(ctx, userID)

    m.logger.Info("user downgraded to free", "user_id", userID, "previous_tier", previousTier)
    return nil
}

// SyncEntitlement reconciles local entitlement with Stripe status
func (m *EntitlementManagerImpl) SyncEntitlement(ctx context.Context, userID string) (*Entitlement, error) {
    entitlement, err := m.getEntitlement(ctx, userID)
    if err != nil {
        return nil, err
    }

    if entitlement.SubscriptionID == "" {
        return entitlement, nil
    }

    stripeSub, err := m.stripe.GetSubscription(entitlement.SubscriptionID)
    if err != nil {
        m.logger.Warn("failed to fetch stripe subscription", "error", err)
        return entitlement, nil
    }

    if stripeSub.Status == "active" && entitlement.Tier != TierPaid {
        if err := m.UpgradeToPaid(ctx, userID, entitlement.SubscriptionID); err != nil {
            return nil, err
        }
        entitlement.Tier = TierPaid
    }

    if stripeSub.Status == "canceled" && entitlement.Tier == TierPaid {
        if time.Now().After(stripeSub.CancelAt) {
            _ = m.DowngradeToFree(ctx, userID)
            entitlement.Tier = TierFree
        }
    }

    return entitlement, nil
}

// Helper methods

func (m *EntitlementManagerImpl) getEntitlement(ctx context.Context, userID string) (*Entitlement, error) {
    entitlement, err := m.cache.Get(ctx, userID)
    if err == nil {
        return entitlement, nil
    }

    entitlement, err = m.repo.GetByUserID(ctx, userID)
    if err != nil {
        if err == sql.ErrNoRows {
            entitlement = &Entitlement{
                UserID:        userID,
                Tier:          TierFree,
                SearchCount24h: 0,
                CreatedAt:     time.Now(),
                UpdatedAt:     time.Now(),
            }
            if err := m.repo.Create(ctx, entitlement); err != nil {
                return nil, err
            }
            return entitlement, nil
        }
        return nil, err
    }

    m.cache.Set(ctx, entitlement, 5*time.Minute)
    return entitlement, nil
}

func (m *EntitlementManagerImpl) canAccessPaidFeature(entitlement *Entitlement) bool {
    if entitlement.Tier == TierPaid {
        return true
    }
    if entitlement.Tier == TierTrial && !m.isTrialExpired(entitlement) {
        return true
    }
    return false
}

func (m *EntitlementManagerImpl) isTrialExpired(entitlement *Entitlement) bool {
    if entitlement.TrialEndAt == nil {
        return true
    }
    return time.Now().After(*entitlement.TrialEndAt)
}

func (m *EntitlementManagerImpl) getRemainingSearches(ctx context.Context, userID string) (int, error) {
    count, err := m.cache.GetSearchCount(ctx, userID)
    if err != nil {
        entitlement, err := m.getEntitlement(ctx, userID)
        if err != nil {
            return 0, err
        }
        count = entitlement.SearchCount24h
    }
    return max(0, 3-count), nil
}
```
