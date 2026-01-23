## FILE: UsageLimiter.md
**Traceability:** ARCH-007

### 1. Data Structures & Types

```go
package subscription

import (
	"time"
)

// Tier represents the subscription tier level
type Tier string

const (
	TierFree  Tier = "free"
	TierPaid  Tier = "paid"
	TierTrial Tier = "trial"
)

// UsageLimit defines the usage constraints for a tier
type UsageLimit struct {
	MaxSearchesPerDay int
	MaxItemsPerSearch int
	FeaturesEnabled   []string
}

// Entitlement represents a user's subscription status and usage tracking
type Entitlement struct {
	UserID         string
	Tier           Tier
	TrialExpiresAt *time.Time
	SearchCount    int
	LastSearchDate time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// CheckResult contains the result of an entitlement check
type CheckResult struct {
	Allowed          bool
	Reason           DenialReason
	RemainingSearches int
	ResetAt          time.Time
	CurrentTier      Tier
}

// DenialReason specifies why a request was denied
type DenialReason string

const (
	DenialNone          DenialReason = ""
	DenialQuotaExceeded DenialReason = "quota_exceeded"
	DenialTierRestricted DenialReason = "tier_restricted"
	DenialTrialExpired  DenialReason = "trial_expired"
)

// SearchRequest represents a search operation request
type SearchRequest struct {
	UserID  string
	ItemIDs []string // Single item for free tier, multiple for paid/trial
}

// TierLimits provides the usage limits for each tier
var TierLimits = map[Tier]UsageLimit{
	TierFree: {
		MaxSearchesPerDay: 3,
		MaxItemsPerSearch: 1,
		FeaturesEnabled: []string{
			"basic_search",
		},
	},
	TierPaid: {
		MaxSearchesPerDay: -1, // Unlimited
		MaxItemsPerSearch: -1, // Unlimited
		FeaturesEnabled: []string{
			"basic_search",
			"multi_item_search",
			"advanced_filters",
			"export_results",
		},
	},
	TierTrial: {
		MaxSearchesPerDay: -1, // Unlimited
		MaxItemsPerSearch: -1, // Unlimited
		FeaturesEnabled: []string{
			"basic_search",
			"multi_item_search",
			"advanced_filters",
			"export_results",
		},
	},
}

// Redis key patterns for usage tracking
const (
	usageKeyPattern    = "user:%s:usage"     // Hash: date -> count
	entitlementKey     = "entitlement:%s"    // Hash: user entitlement data
	trialExpiryKey     = "trial:%s:expiry"   // String: trial expiration timestamp
)
```

### 2. Logic & Algorithms (Step-by-Step)

**Algorithm: CheckAndIncrementUsage**

```
1. INPUT: userID (string), itemCount (int)
2. Fetch entitlement for userID from Redis cache or PostgreSQL
   a. IF cache miss:
      i. Query entitlement from ARCH-005 (Data Repository)
      ii. Cache result in Redis with 5-minute TTL
3. DETERMINE user tier:
   a. IF trial_expires_at exists AND trial_expires_at > NOW():
      i. Tier = TierTrial
   b. ELSE IF subscription_status = "active":
      i. Tier = TierPaid
   c. ELSE:
      i. Tier = TierFree
4. GET current date in UTC (YYYY-MM-DD format)
5. FETCH current usage count for userID from Redis hash
   a. Key: format("user:%s:usage", userID)
   b. Field: current date string
6. RETRIEVE tier limits for determined tier
7. VALIDATE item count against MaxItemsPerSearch:
   a. IF itemCount > MaxItemsPerSearch AND MaxItemsPerSearch != -1:
      i. RETURN CheckResult{Allowed: false, Reason: DenialTierRestricted, CurrentTier: tier}
8. IF tier is TierFree:
   a. IF current usage count >= MaxSearchesPerDay:
      i. CALCULATE reset time: start of next UTC day
      ii. RETURN CheckResult{Allowed: false, Reason: DenialQuotaExceeded, ResetAt: resetTime, CurrentTier: tier}
   b. INCREMENT usage count in Redis (atomic HINCRBY)
      i. SET expiry to 48 hours (account for timezone edge cases)
   c. RETURN CheckResult{Allowed: true, RemainingSearches: MaxSearchesPerDay - newCount, CurrentTier: tier}
9. IF tier is TierPaid OR TierTrial:
   a. RETURN CheckResult{Allowed: true, CurrentTier: tier}
10. OUTPUT: CheckResult
```

**Algorithm: GetEntitlement**

```
1. INPUT: userID (string)
2. CHECK Redis cache first:
   a. Key: format("entitlement:%s", userID)
   b. IF exists AND not expired:
      i. RETURN cached entitlement
3. FALLBACK to PostgreSQL via ARCH-005:
   a. Query entitlements table WHERE user_id = userID
   b. IF no record found:
      i. Create default free tier entitlement
      ii. INSERT into database
   c. Serialize to Entitlement struct
4. CACHE result in Redis:
   a. Key: format("entitlement:%s", userID)
   b. TTL: 5 minutes
5. OUTPUT: Entitlement
```

**Algorithm: ResetDailyUsage (Cron Job - runs at midnight UTC)**

```
1. For each user with cached usage data older than 24 hours:
   a. DELETE old usage hash fields for dates before yesterday
2. CLEANUP orphan usage keys (keys with no fields)
3. OUTPUT: void
```

### 3. State Management & Error Handling

**State Transitions:**

| Current State | Event | Next State | Action |
|---------------|-------|------------|--------|
| None | User registered | Free tier | Initialize entitlement record |
| Free tier | Payment success | Paid tier | Update tier, clear usage limits |
| Free tier | Trial activated | Trial tier | Set trial_expiry = now + 7 days |
| Trial tier | Trial expires | Free tier | Clear trial_expiry, reset tier |
| Paid tier | Subscription cancelled | Free tier | Downgrade at period end |
| Any | Search request | Check result | Validate and increment |

**Error States:**

| Error Condition | Handling Strategy | Retryable |
|-----------------|-------------------|-----------|
| Redis connection timeout | Fail open (allow request), log error, async retry | Yes |
| Redis INCRBY fails | Return 503, do not process search | Yes |
| PostgreSQL entitlement fetch fails | Return 500, do not process search | Yes |
| Cache stampede (many requests for new cache) | Use singleflight pattern | N/A |
| Invalid user ID | Return 400 Bad Request | No |
| Trial expiry in past (stale cache) | Force cache refresh, deny trial access | No |

**Failure Modes:**

1. **Redis Unavailable:**
   - Behavior: Fail open for paid/trial users, fail closed for free tier (quota check via PostgreSQL fallback)
   - Recovery: Log to GCP Cloud Monitoring, trigger alerts

2. **PostgreSQL Unavailable:**
   - Behavior: Serve cached entitlement if available; for new users, deny access
   - Recovery: Queue entitlement creation for retry

3. **Clock Skew:**
   - Behavior: Use UTC exclusively for all time calculations
   - Prevention: NTP sync on servers, validate client timestamps

**Rate Limiting Headers:**

```
X-RateLimit-Limit: 3           // Max searches for free tier
X-RateLimit-Remaining: 2       // Remaining searches
X-RateLimit-Reset: 1737580800  // Unix timestamp of reset
X-Tier: free                   // Current tier
```

### 4. Component Interfaces

```go
package subscription

import (
	"context"
	"time"
)

// UsageLimiter handles entitlement checks and usage tracking
type UsageLimiter interface {
	// CheckUsage validates and increments usage for a search request
	CheckUsage(ctx context.Context, userID string, itemCount int) (*CheckResult, error)

	// GetEntitlement retrieves the current entitlement for a user
	GetEntitlement(ctx context.Context, userID string) (*Entitlement, error)

	// SetTier updates a user's subscription tier
	SetTier(ctx context.Context, userID string, tier Tier) error

	// ActivateTrial activates a trial period for a user
	ActivateTrial(ctx context.Context, userID string, duration time.Duration) error

	// ExpireTrial immediately expires a user's trial
	ExpireTrial(ctx context.Context, userID string) error

	// ResetUsage resets daily usage counter for a user
	ResetUsage(ctx context.Context, userID string) error

	// GetUsageStats returns usage statistics for monitoring
	GetUsageStats(ctx context.Context, userID string) (*UsageStats, error)
}

// UsageStats contains usage telemetry
type UsageStats struct {
	UserID           string
	CurrentTier      Tier
	SearchesToday    int
	DailyLimit       int
	TrialDaysRemaining int
	LastSearchAt     *time.Time
}
```

**Implementation Signatures:**

```go
type usageLimiter struct {
	redis *redis.Client
	db    *sql.DB
	repo  repository.EntitlementRepository
}

// NewUsageLimiter creates a new UsageLimiter instance
func NewUsageLimiter(redis *redis.Client, db *sql.DB, repo repository.EntitlementRepository) UsageLimiter {
	return &usageLimiter{
		redis: redis,
		db:    db,
		repo:  repo,
	}
}

// CheckUsage implements UsageLimiter.CheckUsage
func (u *usageLimiter) CheckUsage(ctx context.Context, userID string, itemCount int) (*CheckResult, error) {
	entitlement, err := u.GetEntitlement(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get entitlement: %w", err)
	}

	tier := u.determineTier(entitlement)
	limits := TierLimits[tier]

	if itemCount > limits.MaxItemsPerSearch && limits.MaxItemsPerSearch != -1 {
		return &CheckResult{
			Allowed:  false,
			Reason:   DenialTierRestricted,
			CurrentTier: tier,
		}, nil
	}

	if tier == TierFree {
		return u.checkFreeTierQuota(ctx, userID, limits)
	}

	return &CheckResult{
		Allowed:     true,
		Reason:      DenialNone,
		CurrentTier: tier,
	}, nil
}

// determineTier returns the effective tier based on entitlement state
func (u *usageLimiter) determineTier(ent *Entitlement) Tier {
	if ent.TrialExpiresAt != nil && ent.TrialExpiresAt.After(time.Now()) {
		return TierTrial
	}
	return ent.Tier
}

// checkFreeTierQuota checks and increments quota for free tier users
func (u *usageLimiter) checkFreeTierQuota(ctx context.Context, userID string, limits UsageLimit) (*CheckResult, error) {
	today := time.Now().UTC().Format("2006-01-02")
	key := fmt.Sprintf(usageKeyPattern, userID)

	count, err := u.redis.HIncrBy(ctx, key, today, 1).Result()
	if err != nil {
		u.redis.Set(ctx, fmt.Sprintf("entitlement:%s", userID), "", time.Minute*5) // Invalidate cache
		return nil, fmt.Errorf("failed to increment usage: %w", err)
	}

	u.redis.Expire(ctx, key, time.Hour*48)

	remaining := limits.MaxSearchesPerDay - int(count)
	if remaining < 0 {
		remaining = 0
	}

	if count > int64(limits.MaxSearchesPerDay) {
		resetTime := u.getNextMidnightUTC()
		return &CheckResult{
			Allowed:           false,
			Reason:            DenialQuotaExceeded,
			RemainingSearches: remaining,
			ResetAt:           resetTime,
			CurrentTier:       TierFree,
		}, nil
	}

	return &CheckResult{
		Allowed:           true,
		Reason:            DenialNone,
		RemainingSearches: remaining,
		CurrentTier:       TierFree,
	}, nil
}

// getNextMidnightUTC returns the start of the next UTC day
func (u *usageLimiter) getNextMidnightUTC() time.Time {
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
}
```

**HTTP Handler Integration:**

```go
package handlers

type SearchHandler struct {
	limiter subscription.UsageLimiter
}

func (h *SearchHandler) Search(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	var req SearchRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	result, err := h.limiter.CheckUsage(c.Context(), userID, len(req.ItemIDs))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "service unavailable"})
	}

	if !result.Allowed {
		c.Set("X-RateLimit-Limit", "3")
		c.Set("X-RateLimit-Remaining", fmt.Sprintf("%d", result.RemainingSearches))
		c.Set("X-RateLimit-Reset", fmt.Sprintf("%d", result.ResetAt.Unix()))
		c.Set("X-Tier", string(result.CurrentTier))

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error":        "quota exceeded",
			"reason":       result.Reason,
			"resets_at":    result.ResetAt,
			"tier":         result.CurrentTier,
		})
	}

	c.Set("X-Tier", string(result.CurrentTier))
	return c.Next()
}
```
