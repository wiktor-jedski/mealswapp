# RedisCache (Server)

**Traceability:** ARCH-011

## 1. Data Structures & Types

### 1.1 Configuration Types

```go
type RedisConfig struct {
    Addr     string
    Password string
    DB       int
    PoolSize int
    DialTimeout  time.Duration
    ReadTimeout  time.Duration
    WriteTimeout time.Duration
}

type CacheTTLConfig struct {
    FoodItem        time.Duration
    SimilarityCalc  time.Duration
    SessionData     time.Duration
    LPJobResult     time.Duration
    SearchHistory   time.Duration
    UserPrefix      string
}
```

### 1.2 Cache Key Types

```go
type CacheKeyPrefix string

const (
    KeyPrefixFoodItem     CacheKeyPrefix = "food:"
    KeyPrefixSimilarity   CacheKeyPrefix = "sim:"
    KeyPrefixSession      CacheKeyPrefix = "session:"
    KeyPrefixLPJob        CacheKeyPrefix = "lpjob:"
    KeyPrefixSearchHist   CacheKeyPrefix = "search:"
    KeyPrefixUserData     CacheKeyPrefix = "user:"
)

type CacheKey struct {
    Prefix CacheKeyPrefix
    ID     string
}

func (k CacheKey) String() string {
    return string(k.Prefix) + k.ID
}
```

### 1.3 Cached Data Types

```go
type CachedFoodItem struct {
    ID          string
    Name        string
    Calories    float64
    Protein     float64
    Carbs       float64
    Fat         float64
    ImageURL    string
    Category    string
    CachedAt    time.Time
}

type CachedSimilarityResult struct {
    SourceItemID  string
    TargetItems   []SimilarItem
    CalculatedAt  time.Time
}

type SimilarItem struct {
    ItemID     string
    Score      float64
    Name       string
    ImageURL   string
}

type CachedLPJobResult struct {
    JobID       string
    UserID      string
    Status      JobStatus
    Result      *LPJobOutput
    ErrorMsg    string
    CreatedAt   time.Time
    ExpiresAt   time.Time
}

type JobStatus string

const (
    JobStatusPending   JobStatus = "pending"
    JobStatusRunning   JobStatus = "running"
    JobStatusCompleted JobStatus = "completed"
    JobStatusFailed    JobStatus = "failed"
)

type LPJobOutput struct {
    TotalCalories  float64
    TotalProtein   float64
    TotalCarbs     float64
    TotalFat       float64
    Items          []LPMealItem
}

type LPMealItem struct {
    FoodItemID  string
    Quantity    float64
    Name        string
}
```

### 1.4 User Data Purge Types

```go
type UserPurgeResult struct {
    UserID           string
    DeletedKeys      []string
    SessionInvalidated bool
    SearchHistoryCleared bool
    Error            error
}

type UserPurgeRequest struct {
    UserID   string
    Reason   PurgeReason
}

type PurgeReason string

const (
    PurgeReasonAccountDeletion PurgeReason = "account_deletion"
    PurgeReasonUserRequest    PurgeReason = "user_request"
    PurgeReasonAdminAction    PurgeReason = "admin_action"
)
```

### 1.5 Main Cache Struct

```go
type RedisCache struct {
    client    *redis.Client
    ttlConfig CacheTTLConfig
    logger    *log.Logger
}

type CacheResult[T any] struct {
    Data   T
    Hit    bool
    Error  error
}
```

## 2. Logic & Algorithms

### 2.1 Initialization

```
INITIALIZE_REDIS_CACHE(config, ttlConfig, logger):
1. Create redis.Client with config.Addr, config.Password, config.DB, config.PoolSize
2. Set dial, read, and write timeouts from config
3. Verify connection with PING command
4. If PING fails, return error with connection details
5. Store client and ttlConfig in RedisCache struct
6. Return initialized RedisCache instance
```

### 2.2 Food Item Caching

```
CACHE_FOOD_ITEM(item):
1. Generate cache key: CacheKey{KeyPrefixFoodItem, item.ID}
2. Serialize CachedFoodItem to JSON
3. SETEX key with ttlConfig.FoodItem duration
4. If error, log error and return false
5. Return true on success

GET_FOOD_ITEM(itemID):
1. Generate cache key: CacheKey{KeyPrefixFoodItem, itemID}
2. GET key from Redis
3. If nil, return CacheResult{Hit: false}
4. Deserialize JSON to CachedFoodItem
5. Return CacheResult{Data: item, Hit: true}
6. If error, return CacheResult{Error: error}
```

### 2.3 Similarity Calculation Caching

```
CACHE_SIMILARITY_RESULT(sourceItemID, result):
1. Generate cache key: CacheKey{KeyPrefixSimilarity, sourceItemID}
2. Serialize CachedSimilarityResult to JSON
3. SETEX key with ttlConfig.SimilarityCalc duration
4. Return success/failure

GET_SIMILARITY_RESULT(sourceItemID):
1. Generate cache key: CacheKey{KeyPrefixSimilarity, sourceItemID}
2. GET key from Redis
3. If nil, return CacheResult{Hit: false}
4. Deserialize and return cached result
```

### 2.4 LP Job Result Caching

```
CACHE_LP_JOB_RESULT(job):
1. Generate cache key: CacheKey{KeyPrefixLPJob, job.JobID}
2. Serialize CachedLPJobResult to JSON
3. Calculate TTL as job.ExpiresAt - time.Now()
4. If TTL <= 0, use default ttlConfig.LPJobResult
5. SETEX key with calculated TTL
6. Return success/failure

GET_LP_JOB_RESULT(jobID):
1. Generate cache key: CacheKey{KeyPrefixLPJob, jobID}
2. GET key from Redis
3. If nil, return CacheResult{Hit: false}
4. Deserialize and return cached result
5. Check if job is expired; if so, return CacheResult{Hit: false}
```

### 2.5 Cache Invalidation (Admin Updates)

```
INVALIDATE_FOOD_ITEM(itemID):
1. Generate cache key: CacheKey{KeyPrefixFoodItem, itemID}
2. DEL key from Redis
3. Log invalidation action

INVALIDATE_SIMILARITY_RESULTS(itemID):
1. Generate pattern: KeyPrefixSimilarity + itemID + ":*"
2. SCAN matching keys
3. DEL all matching keys
4. Log invalidation count

INVALIDATE_USER_RELATED(userID):
1. Generate patterns for each prefix with userID
2. SCAN matching keys across all patterns
3. DEL all matching keys
4. Return count of deleted keys
```

### 2.6 User Data Purge (GDPR)

```
PURGE_USER_DATA(request):
1. keysToDelete = []
2. patterns = [
    KeyPrefixFoodItem + request.UserID + ":*",
    KeyPrefixSimilarity + "*" + request.UserID + "*",
    KeyPrefixSession + request.UserID,
    KeyPrefixLPJob + request.UserID + ":*",
    KeyPrefixSearchHist + request.UserID
   ]
3. FOR each pattern in patterns:
   a. SCAN all matching keys
   b. Add to keysToDelete
4. IF keysToDelete is not empty:
   a. DEL keysToDelete
5. Construct UserPurgeResult with:
   a. UserID = request.UserID
   b. DeletedKeys = keysToDelete
   c. SessionInvalidated = true
   d. SearchHistoryCleared = true
6. Return UserPurgeResult
```

### 2.7 LRU Eviction (via TTL)

```
Redis uses built-in eviction, but custom LRU can be implemented:

IMPLEMENT_CUSTOM_LRU(maxItems):
1. Create sorted set: "lru:order"
2. When caching new item:
   a. ZADD "lru:order" with current timestamp as score
   b. ZREMRANGEBYRANK if count > maxItems
3. On cache miss, check if expired
4. Periodically clean expired keys
```

### 2.8 Session Data Management

```
SET_SESSION(userID, sessionData, ttl):
1. key = CacheKey{KeyPrefixSession, userID}
2. Serialize sessionData to JSON
3. SETEX key with ttl duration
4. Return success/failure

GET_SESSION(userID):
1. key = CacheKey{KeyPrefixSession, userID}
2. GET key from Redis
3. If nil, return CacheResult{Hit: false}
4. Deserialize and return session data

INVALIDATE_SESSION(userID):
1. key = CacheKey{KeyPrefixSession, userID}
2. DEL key from Redis
3. Return success
```

## 3. State Management & Error Handling

### 3.1 Error States

| Error Condition | Error Type | Handling Strategy |
| :--- | :--- | :--- |
| Redis connection timeout | ConnectionError | Retry with exponential backoff, max 3 attempts |
| Redis connection refused | ConnectionError | Log error, return cache miss to caller |
| Authentication failed | AuthError | Return fatal error, application cannot start |
| Out of memory | MemoryError | Evict oldest items, log warning |
| Key not found | CacheMiss | Return CacheResult{Hit: false} |
| Serialization error | InternalError | Log error, return cache miss |
| Deserialization error | InternalError | Log error, delete corrupted key, return cache miss |
| Network interruption | NetworkError | Retry operation, fallback to cache miss |
| TTL configuration error | ConfigError | Use default TTL values |

### 3.2 State Transitions

```
State Machine for Cache Operations:

IDLE -> CONNECTING -> CONNECTED
                      |
                      -> ERROR (on connection failure)
                         |
                         -> RETRYING (up to 3 attempts)
                            |
                            -> CONNECTED (on success)
                            -> ERROR (on exhaustion)

CACHE_OPERATION:
READY -> IN_PROGRESS -> COMPLETED
                  |       |
                  |       -> ERROR (on Redis error)
                  |           |
                  |           -> FALLBACK (return cache miss)
                  |
                  -> TIMEOUT
                      |
                      -> FALLBACK

USER_PURGE:
PENDING -> EXECUTING -> COMPLETED
                  |       |
                  |       -> PARTIAL (some keys failed)
                  |           |
                  |           -> LOGGED
                  |
                  -> ERROR
                      |
                      -> LOGGED + RETURN_ERROR
```

### 3.3 Retry Logic

```
RETRY_OPERATION(operation, maxRetries):
1. attempt = 0
2. WHILE attempt < maxRetries:
   a. result = operation()
   b. IF result is success:
      i. Return result
   c. IF result is retryable error:
      i. sleep = 2^attempt * 100ms
      ii. sleep(sleep)
      iii. attempt++
   d. IF result is non-retryable:
      i. Return result
3. Return error: "max retries exceeded"
```

### 3.4 Health Check

```
HEALTH_CHECK():
1. PING Redis server
2. IF PING succeeds:
   a. Get memory usage: INFO memory
   b. Get connected clients: INFO clients
   c. Return Healthy status with metrics
3. IF PING fails:
   a. Return Unhealthy status
   b. Log error details
   c. Trigger reconnection attempt
```

### 3.5 Logging

```
LOG_CACHE_OPERATION(operation, key, result):
1. IF result.Error != nil:
   a. logger.Error("cache_operation_failed",
      "operation", operation,
      "key", key,
      "error", result.Error)
2. ELSE:
   a. logger.Debug("cache_operation_success",
      "operation", operation,
      "key", key,
      "hit", result.Hit)

LOG_PURGE_EVENT(result):
1. logger.Info("user_data_purged",
   "user_id", result.UserID,
   "keys_deleted", len(result.DeletedKeys),
   "session_invalidated", result.SessionInvalidated)
```

## 4. Component Interfaces

### 4.1 Public Methods

```go
type RedisCacheInterface interface {
    // Initialization
    NewRedisCache(config RedisConfig, ttlConfig CacheTTLConfig, logger *log.Logger) (*RedisCache, error)
    
    // Connection Management
    Close() error
    Ping() error
    HealthCheck() error
    
    // Food Item Cache
    CacheFoodItem(item *CachedFoodItem) error
    GetFoodItem(itemID string) (*CacheResult[CachedFoodItem], error)
    InvalidateFoodItem(itemID string) error
    
    // Similarity Cache
    CacheSimilarityResult(sourceItemID string, result *CachedSimilarityResult) error
    GetSimilarityResult(sourceItemID string) (*CacheResult[CachedSimilarityResult], error)
    InvalidateSimilarityForItem(itemID string) error
    
    // LP Job Cache
    CacheLPJobResult(job *CachedLPJobResult) error
    GetLPJobResult(jobID string) (*CacheResult[CachedLPJobResult], error)
    InvalidateLPJobResult(jobID string) error
    
    // Session Cache
    SetSession(userID string, sessionData interface{}, ttl time.Duration) error
    GetSession(userID string) (*CacheResult[interface{}], error)
    InvalidateSession(userID string) error
    
    // Search History Cache
    CacheSearchHistory(userID string, queries []string) error
    GetSearchHistory(userID string) (*CacheResult[[]string], error)
    ClearSearchHistory(userID string) error
    
    // Cache Invalidation (Admin)
    InvalidateUserRelatedData(userID string) (int, error)
    InvalidateAllFoodItems() error
    
    // User Data Purge (GDPR)
    PurgeUserData(request *UserPurgeRequest) (*UserPurgeResult, error)
    
    // Utility
    GetTTL(key string) (time.Duration, error)
    FlushPattern(pattern string) (int, error)
}
```

### 4.2 Private Methods

```go
type redisCachePrivate interface {
    buildKey(prefix CacheKeyPrefix, id string) string
    serialize(data interface{}) ([]byte, error)
    deserialize[T any](data []byte) (*T, error)
    executeWithRetry(operation func() error, maxRetries int) error
    scanKeys(pattern string) ([]string, error)
    deleteKeys(keys []string) error
    logOperation(operation string, key string, result error)
}
```

### 4.3 Usage Examples

```go
// Initialization
cache, err := NewRedisCache(config, ttlConfig, logger)
if err != nil {
    log.Fatalf("Failed to initialize Redis cache: %v", err)
}
defer cache.Close()

// Caching food item
foodItem := &CachedFoodItem{
    ID:       "food-123",
    Name:     "Chicken Breast",
    Calories: 165.0,
    Protein:  31.0,
    Carbs:    0.0,
    Fat:      3.6,
    ImageURL: "https://storage.example.com/chicken.jpg",
    Category: "protein",
    CachedAt: time.Now(),
}
err = cache.CacheFoodItem(foodItem)
if err != nil {
    log.Printf("Failed to cache food item: %v", err)
}

// Retrieving food item
result, err := cache.GetFoodItem("food-123")
if err != nil {
    log.Printf("Failed to get food item: %v", err)
} else if !result.Hit {
    log.Printf("Cache miss for food-123")
} else {
    log.Printf("Found cached item: %s", result.Data.Name)
}

// GDPR user purge
purgeRequest := &UserPurgeRequest{
    UserID: "user-456",
    Reason: PurgeReasonAccountDeletion,
}
purgeResult, err := cache.PurgeUserData(purgeRequest)
if err != nil {
    log.Printf("Failed to purge user data: %v", err)
} else {
    log.Printf("Purged %d keys for user %s", len(purgeResult.DeletedKeys), purgeResult.UserID)
}
```

### 4.4 Fiber Middleware Integration

```go
func CacheMiddleware(cache *RedisCache) fiber.Handler {
    return func(c *fiber.Ctx) error {
        // Store cache in context
        c.Locals("redisCache", cache)
        return c.Next()
    }
}

func GetCacheFromContext(c *fiber.Ctx) *RedisCache {
    return c.Locals("redisCache").(*RedisCache)
}
```

### 4.5 Configuration Defaults

```go
DefaultCacheTTLConfig := CacheTTLConfig{
    FoodItem:        24 * time.Hour,
    SimilarityCalc:  7 * 24 * time.Hour,
    SessionData:     7 * 24 * time.Hour,
    LPJobResult:     24 * time.Hour,
    SearchHistory:   30 * 24 * time.Hour,
    UserPrefix:      "user:",
}

DefaultRedisConfig := RedisConfig{
    Addr:            "localhost:6379",
    Password:        "",
    DB:              0,
    PoolSize:        100,
    DialTimeout:     5 * time.Second,
    ReadTimeout:     3 * time.Second,
    WriteTimeout:    3 * time.Second,
}
```
