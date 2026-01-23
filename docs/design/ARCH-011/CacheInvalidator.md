---
## FILE: CacheInvalidator.md
**Traceability:** ARCH-011

### 1. Data Structures & Types

```go
package cache

import (
	"context"
	"time"
)

type CacheKey string
type CachePattern string
type ItemID string
type Category string

type InvalidationRequest struct {
	ItemIDs     []ItemID    `json:"item_ids"`
	Categories  []Category  `json:"categories"`
	Pattern     CachePattern `json:"pattern"`
	TTL         time.Duration `json:"ttl"`
	PropagateTo []ClientEndpoint `json:"propagate_to"`
}

type ClientEndpoint string

const (
	EndpointRedis          ClientEndpoint = "redis"
	EndpointServiceWorker  ClientEndpoint = "service_worker"
)

type CacheInvalidationResult struct {
	Endpoint   ClientEndpoint `json:"endpoint"`
	KeysDeleted int           `json:"keys_deleted"`
	Error      error          `json:"error"`
	Timestamp  time.Time      `json:"timestamp"`
}

type ItemMetadata struct {
	ID          ItemID    `json:"id"`
	Name        string    `json:"name"`
	Category    Category  `json:"category"`
	ImageURL    string    `json:"image_url"`
	UpdatedAt   time.Time `json:"updated_at"`
	Version     int       `json:"version"`
}

type ServiceWorkerMessage struct {
	Type       string          `json:"type"`
	Action     string          `json:"action"`
	Patterns   []CachePattern  `json:"patterns"`
	ItemIDs    []ItemID        `json:"item_ids"`
	Timestamp  time.Time       `json:"timestamp"`
}

type RedisCacheKeys struct {
	FoodItemPrefix    CachePattern = "food_item:"
	SimilarityPrefix  CachePattern = "similarity:"
	SearchQueryPrefix CachePattern = "search_query:"
	UserDataPrefix    CachePattern = "user_data:"
	JobResultPrefix   CachePattern = "job_result:"
}

type InvalidationPolicy struct {
	ImmediateTTL  time.Duration
	DelayedTTL    time.Duration
	BatchSize     int
	MaxRetries    int
	RetryDelay    time.Duration
}

type CacheInvalidator struct {
	redisClient    *redis.Client
	httpClient     *http.Client
	policy         InvalidationPolicy
	metrics        *InvalidatorMetrics
	pubsub         *redis.PubSub
}

type InvalidatorMetrics struct {
	KeysInvalidated  int64
	ErrorsCount      int64
	LastRun          time.Time
	AvgLatencyMs     float64
}

type BatchInvalidationJob struct {
	ID          string
	Requests    []InvalidationRequest
	Status      JobStatus
	CreatedAt   time.Time
	CompletedAt *time.Time
}

type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusProcessing JobStatus = "processing"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
)
```

### 2. Logic & Algorithms (Step-by-Step)

**Algorithm: InvalidateCache**

```
FUNCTION InvalidateCache(ctx context.Context, req InvalidationRequest) []CacheInvalidationResult
    results := EMPTY_LIST
    workers := EMPTY_LIST

    IF req.PropagateTo CONTAINS EndpointRedis THEN
        workers.Add(RedisInvalidationWorker)
    END IF

    IF req.PropagateTo CONTAINS EndpointServiceWorker THEN
        workers.Add(ServiceWorkerInvalidationWorker)
    END IF

    FOR EACH worker IN workers DO
        result := EXECUTE worker(ctx, req)
        results.Add(result)
        RECORD_METRIC(worker.Endpoint, result)
    END FOR

    RETURN results
END FUNCTION
```

**Algorithm: RedisInvalidationWorker**

```
FUNCTION RedisInvalidationWorker(ctx context.Context, req InvalidationRequest) CacheInvalidationResult
    keysDeleted := 0
    client := GetRedisClient()

    FOR EACH itemID IN req.ItemIDs DO
        FOR EACH prefix IN [FoodItemPrefix, SimilarityPrefix, SearchQueryPrefix] DO
            pattern := CONCAT(prefix, itemID)
            keys := client.Keys(ctx, pattern)

            FOR EACH key IN keys DO
                client.Del(ctx, key)
                keysDeleted += 1
            END FOR
        END FOR
    END FOR

    IF req.Pattern != "" THEN
        keys := client.Keys(ctx, req.Pattern)
        FOR EACH key IN keys DO
            client.Del(ctx, key)
            keysDeleted += 1
        END FOR
    END IF

    RETURN CacheInvalidationResult{
        Endpoint:   EndpointRedis,
        KeysDeleted: keysDeleted,
        Timestamp:  NOW()
    }
END FUNCTION
```

**Algorithm: ServiceWorkerInvalidationWorker**

```
FUNCTION ServiceWorkerInvalidationWorker(ctx context.Context, req InvalidationRequest) CacheInvalidationResult
    message := ServiceWorkerMessage{
        Type:      "CACHE_INVALIDATION",
        Action:    "PURGE",
        Patterns:  EXTRACT_PATTERNS(req),
        ItemIDs:   req.ItemIDs,
        Timestamp: NOW()
    }

    BROADCAST_TO_ALL_CLIENTS(message)

    RETURN CacheInvalidationResult{
        Endpoint:   EndpointServiceWorker,
        KeysDeleted: LEN(req.ItemIDs),
        Timestamp:  NOW()
    }
END FUNCTION
```

**Algorithm: BroadcastToAllClients (WebSocket/BroadcastChannel)**

```
FUNCTION BroadcastToAllClients(message ServiceWorkerMessage)
    hub := GetWebSocketHub()

    FOR EACH client IN hub.GetConnectedClients() DO
        IF client.HasServiceWorker() THEN
            client.Send(message)
        END IF
    END FOR
END FUNCTION
```

**Algorithm: HandleAdminDataUpdate**

```
FUNCTION HandleAdminDataUpdate(ctx context.Context, updatedItems []ItemMetadata)
    req := InvalidationRequest{
        ItemIDs:      EXTRACT_IDS(updatedItems),
        Categories:   EXTRACT_CATEGORIES(updatedItems),
        PropagateTo:  [EndpointRedis, EndpointServiceWorker],
        TTL:          0
    }

    results := InvalidateCache(ctx, req)

    FOR EACH result IN results DO
        IF result.Error != nil THEN
            LOG_ERROR("Cache invalidation failed", result)
            IF result.Endpoint == EndpointRedis THEN
                RETRY_WITH_BACKOFF(ctx, InvalidateCache, req)
            END IF
        END IF
    END FOR

    UPDATE_ITEM_VERSIONS(updatedItems)
END FUNCTION
```

**Algorithm: BatchInvalidation**

```
FUNCTION BatchInvalidation(ctx context.Context, requests []InvalidationRequest) BatchInvalidationJob
    job := BatchInvalidationJob{
        ID:         GENERATE_UUID(),
        Requests:   requests,
        Status:     JobStatusPending,
        CreatedAt:  NOW()
    }

    FOR EACH req IN requests DO
        InvalidateCache(ctx, req)
    END FOR

    job.Status = JobStatusCompleted
    completed := NOW()
    job.CompletedAt = &completed

    RETURN job
END FUNCTION
```

### 3. State Management & Error Handling

| State | Trigger | Transition | Action |
| :--- | :--- | :--- | :--- |
| Idle | Initial state | On invalidation request | ProcessRequest |
| Processing | Cache key found | On batch completion | Cleanup |
| Processing | Redis connection lost | Retry exhaustion | TransitionToFailed |
| Failed | Max retries reached | Manual intervention required | LogAndAlert |
| RetryBackoff | Transient error | On timer expiry | RetryRequest |

**Error States:**

| Error Code | Description | Recovery Strategy |
| :--- | :--- | :--- |
| `ERR_REDIS_CONNECTION` | Redis client disconnected | Reconnect with exponential backoff |
| `ERR_REDIS_TIMEOUT` | Redis operation timed out | Retry with increased timeout |
| `ERR_REDIS_KEY_NOT_FOUND` | Key to delete not found | Log and continue (idempotent) |
| `ERR_WEBSOCKET_SEND` | Client message failed | Queue for later broadcast |
| `ERR_BATCH_TOO_LARGE` | Batch size exceeds limit | Split into smaller batches |
| `ERR_INVALID_ITEM_ID` | Item ID format invalid | Validate input before processing |

**State Transitions:**

```
STATE Machine: CacheInvalidator

STATE Idle:
    ON Enter:
        InitializeMetrics()
    ON Event Invalidate([itemIDs]):
        SET currentItems = itemIDs
        TRANSITION TO Processing

STATE Processing:
    ON Enter:
        StartMetricsTimer()
    ON Event RedisComplete(success):
        IF success THEN
            SET redisResult = success
            IF serviceWorkerComplete THEN
                TRANSITION TO Cleanup
            END IF
        ELSE
            SET retryCount += 1
            IF retryCount < MaxRetries THEN
                TRANSITION TO RetryBackoff
            ELSE
                TRANSITION TO Failed
            END IF
        END IF
    ON Event ServiceWorkerComplete(success):
        SET serviceWorkerResult = success
        IF redisComplete THEN
            TRANSITION TO Cleanup
        END IF

STATE RetryBackoff:
    ON Enter:
        ScheduleRetry(delay * exponentialBase ^ retryCount)
    ON Event TimerExpiry:
        TRANSITION TO Processing

STATE Cleanup:
    ON Enter:
        FinalizeMetrics()
        LogResults()
    ON Event Done:
        TRANSITION TO Idle

STATE Failed:
    ON Enter:
        LogFailure()
        AlertAdmin()
    ON Event ManualReset:
        TRANSITION TO Idle
```

### 4. Component Interfaces

```go
type CacheInvalidatorInterface interface {
	InvalidateCache(ctx context.Context, req InvalidationRequest) []CacheInvalidationResult
	InvalidateByItemIDs(ctx context.Context, itemIDs []ItemID) error
	InvalidateByPattern(ctx context.Context, pattern CachePattern) error
	InvalidateByCategory(ctx context.Context, category Category) error
	BroadcastToServiceWorkers(ctx context.Context, msg ServiceWorkerMessage) error
	GetMetrics(ctx context.Context) (*InvalidatorMetrics, error)
	ResetMetrics(ctx context.Context) error
}

type RedisCacheOperations interface {
	Keys(ctx context.Context, pattern string) *redis.StringSliceCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	Exists(ctx context.Context, keys ...string) *redis.IntCmd
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Pipeline() redis.Pipeliner
}

type ServiceWorkerCommunicator interface {
	Connect(ctx context.Context, userID string) error
	Disconnect(userID string)
	Send(ctx context.Context, userID string, msg ServiceWorkerMessage) error
	Broadcast(ctx context.Context, msg ServiceWorkerMessage) error
	GetConnectedClients() []string
}
```

**Function Signatures:**

```go
func NewCacheInvalidator(redisClient *redis.Client, httpClient *http.Client, policy InvalidationPolicy) *CacheInvalidator

func (ci *CacheInvalidator) InvalidateCache(ctx context.Context, req InvalidationRequest) []CacheInvalidationResult

func (ci *CacheInvalidator) InvalidateByItemIDs(ctx context.Context, itemIDs []ItemID) error

func (ci *CacheInvalidator) InvalidateByPattern(ctx context.Context, pattern CachePattern) error

func (ci *CacheInvalidator) InvalidateByCategory(ctx context.Context, category Category) error

func (ci *CacheInvalidator) BroadcastToServiceWorkers(ctx context.Context, msg ServiceWorkerMessage) error

func (ci *CacheInvalidator) HandleAdminDataUpdate(ctx context.Context, items []ItemMetadata) error

func (ci *CacheInvalidator) BatchInvalidation(ctx context.Context, requests []InvalidationRequest) BatchInvalidationJob

func (ci *CacheInvalidator) GetMetrics(ctx context.Context) (*InvalidatorMetrics, error)

func (ci *CacheInvalidator) ResetMetrics(ctx context.Context) error

func (ci *CacheInvalidator) Start(ctx context.Context) error

func (ci *CacheInvalidator) Stop(ctx context.Context) error
```

**External Dependencies:**

```go
import (
	"context"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)
```
