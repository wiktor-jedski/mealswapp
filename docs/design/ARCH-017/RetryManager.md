# RetryManager

**Traceability:** ARCH-017

---

## 1. Data Structures & Types

### Backend (Go)

```go
package retry

import "time"

type RetryPolicy struct {
    MaxAttempts       int           // Maximum number of retry attempts
    InitialDelay      time.Duration // Initial delay before first retry
    MaxDelay          time.Duration // Maximum delay between retries
    Multiplier        float64       // Delay multiplier for exponential backoff
    Jitter            bool          // Enable random jitter to prevent thundering herd
    RetryableStatuses []int         // HTTP status codes that should trigger retry
}

type RetryableError interface {
    Error() string
    IsRetryable() bool
    ShouldRetry() bool
}

type RetryResult struct {
    Success    bool
    Attempts   int
    LastError  error
    TotalTime  time.Duration
    FinalValue any // The successfully retrieved value after retries
}

type BackoffCalculator struct {
    policy RetryPolicy
}

type CircuitBreakerState int

const (
    CircuitStateClosed CircuitBreakerState = iota
    CircuitStateOpen
    CircuitStateHalfOpen
)

type CircuitBreaker struct {
    state             CircuitBreakerState
    failureCount      int
    successCount      int
    lastFailureTime   time.Time
    failureThreshold  int
    successThreshold  int
    timeoutDuration   time.Duration
    policy            RetryPolicy
}
```

### Frontend (TypeScript/Svelte)

```typescript
interface RetryPolicy {
  maxAttempts: number;
  initialDelayMs: number;
  maxDelayMs: number;
  multiplier: number;
  jitter: boolean;
  retryableStatusCodes: number[];
}

interface RetryConfig {
  enabled: boolean;
  policy: RetryPolicy;
  onRetry?: (error: Error, attempt: number) => void;
  onSuccess?: (attempt: number, duration: number) => void;
  onFinalFailure?: (error: Error, attempts: number) => void;
}

interface RetryState {
  attempt: number;
  isRetrying: boolean;
  lastError: Error | null;
  nextRetryAt: number | null;
  totalDuration: number;
}

type RetryResult<T> =
  | { success: true; data: T; attempts: number; totalDuration: number }
  | { success: false; error: Error; attempts: number; totalDuration: number };
```

---

## 2. Logic & Algorithms

### 2.1 Exponential Backoff Calculation

```
Algorithm: CalculateBackoffDelay(attempt, policy)
Input: attempt (current retry attempt number), policy (RetryPolicy)
Output: delay duration for this attempt

1. baseDelay ← policy.InitialDelay × (policy.Multiplier ^ (attempt - 1))
2. cappedDelay ← min(baseDelay, policy.MaxDelay)
3. if policy.Jitter is true:
4.     jitterRange ← cappedDelay × 0.1  // 10% jitter range
5.     randomOffset ← random between (-jitterRange, +jitterRange)
6.     finalDelay ← cappedDelay + randomOffset
7. else:
8.     finalDelay ← cappedDelay
9. return max(0, finalDelay)
```

### 2.2 Retry Execution Flow

```
Algorithm: ExecuteWithRetry(operation, policy, timeout)
Input: operation (function to execute), policy (RetryPolicy), timeout (overall timeout)
Output: RetryResult

1. startTime ← current time
2. lastError ← null
3. attempts ← 0

4. while attempts < policy.MaxAttempts:
5.     attempts ← attempts + 1
6.     try:
7.         result ← operation()
8.         if result is not error:
9.             return RetryResult {
10.                Success: true,
11.                Attempts: attempts,
12.                TotalTime: currentTime - startTime,
13.                FinalValue: result
14.            }
15.    catch error as e:
16.        lastError ← e
17.        if e.ShouldRetry() is false:
18.            break
19.        if currentTime - startTime > timeout:
20.            break
21.        if policy.MaxAttempts - attempts > 0:
22.            delay ← CalculateBackoffDelay(attempts, policy)
23.            sleep(delay)

24. return RetryResult {
25.    Success: false,
26.    Attempts: attempts,
27.    LastError: lastError,
28.    TotalTime: currentTime - startTime
29. }
```

### 2.3 Circuit Breaker State Transitions

```
Algorithm: CircuitBreakerExecute(operation, circuitBreaker)
Input: operation, circuitBreaker
Output: result or error

1. if circuitBreaker.state == CircuitStateOpen:
2.     if currentTime - circuitBreaker.lastFailureTime > circuitBreaker.timeoutDuration:
3.         circuitBreaker.state ← CircuitStateHalfOpen
4.     else:
5.         return error "Circuit breaker is open"

6. try:
7.     result ← operation()
8.     circuitBreaker.successCount ← circuitBreaker.successCount + 1

9.     if circuitBreaker.state == CircuitStateHalfOpen:
10.        if circuitBreaker.successCount >= circuitBreaker.successThreshold:
11.            circuitBreaker.state ← CircuitStateClosed
12.            circuitBreaker.failureCount ← 0
13.            circuitBreaker.successCount ← 0

14.    return result

15. catch error as e:
16.    circuitBreaker.failureCount ← circuitBreaker.failureCount + 1
17.    circuitBreaker.lastFailureTime ← currentTime

18.    if circuitBreaker.failureCount >= circuitBreaker.failureThreshold:
19.        circuitBreaker.state ← CircuitStateOpen
20.        circuitBreaker.successCount ← 0

21.    return error e
```

### 2.4 HTTP Status Code Retry Determination

```
Algorithm: IsStatusRetryable(statusCode, policy)
Input: statusCode, policy
Output: boolean

1. for each code in policy.RetryableStatuses:
2.     if statusCode == code:
3.         return true

4. if statusCode >= 500:  // Server errors
5.     return true

6. if statusCode == 429:  // Too Many Requests
7.     return true

8. if statusCode == 408:  // Request Timeout
9.     return true

10. return false
```

### 2.5 Frontend Retry Hook Flow

```
Algorithm: useRetry(config)
Input: config (RetryConfig)
Output: { execute, state, reset }

1. state ← RetryState {
2.     attempt: 0,
3.     isRetrying: false,
4.     lastError: null,
5.     nextRetryAt: null,
6.     totalDuration: 0
7. }

8. function execute<T>(operation: () => Promise<T>): Promise<RetryResult<T>>:
9.     startTime ← performance.now()
10.    state.attempt ← 0
11.    state.isRetrying ← true

12.    result ← ExecuteWithRetry(async () => {
13.        state.attempt ← state.attempt + 1
14.        return await operation()
15.    }, config.policy, config.timeout)

16.    state.isRetrying ← false
17.    state.totalDuration ← performance.now() - startTime

18.    if result.Success:
19.        config.onSuccess?.(state.attempt, state.totalDuration)
20.        return { success: true, data: result.FinalValue, attempts: state.attempt, totalDuration: state.totalDuration }
21.    else:
22.        state.lastError ← result.LastError
23.        config.onFinalFailure?.(result.LastError, state.attempt)
24.        return { success: false, error: result.LastError, attempts: state.attempt, totalDuration: state.totalDuration }

25. function reset():
26.     state ← initial state
27.     return state

28. return { execute, state, reset }
```

---

## 3. State Management & Error Handling

### 3.1 Error States

| State | Condition | Transition |
|-------|-----------|------------|
| Idle | No retry in progress | → Retrying when `execute()` called |
| Retrying | Retry operation in progress | → Success when operation succeeds<br>→ Failed when all attempts exhausted or non-retryable error |
| Success | Operation completed successfully | → Idle when `reset()` called |
| Failed | All retries exhausted | → Idle when `reset()` called |

### 3.2 Non-Retryable Errors

- **400 Bad Request** - Client error, should not retry
- **401 Unauthorized** - Auth failure, should not retry (refresh token flow instead)
- **403 Forbidden** - Permission denied, should not retry
- **404 Not Found** - Resource doesn't exist, should not retry
- **422 Unprocessable Entity** - Validation error, should not retry
- **Context cancellation** - User cancelled the request
- **Timeout exceeded** - Overall operation timeout reached

### 3.3 Retryable Errors

- **500 Internal Server Error** - Server-side transient failure
- **502 Bad Gateway** - Upstream service unavailable
- **503 Service Unavailable** - Service temporarily overloaded
- **504 Gateway Timeout** - Upstream timeout
- **429 Too Many Requests** - Rate limited (respect Retry-After header)
- **Network timeout** - Connection timeout
- **Network disconnection** - Connection refused/dropped
- **DNS resolution failure** - Temporary DNS issue

### 3.4 Graceful Degradation Integration

```
State: FeatureFlag = "search_enabled" = true

If RetryManager fails for search operation after max attempts:
1. Set FeatureFlag.search_enabled = false
2. Update UI to show degraded state for search
3. Show user-friendly message: "Search is temporarily unavailable"
4. Queue operation for background retry when connectivity restored
5. Set retry timer for 30 seconds to attempt recovery
```

### 3.5 Timeout Handling

- **Per-attempt timeout**: Each retry attempt has its own timeout
- **Overall timeout**: Combined time for all retries must not exceed limit
- **10-second rule**: As per ARCH-017, show timeout notification after 10 seconds if no response
- **Manual retry option**: Always offer manual retry button after timeout

---

## 4. Component Interfaces

### 4.1 Backend Go Interfaces

```go
package retry

type RetryableOperation func() (any, error)

// RetryManager handles retry logic with exponential backoff and circuit breaker
type RetryManager struct {
    policy          RetryPolicy
    circuitBreaker  *CircuitBreaker
    metrics         *MetricsRecorder
}

func NewRetryManager(policy RetryPolicy, opts ...Option) *RetryManager

func (rm *RetryManager) Execute(ctx context.Context, op RetryableOperation) *RetryResult

func (rm *RetryManager) ExecuteWithCircuitBreaker(ctx context.Context, op RetryableOperation) *RetryResult

func (rm *RetryManager) WithCircuitBreaker(failureThreshold, successThreshold int, timeout time.Duration) *RetryManager

func (rm *RetryManager) WithMetrics(metrics *MetricsRecorder) *RetryManager

func (rm *RetryManager) GetMetrics() RetryMetrics

// RetryPolicy builder methods
func WithMaxAttempts(n int) Option
func WithInitialDelay(d time.Duration) Option
func WithMaxDelay(d time.Duration) Option
func WithMultiplier(m float64) Option
func WithJitter(enabled bool) Option
func WithRetryableStatuses(statuses []int) Option

// HTTP-specific helpers
func DefaultHTTPRetryPolicy() RetryPolicy
func IsHTTPRetryable(statusCode int, body []byte) bool
```

### 4.2 Backend Fiber Middleware

```go
package middleware

import "github.com/gofiber/fiber/v2"

func RetryMiddleware(policy retry.RetryPolicy) fiber.Handler

func NewRetryConfig() RetryConfig
```

### 4.3 Frontend Svelte Stores

```typescript
// src/lib/stores/retry.ts
import { writable, derived } from 'svelte/store';

export interface RetryStoreState {
  isRetrying: boolean;
  attempt: number;
  lastError: Error | null;
  nextRetryAt: number | null;
}

function createRetryStore() {
  const { subscribe, set, update } = writable<RetryStoreState>({
    isRetrying: false,
    attempt: 0,
    lastError: null,
    nextRetryAt: null
  });

  return {
    subscribe,
    startRetry: (nextRetryAt: number) => update(s => ({ ...s, isRetrying: true, nextRetryAt })),
    incrementAttempt: () => update(s => ({ ...s, attempt: s.attempt + 1 })),
    setError: (error: Error) => update(s => ({ ...s, lastError: error, isRetrying: false })),
    setSuccess: () => update(s => ({ ...s, isRetrying: false, attempt: 0, lastError: null, nextRetryAt: null })),
    reset: () => set({ isRetrying: false, attempt: 0, lastError: null, nextRetryAt: null })
  };
}

export const retryStore = createRetryStore();
```

### 4.4 Frontend React-like Hook (for Svelte)

```typescript
// src/lib/hooks/useRetry.ts
import { get } from 'svelte/store';
import { retryStore } from '../stores/retry';

export function useRetry<T>(
  operation: () => Promise<T>,
  config: RetryConfig
): {
  execute: () => Promise<RetryResult<T>>;
  state: typeof retryStore;
  reset: () => void;
} {
  const state = retryStore;

  const execute = async (): Promise<RetryResult<T>> => {
    state.startRetry(Date.now() + config.policy.initialDelayMs);
    
    const result = await executeWithRetry(operation, config);
    
    if (result.success) {
      state.setSuccess();
    } else {
      state.setError(result.error);
    }
    
    return result;
  };

  const reset = () => state.reset();

  return { execute, state, reset };
}
```

### 4.5 TanStack Query Integration

```typescript
// Built-in retry behavior via TanStack Query
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: (failureCount, error) => {
        if (error instanceof NetworkError) return true;
        if (error instanceof TimeoutError) return true;
        if (error instanceof ServerError && error.status >= 500) return true;
        return false;
      },
      retryDelay: (attemptIndex) => {
        return Math.min(1000 * 2 ** attemptIndex, 30000);
      },
      retryOnMount: true,
      refetchOnWindowFocus: true
    }
  }
});
```

### 4.6 Error Message Mapping

```go
// Maps retryable errors to user-friendly messages
func MapRetryErrorToUserMessage(err error) string {
    switch {
    case errors.Is(err, context.DeadlineExceeded):
        return "The request timed out. Please try again."
    case errors.Is(err, ErrNetworkUnavailable):
        return "Network connection lost. Retrying..."
    case errors.Is(err, ErrServerUnavailable):
        return "Service temporarily unavailable. Retrying..."
    case errors.Is(err, ErrRateLimited):
        return "Too many requests. Please wait a moment."
    default:
        return "Something went wrong. Retrying..."
    }
}
```

---

## 5. Configuration Defaults

### Backend (Go)

```go
var DefaultRetryPolicy = RetryPolicy{
    MaxAttempts:       3,
    InitialDelay:      100 * time.Millisecond,
    MaxDelay:          5 * time.Second,
    Multiplier:        2.0,
    Jitter:            true,
    RetryableStatuses: []int{500, 502, 503, 504},
}

var SearchRetryPolicy = RetryPolicy{
    MaxAttempts:       5,
    InitialDelay:      200 * time.Millisecond,
    MaxDelay:          10 * time.Second,
    Multiplier:        1.5,
    Jitter:            true,
    RetryableStatuses: []int{500, 502, 503, 504, 429},
}
```

### Frontend (TypeScript)

```typescript
const defaultRetryPolicy: RetryPolicy = {
  maxAttempts: 3,
  initialDelayMs: 1000,
  maxDelayMs: 30000,
  multiplier: 2,
  jitter: true,
  retryableStatusCodes: [500, 502, 503, 504]
};

const searchRetryPolicy: RetryPolicy = {
  maxAttempts: 5,
  initialDelayMs: 500,
  maxDelayMs: 10000,
  multiplier: 1.5,
  jitter: true,
  retryableStatusCodes: [429, 500, 502, 503, 504]
};
```

---

## 6. Metrics & Monitoring

### Backend Metrics

```go
type RetryMetrics struct {
    TotalRetries        int64
    SuccessfulRetries   int64
    FailedRetries       int64
    AverageAttempts     float64
    AverageLatency      time.Duration
    LastRetryTime       time.Time
    CircuitBreakerTrips int64
}

func (m *MetricsRecorder) RecordRetry(attempt int, success bool, latency time.Duration)
func (m *MetricsRecorder) RecordCircuitBreakerTrip()
func (m *MetricsRecorder) GetMetrics() RetryMetrics
```

### Frontend Telemetry

```typescript
interface RetryTelemetry {
  operationName: string;
  attemptCount: number;
  success: boolean;
  totalDurationMs: number;
  errorType: string;
  circuitBreakerTriggered?: boolean;
}

function trackRetry(telemetry: RetryTelemetry) {
  // Send to analytics/monitoring
  analytics.track('retry_execution', telemetry);
}
```
