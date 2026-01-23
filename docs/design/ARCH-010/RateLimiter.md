# RateLimiter

**Traceability:** ARCH-010

## 1. Data Structures & Types

```go
package middleware

import (
    "time"

    "github.com/gofiber/fiber/v2"
)

// RateLimiterConfig holds the configuration for the rate limiter.
type RateLimiterConfig struct {
    // MaxRequests is the maximum number of requests allowed per window.
    MaxRequests int

    // WindowDuration is the duration of the sliding window.
    WindowDuration time.Duration

    // KeyGenerator is a function to generate rate limit keys.
    // Defaults to client IP if not provided.
    KeyGenerator func(*fiber.Ctx) string

    // SkipFailedRequests determines whether failed requests count towards the limit.
    SkipFailedRequests bool

    // SkipSuccessfulRequests determines whether successful requests count towards the limit.
    SkipSuccessfulRequests bool

    // LimitReachedResponse is the custom response for rate limited requests.
    LimitReachedResponse fiber.Handler

    // Skip is a function to determine if the request should be skipped.
    Skip func(*fiber.Ctx) bool

    // ExpirationDuration is the TTL for rate limit keys in Redis.
    ExpirationDuration time.Duration
}

// RateLimiterStats holds statistics for the rate limiter.
type RateLimiterStats struct {
    // CurrentRequests is the current number of requests in the window.
    CurrentRequests int64

    // RemainingRequests is the remaining requests in the window.
    RemainingRequests int64

    // ResetTime is the time when the rate limit resets.
    ResetTime time.Time

    // IsLimited indicates if the request is rate limited.
    IsLimited bool
}

// EndpointConfig holds endpoint-specific rate limit configuration.
type EndpointConfig struct {
    // Pattern is the URL pattern for the endpoint.
    Pattern string

    // MaxRequests is the maximum requests for this endpoint.
    MaxRequests int

    // WindowDuration is the window duration for this endpoint.
    WindowDuration time.Duration
}
```

## 2. Logic & Algorithms

### 2.1 Sliding Window Algorithm

The rate limiter uses a sliding window algorithm implemented with Redis.

```
1. Generate rate limit key using KeyGenerator (defaults to client IP)
2. Check if request should be skipped (Skip function returns true)
3. Calculate current window start time (now - WindowDuration)
4. Execute Redis pipeline:
   a. ZREMRANGEBYSCORE - Remove expired entries outside window
   b. ZADD - Add current request timestamp
   c. ZCARD - Count total requests in window
   d. EXPIRE - Set TTL for key expiration
5. If count exceeds MaxRequests:
   a. Return 429 Too Many Requests response
   b. Set rate limit headers
6. If request allowed:
   a. Set rate limit headers (X-RateLimit-Limit, X-RateLimit-Remaining, X-RateLimit-Reset)
   b. Continue to next middleware/handler
```

### 2.2 Endpoint-Specific Configuration

```
1. Iterate through EndpointConfig list to find matching pattern
2. If match found, use endpoint-specific MaxRequests and WindowDuration
3. If no match, use default RateLimiterConfig
```

### 2.3 Login-Specific Rate Limiting (10 failed attempts per 10 minutes)

```
1. Generate login-specific key: "login_failed:{ip}:{window_start}"
2. Increment counter on failed authentication
3. If counter exceeds 10:
   a. Return 429 with message explaining lockout
   b. Include retry-after header
4. On successful login, clear failed attempt counter
```

### 2.4 Rate Limit Headers

```
X-RateLimit-Limit: Total requests allowed in window
X-RateLimit-Remaining: Requests remaining in current window
X-RateLimit-Reset: Unix timestamp when window resets
Retry-After: Seconds until rate limit resets (only on 429)
```

## 3. State Management & Error Handling

### 3.1 Error States

| Error State | Condition | Response | Action |
|-------------|-----------|----------|--------|
| RateLimitExceeded | Request count exceeds MaxRequests | HTTP 429 | Return rate limit response, set Retry-After header |
| RedisConnectionFailed | Redis unavailable | HTTP 503 | Allow request, log error, continue |
| RedisTimeout | Redis operation times out | HTTP 503 | Allow request, log error, continue |
| InvalidConfiguration | MaxRequests <= 0 or WindowDuration <= 0 | N/A | Use defaults, log warning |
| LoginLockoutActive | >10 failed login attempts | HTTP 429 | Return login lockout message |

### 3.2 State Transitions

```
State: NotLimited -> (count > MaxRequests) -> Limited
State: Limited -> (window expires) -> NotLimited
State: NotLimited -> (Redis failure) -> DegradedMode (allow requests)
State: DegradedMode -> (Redis recovers) -> NotLimited
```

### 3.3 Degraded Mode

On Redis failure, the rate limiter enters degraded mode:
- All requests are allowed
- Errors are logged with context
- System continues to attempt Redis connection
- Once Redis recovers, normal rate limiting resumes

## 4. Component Interfaces

```go
// NewRateLimiter creates a new rate limiter middleware with the given configuration.
func NewRateLimiter(config RateLimiterConfig) fiber.Handler

// NewLoginRateLimiter creates a rate limiter specifically for login endpoints.
func NewLoginRateLimiter(maxAttempts int, windowDuration time.Duration) fiber.Handler

// EndpointLimiter creates a rate limiter with endpoint-specific configurations.
func EndpointLimiter(endpointConfigs []EndpointConfig, defaultConfig RateLimiterConfig) fiber.Handler

// GetRateLimiterStats returns current rate limiting statistics for a key.
func GetRateLimiterStats(ctx *fiber.Ctx) (RateLimiterStats, error)

// ResetRateLimiter clears the rate limit for a specific key.
func ResetRateLimiter(ctx *fiber.Ctx, key string) error

// RateLimiter returns the Fiber rate limiter middleware instance.
func RateLimiter(config ...Config) fiber.Handler
```

### 4.1 NewRateLimiter Signature

```go
func NewRateLimiter(config RateLimiterConfig) fiber.Handler
```

**Parameters:**
- `config` - RateLimiterConfig with rate limiting settings

**Returns:**
- `fiber.Handler` - Middleware handler function

**Behavior:**
- Initializes Redis client for rate limit storage
- Sets up default key generator (client IP)
- Configures skip functions if provided
- Returns middleware that enforces rate limits

### 4.2 NewLoginRateLimiter Signature

```go
func NewLoginRateLimiter(maxAttempts int, windowDuration time.Duration) fiber.Handler
```

**Parameters:**
- `maxAttempts` - Maximum failed login attempts (default: 10)
- `windowDuration` - Time window for counting attempts (default: 10 minutes)

**Returns:**
- `fiber.Handler` - Middleware handler for login endpoints

**Behavior:**
- Creates specialized rate limiter for authentication
- Uses separate Redis key namespace for login failures
- On lockout, includes retry-after in response
- Clears counter on successful login

### 4.3 EndpointLimiter Signature

```go
func EndpointLimiter(endpointConfigs []EndpointConfig, defaultConfig RateLimiterConfig) fiber.Handler
```

**Parameters:**
- `endpointConfigs` - Slice of endpoint-specific configurations
- `defaultConfig` - Default configuration for unmatched endpoints

**Returns:**
- `fiber.Handler` - Middleware that applies per-endpoint limits

**Behavior:**
- Matches request path against endpoint patterns
- Applies specific configuration for matched endpoints
- Falls back to default configuration
- Patterns support Fiber wildcards and parameters

### 4.4 GetRateLimiterStats Signature

```go
func GetRateLimiterStats(ctx *fiber.Ctx) (RateLimiterStats, error)
```

**Parameters:**
- `ctx` - Fiber context with rate limit key

**Returns:**
- `RateLimiterStats` - Current rate limiting statistics
- `error` - Any error encountered

**Behavior:**
- Queries Redis for current request count
- Calculates remaining requests
- Returns reset time as time.Time

### 4.5 ResetRateLimiter Signature

```go
func ResetRateLimiter(ctx *fiber.Ctx, key string) error
```

**Parameters:**
- `ctx` - Fiber context
- `key` - Rate limit key to reset

**Returns:**
- `error` - Any error encountered

**Behavior:**
- Deletes rate limit key from Redis
- Used for admin operations or testing
