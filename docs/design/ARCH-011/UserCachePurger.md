# UserCachePurger

**Traceability:** ARCH-011

## 1. Data Structures & Types

```go
package cache

import (
    "context"
    "time"

    "github.com/gofiber/fiber/v2"
    "github.com/redis/go-redis/v9"
)

type UserCachePurger struct {
    redisClient *redis.Client
    sessionStore fiber.Store
}

type PurgeResult struct {
    UserID            string        `json:"user_id"`
    RedisKeysDeleted  int           `json:"redis_keys_deleted"`
    SessionInvalidated bool         `json:"session_invalidated"`
    SearchHistoryCleared bool       `json:"search_history_cleared"`
    Errors            []string      `json:"errors,omitempty"`
    Duration          time.Duration `json:"duration"`
}

type PurgeOptions struct {
    UserID           string
    InvalidateSession bool
    ClearSearchHistory bool
    Timeout          time.Duration
}

const (
    userCacheKeyPrefix     = "user:"
    userSessionPrefix      = "session:"
    userSearchHistoryPrefix = "search:history:"
    defaultPurgeTimeout    = 30 * time.Second
)
```

## 2. Logic & Algorithms

### 2.1 Main Purge Flow

```
PURGE USER DATA (userID: string, options: PurgeOptions): PurgeResult
1.  START_TIMER
2.  Initialize PurgeResult with userID
3.  IF options.Timeout not set THEN set to defaultPurgeTimeout
4.  Create cancellable context with timeout
5.  Launch goroutine: PURGE_REDIS_KEYS(userID, context) → returns keysDeleted, error
6.  Launch goroutine: INVALIDATE_SESSIONS(userID, context) → returns success, error
7.  Launch goroutine: CLEAR_SEARCH_HISTORY(userID, context) → returns success, error
8.  Wait for all goroutines or context timeout
9.  Aggregate results and errors
10. STOP_TIMER and set Duration
11. RETURN PurgeResult
```

### 2.2 Purge Redis Keys

```
PURGE_REDIS_KEYS(userID: string, ctx: context.Context): (int, error)
1.  pattern = userCacheKeyPrefix + userID + ":*"
2.  cursor = 0
3.  totalDeleted = 0
4.  REPEAT
5.      SCAN keys using pattern with cursor
6.      IF error THEN RETURN totalDeleted, error
7.      IF keys NOT empty THEN
8.          DELETE keys
9.          IF error THEN log error but continue
10.         totalDeleted += len(keys)
11.     END IF
12.     cursor = nextCursor
13. UNTIL cursor == 0
14. RETURN totalDeleted, nil
```

### 2.3 Invalidate Sessions

```
INVALIDATE_SESSIONS(userID: string, ctx: context.Context): (bool, error)
1.  sessionID = "session:" + userID
2.  DELETE sessionID from Redis
3.  IF error AND error != redis.Nil THEN RETURN false, error
4.  Call fiber session middleware to destroy all sessions for userID
5.  RETURN true, nil
```

### 2.4 Clear Search History

```
CLEAR_SEARCH_HISTORY(userID: string, ctx: context.Context): (bool, error)
1.  historyKey = userSearchHistoryPrefix + userID
2.  DELETE historyKey from Redis
3.  IF error AND error != redis.Nil THEN RETURN false, error
4.  pattern = userSearchHistoryPrefix + userID + ":*"
5.  SCAN and DELETE all matching keys
6.  RETURN true, nil
```

## 3. State Management & Error Handling

### 3.1 Error States

| Error Condition | Handling Strategy |
| :--- | :--- |
| Redis connection timeout | Retry once with extended timeout, return error if fails |
| Redis key deletion partial failure | Log failed keys, continue with remaining, report in PurgeResult.Errors |
| Session invalidation fails | Log error, mark session_invalidated=false, continue |
| Search history clear fails | Log error, mark search_history_cleared=false, continue |
| Context timeout before completion | Cancel all operations, return partial results with timeout error |
| UserID empty or invalid | Return error immediately without attempting purge |

### 3.2 State Transitions

```
Initial State → Running State
├── All operations launched concurrently
├── Each operation reports status
└── On completion → Completed State

Running State → Completed State
├── All goroutines finished
├── Results aggregated
├── Duration recorded
└── Errors collected

Running State → Failed State
├── Context timeout reached
├── Critical error (e.g., Redis unavailable)
└── Partial results returned with errors
```

### 3.3 Retry Logic

- Redis key deletion: No retry for individual keys, SCAN continues on error
- Session invalidation: Single retry after 100ms delay
- Search history clear: Single retry after 100ms delay

## 4. Component Interfaces

### 4.1 Constructor

```go
func NewUserCachePurger(redisClient *redis.Client, sessionStore fiber.Store) *UserCachePurger
```

### 4.2 Public Methods

```go
func (p *UserCachePurger) PurgeUserData(ctx context.Context, opts PurgeOptions) (*PurgeResult, error)
```

**Parameters:**
- `ctx`: Context for cancellation and timeout
- `opts`: PurgeOptions containing userID and purge configuration

**Returns:**
- `*PurgeResult`: Summary of purge operations
- `error`: Critical error if purge could not start

**Behavior:**
- Deletes all Redis keys prefixed with user ID
- Invalidates user session tokens if options.InvalidateSession is true
- Clears server-side search history if options.ClearSearchHistory is true
- Returns partial results if some operations fail

```go
func (p *UserCachePurger) PurgeOnlyRedisKeys(ctx context.Context, userID string) (int, error)
```

**Parameters:**
- `ctx`: Context for cancellation
- `userID`: User identifier

**Returns:**
- `int`: Number of keys deleted
- `error`: Error if SCAN or DELETE operations fail

```go
func (p *UserCachePurger) InvalidateUserSessions(ctx context.Context, userID string) error
```

**Parameters:**
- `ctx`: Context for cancellation
- `userID`: User identifier

**Returns:**
- `error`: Error if session destruction fails

```go
func (p *UserCachePurger) ClearUserSearchHistory(ctx context.Context, userID string) error
```

**Parameters:**
- `ctx`: Context for cancellation
- `userID`: User identifier

**Returns:**
- `error`: Error if history clear fails

### 4.3 Helper Methods

```go
func (p *UserCachePurger) countKeysWithPrefix(ctx context.Context, prefix string) (int, error)
```

**Returns:** Count of keys matching prefix pattern (used for testing/verification)

```go
func (p *UserCachePurger) buildUserKeyPrefix(userID string) string
```

**Returns:** Formatted key prefix for user-specific cache entries
