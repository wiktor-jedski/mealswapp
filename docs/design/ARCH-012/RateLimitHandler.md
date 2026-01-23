# FILE: RateLimitHandler.md
**Traceability:** ARCH-012

---

## 1. Data Structures & Types

```go
package rate

import (
	"context"
	"time"
)

// RateLimitConfig holds configuration for rate limiting per external API
type RateLimitConfig struct {
	API               string        // API identifier (e.g., "usda", "openfoodfacts")
	RequestsPerSecond float64       // Rate limit: requests per second
	RequestsPerMinute float64       // Rate limit: requests per minute
	RequestsPerDay    int           // Rate limit: requests per day
	BurstSize         int           // Maximum burst capacity
	BackoffMultiplier float64       // Multiplier for exponential backoff
	MaxBackoff        time.Duration // Maximum backoff duration
}

// APIResponse represents a response from an external API
type APIResponse[T any] struct {
	Data      T
	RateLimit RateLimitStatus
	Metadata  APIResponseMetadata
}

// RateLimitStatus contains rate limit information from API response headers
type RateLimitStatus struct {
	Remaining     int64     // Remaining requests in current window
	ResetAt       time.Time // Time when the limit resets
	Limit         int64     // Total limit for the window
	RetryAfter    time.Duration // Recommended wait before retry
	IsRateLimited bool      // Whether request was rate limited
}

// TokenBucket represents a token bucket for rate limiting
type TokenBucket struct {
	Capacity        int           // Maximum tokens in bucket
	Tokens          float64       // Current tokens available
	RefillRate      float64       // Tokens added per second
	LastRefillTime  time.Time     // Last time tokens were refilled
	mu              sync.Mutex    // Mutex for thread safety
}

// RateLimitHandler manages rate limiting for multiple external APIs
type RateLimitHandler struct {
	buckets     map[string]*TokenBucket // Token buckets per API
	configs     map[string]RateLimitConfig
	redisClient *redis.Client
	ctx         context.Context
	logger      *log.Logger
}

// RateLimitResult contains the result of a rate-limited operation
type RateLimitResult[T any] struct {
	Success     bool
	Data        T
	WaitTime    time.Duration
	RetryCount  int
	Error       error
	RateLimit   RateLimitStatus
}

// WindowCounter tracks request counts within a time window
type WindowCounter struct {
	WindowSize  time.Duration
	Count       int
	WindowStart time.Time
}
```

---

## 2. Logic & Algorithms (Step-by-Step)

### 2.1 Token Bucket Algorithm

```
FUNCTION AcquireToken(apiID string, requiredTokens int) -> (acquired bool, waitTime time.Duration)
1. bucket ← buckets[apiID]
2. IF bucket does not exist THEN
3.     bucket ← CreateTokenBucket(configs[apiID])
4.     buckets[apiID] ← bucket
5. END IF

6. bucket.RefillTokens()
7. tokensAvailable ← bucket.Tokens

8. IF tokensAvailable >= requiredTokens THEN
9.     bucket.Tokens ← tokensAvailable - requiredTokens
10.    RETURN (true, 0)
11. END IF

12. waitTime ← CalculateWaitTime(bucket, requiredTokens)
13. RETURN (false, waitTime)
END FUNCTION
```

### 2.2 Token Refill Process

```
FUNCTION RefillTokens()
1. currentTime ← time.Now()
2. elapsed ← currentTime - LastRefillTime
3. tokensToAdd ← elapsed.Seconds() * RefillRate
4. Tokens ← min(Capacity, Tokens + tokensToAdd)
5. LastRefillTime ← currentTime
END FUNCTION
```

### 2.3 ExecuteWithRateLimit (Main Algorithm)

```
FUNCTION ExecuteWithRateLimit[T any](
    apiID string,
    operation func() APIResponse[T],
    maxRetries int
) RateLimitResult[T]

1. result ← RateLimitResult[T]{RetryCount: 0}
2. config ← configs[apiID]

3. FOR retryCount FROM 0 TO maxRetries DO
4.     acquired, waitTime ← AcquireToken(apiID, 1)

5.     IF acquired THEN
6.         response ← operation()
7.         UpdateRateLimitStatus(apiID, response.RateLimit)
8.         result.Success ← true
9.         result.Data ← response.Data
10.        result.RateLimit ← response.RateLimit
11.        RETURN result
12.    END IF

13.    IF waitTime > config.MaxBackoff THEN
14.        result.Error ← RateLimitExceededError{maxRetries, apiID}
15.        RETURN result
16.    END IF

17.    Sleep(waitTime)
18.    result.RetryCount ← retryCount + 1
19. END FOR

20. result.Error ← MaxRetriesExceededError{maxRetries, apiID}
21. RETURN result
END FUNCTION
```

### 2.4 Exponential Backoff Calculation

```
FUNCTION CalculateWaitTime(bucket TokenBucket, requiredTokens int) -> time.Duration
1. tokensNeeded ← requiredTokens - bucket.Tokens
2. refillTimeNeeded ← tokensNeeded / bucket.RefillRate
3. backoff ← time.Duration(refillTimeNeeded * float64(time.Second))
4. backoff ← min(backoff, config.MaxBackoff)
5. RETURN backoff
END FUNCTION
```

### 2.5 Daily Limit Enforcement

```
FUNCTION CheckDailyLimit(apiID string) -> (allowed bool, remaining int)
1. key ← BuildRedisKey("ratelimit:daily", apiID, GetCurrentDate())
2. currentCount ← redisClient.Get(ctx, key)

3. dailyLimit ← configs[apiID].RequestsPerDay

4. IF currentCount >= dailyLimit THEN
5.     RETURN (false, 0)
6. END IF

7. remaining ← dailyLimit - currentCount
8. RETURN (true, remaining)
END FUNCTION
```

### 2.6 Rate Limit Handler Initialization

```
FUNCTION NewRateLimitHandler(
    redisClient *redis.Client,
    logger *log.Logger
) RateLimitHandler

1. configs ← map[string]RateLimitConfig{
2.     "usda": RateLimitConfig{
3.         API:               "usda",
4.         RequestsPerSecond: 10,
5.         RequestsPerMinute: 500,
6.         RequestsPerDay:    10000,
7.         BurstSize:         20,
8.         BackoffMultiplier: 2.0,
9.         MaxBackoff:        60 * time.Second,
10.    },
11.    "openfoodfacts": RateLimitConfig{
12.        API:               "openfoodfacts",
13.        RequestsPerSecond: 5,
14.         RequestsPerMinute: 300,
15.         RequestsPerDay:    5000,
16.         BurstSize:         10,
17.         BackoffMultiplier: 2.0,
18.         MaxBackoff:        60 * time.Second,
19.    },
20.}

21. buckets ← make(map[string]*TokenBucket)

22. RETURN RateLimitHandler{
23.     configs:     configs,
24.     buckets:     buckets,
25.     redisClient: redisClient,
26.     logger:      logger,
27. }
END FUNCTION
```

---

## 3. State Management & Error Handling

### 3.1 Error States

| Error Condition | Type | Recovery Strategy |
| :--- | :--- | :--- |
| `RateLimitExceededError` | Transient | Exponential backoff with retry |
| `MaxRetriesExceededError` | Permanent | Return error to caller, log incident |
| `RedisConnectionError` | Transient | Circuit breaker, use local fallback |
| `DailyLimitExceededError` | Permanent | Queue request for next day |
| `APIResponseTimeoutError` | Transient | Retry with shorter timeout |
| `InvalidAPIResponseError` | Permanent | Log error, return empty result |

### 3.2 State Transitions

```
Initial State: READY
  ↓ (AcquireToken succeeds)
Operation State: EXECUTING
  ↓ (Request completes)
Terminal States:
  - SUCCESS (operation completed)
  - RATE_LIMITED (wait and retry)
  - FAILED (error, no retry remaining)
```

### 3.3 Circuit Breaker Integration

```
FUNCTION ExecuteWithCircuitBreaker[T any](
    apiID string,
    operation func() APIResponse[T]
) RateLimitResult[T]

1. IF circuitBreaker.IsOpen(apiID) THEN
2.     RETURN RateLimitResult[T]{
3.         Error: CircuitOpenError{apiID},
4.         WaitTime: circuitBreaker.GetRetryAfter(apiID),
5.     }
6. END IF

7. result ← ExecuteWithRateLimit(apiID, operation, 3)

8. IF result.Error IS RateLimitExceededError THEN
9.     circuitBreaker.RecordFailure(apiID)
10. ELSE IF result.Success THEN
11.     circuitBreaker.RecordSuccess(apiID)
12. END IF

13. RETURN result
END FUNCTION
```

### 3.4 Redis Sync for Distributed Rate Limiting

```
FUNCTION SyncWithRedis(apiID string)
1. FOR each bucket IN buckets DO
2.     key ← BuildRedisKey("ratelimit:tokens", apiID)
3.     redisClient.Set(ctx, key, bucket.Tokens, 24*time.Hour)
4.     dailyKey ← BuildRedisKey("ratelimit:daily", apiID, GetCurrentDate())
5.     redisClient.Incr(ctx, dailyKey)
6.     redisClient.Expire(ctx, dailyKey, 24*time.Hour)
7. END FOR
END FUNCTION
```

### 3.5 Fallback Mechanism

```
FUNCTION GetRateLimitHandler() RateLimitHandler
1. IF redisClient IS connected THEN
2.     RETURN NewRateLimitHandler(redisClient, logger)
3. END IF

4. logger.Warn("Redis unavailable, using local-only rate limiting")
5. RETURN RateLimitHandler{
6.     configs: localConfigs,
7.     buckets: localBuckets,
8.     redisClient: nil,
9.     logger: logger,
10.}
END FUNCTION
```

---

## 4. Component Interfaces

### 4.1 Public Interface

```go
// RateLimitHandler provides rate limiting for external API calls
type RateLimitHandler interface {
	// ExecuteWithRateLimit executes an API operation with rate limiting
	ExecuteWithRateLimit[T any](
		ctx context.Context,
		apiID string,
		operation func(ctx context.Context) (*APIResponse[T], error),
	) *RateLimitResult[T]

	// AcquireToken attempts to acquire rate limit tokens
	AcquireToken(ctx context.Context, apiID string, tokens int) (bool, time.Duration)

	// GetRateLimitStatus returns current rate limit status for an API
	GetRateLimitStatus(ctx context.Context, apiID string) (*RateLimitStatus, error)

	// ResetLimit resets the rate limit for a specific API
	ResetLimit(ctx context.Context, apiID string) error

	// WithConfig overrides the default configuration for an API
	WithConfig(apiID string, config RateLimitLimitConfig) RateLimitHandler
}
```

### 4.2 Internal Functions

```go
// CreateTokenBucket creates a new token bucket with the given configuration
func CreateTokenBucket(config RateLimitConfig) *TokenBucket

// RefillTokens adds tokens to the bucket based on elapsed time
func (tb *TokenBucket) RefillTokens()

// TryConsume attempts to consume tokens from the bucket
func (tb *TokenBucket) TryConsume(tokens int) bool

// WaitForTokens waits until the required tokens are available
func (tb *TokenBucket) WaitForTokens(tokens int, maxWait time.Duration) error

// UpdateRateLimitStatus updates rate limit status from API response headers
func (rh *RateLimitHandler) UpdateRateLimitStatus(apiID string, status RateLimitStatus)

// CalculateBackoff calculates exponential backoff duration
func (rh *RateLimitHandler) CalculateBackoff(retryCount int) time.Duration
```

### 4.3 Configuration Options

```go
// Default configurations for supported APIs
var DefaultConfigs = map[string]RateLimitConfig{
	"usda": {
		RequestsPerSecond: 10,
		RequestsPerMinute: 500,
		RequestsPerDay:    10000,
		BurstSize:         20,
		BackoffMultiplier: 2.0,
		MaxBackoff:        60 * time.Second,
	},
	"openfoodfacts": {
		RequestsPerSecond: 5,
		RequestsPerMinute: 300,
		RequestsPerDay:    5000,
		BurstSize:         10,
		BackoffMultiplier: 2.0,
		MaxBackoff:        60 * time.Second,
	},
}
```

### 4.4 Usage Example

```go
// In external data integration service
func (s *ExternalDataService) SearchUSDAFoods(
	ctx context.Context,
	query string,
) ([]FoodItem, error) {
	handler := s.rateLimitHandler

	result := handler.ExecuteWithRateLimit(ctx, "usda", func(ctx context.Context) (*APIResponse[[]FoodItem], error) {
		return s.usdaClient.Search(ctx, query)
	})

	if !result.Success {
		if result.Error != nil {
			return nil, result.Error
		}
		return []FoodItem{}, nil
	}

	return result.Data, nil
}
```
