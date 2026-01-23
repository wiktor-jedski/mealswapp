# RateLimiter (Fiber Built-in Limiter)

**Traceability:** ARCH-013

## 1. Data Structures & Types

```go
package middleware

import (
    "github.com/gofiber/fiber/v2/middleware/limit"
)

type RateLimiterConfig struct {
    Max            int           `yaml:"max"`
    Expiration     time.Duration `yaml:"expiration"`
    Key            func(c *fiber.Ctx) string `yaml:"-"`
    ErrorResponse  fiber.Handler `yaml:"-"`
    Skip           func(c *fiber.Ctx) bool `yaml:"-"`
    LimitReached   fiber.Handler `yaml:"-"`
}

type RateLimiterStore interface {
    Get(key string) (int, bool)
    Increment(key string) int
    Reset(key string)
}

type inMemoryStore struct {
    mu     sync.RWMutex
    counts map[string]int
    expiry map[string]time.Time
}

type redisStore struct {
    client    *redis.Client
    keyPrefix string
    window    time.Duration
}
```

## 2. Logic & Algorithms (Step-by-Step)

### 2.1 Rate Limiter Initialization

```
1. Load configuration from config.yaml
2. Parse Max requests per window
3. Parse Expiration window duration
4. Create appropriate store (in-memory or Redis)
5. Register middleware with Fiber app
6. Return configured middleware handler
```

### 2.2 Request Rate Limit Check

```
1. Extract client identifier from request
   - If custom Key function provided, use it
   - Default: X-Forwarded-For header or c.IP()

2. Check if request should be skipped
   - If Skip function returns true, continue to next handler

3. Get current request count for client
   - If client not in store, initialize count to 0

4. Increment request count
   - If count exceeds Max, trigger rate limit exceeded
   - Set X-RateLimit-Limit, X-RateLimit-Remaining, X-RateLimit-Reset headers

5. If limit exceeded:
   - Return 429 Too Many Requests status
   - Call LimitReached handler if defined
   - Log rate limit event

6. If within limit:
   - Pass request to next handler
   - Update store with new count
```

### 2.3 Sliding Window Algorithm

```
For in-memory store:
1. Each key has: {count, expiration_time}
2. On increment:
   - Check if current time > expiration_time
   - If expired: reset count to 1, set new expiration
   - If not expired: increment count
3. Cleanup expired entries periodically

For Redis store:
1. Use Redis INCR with TTL
2. Key format: "ratelimit:<client_id>:<window_start>"
3. Window start = current_time / window_size * window_size
4. TTL = window_size + buffer
```

## 3. State Management & Error Handling

### 3.1 Error States

| Error | Condition | Response | Action |
| :--- | :--- | :--- | :--- |
| RateLimitExceeded | Request count > Max | 429 Too Many Requests | Return error response, log event |
| RedisConnectionFailed | Redis store unavailable | 503 Service Unavailable | Fall back to in-memory or reject |
| InvalidConfiguration | Missing Max or invalid values | Panic during initialization | Fail startup, log error |
| ClientIPExtractionFailed | Unable to determine client IP | 400 Bad Request | Skip rate limiting, allow request |

### 3.2 State Transitions

```
IDLE -> ACTIVE
  - First request from client initializes entry

ACTIVE -> ACTIVE
  - Subsequent requests within window increment counter
  - Headers updated with remaining requests

ACTIVE -> EXPIRED
  - Window expires, counter resets to 0
  - Next request starts new window (IDLE -> ACTIVE)

ACTIVE -> BLOCKED
  - Request count exceeds Max
  - Return 429, do not increment
  - Remain in BLOCKED until window expires
```

### 3.3 Recovery Strategies

```
1. Redis Failure Fallback:
   - If Redis store fails, switch to in-memory store
   - Log warning: "Redis rate limit store unavailable, using in-memory"
   - Continue serving requests with in-memory limits

2. Configuration Reload:
   - On SIGHUP, reload configuration
   - Preserve existing counts when possible
   - Apply new Max/Expiration values to new requests

3. Memory Pressure:
   - If in-memory store exceeds max entries, evict oldest
   - Log warning: "Rate limit store memory pressure, evicting entries"
```

## 4. Component Interfaces

### 4.1 Public Interfaces

```go
func NewRateLimiter(cfg RateLimiterConfig) fiber.Handler
```

**Parameters:**
- `cfg`: RateLimiterConfig with Max, Expiration, and optional handlers

**Returns:**
- `fiber.Handler`: Middleware handler function

**Usage:**
```go
app.Use(limit.New(limit.Config{
    Max:        100,
    Expiration: 1 * time.Minute,
}))
```

### 4.2 Configuration Options

```go
type Config struct {
    // Max number of requests per window
    Max int

    // Window expiration duration
    Expiration time.Duration

    // Custom key extractor function
    Key func(*fiber.Ctx) string

    // Custom error response handler
    ErrorResponse fiber.Handler

    // Custom skip function
    Skip func(*fiber.Ctx) bool

    // Custom limit reached handler
    LimitReached fiber.Handler
}
```

### 4.3 Store Interface

```go
type Store interface {
    Get(key string) (int, bool)
    Increment(key string) int
    Reset(key string)
    Close() error
}
```

### 4.4 Response Headers

The rate limiter sets the following headers on all responses:

| Header | Description |
| :--- | :--- |
| X-RateLimit-Limit | Maximum requests allowed in the window |
| X-RateLimit-Remaining | Remaining requests in the current window |
| X-RateLimit-Reset | Unix timestamp when the window resets |
| Retry-After | Seconds until client can retry (only on 429) |
