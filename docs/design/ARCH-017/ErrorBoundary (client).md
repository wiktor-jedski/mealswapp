# ErrorBoundary (client)

**Traceability:** ARCH-017

## 1. Data Structures & Types

```typescript
interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
  errorInfo: ErrorInfo | null;
  isRetrying: boolean;
  retryCount: number;
  errorCode: string | null;
}

type ErrorSeverity = 'critical' | 'warning' | 'info';

interface MappedError {
  userMessage: string;
  severity: ErrorSeverity;
  showRetry: boolean;
  isRecoverable: boolean;
  fallbackUI?: string;
}

interface ErrorInfo {
  componentStack: string;
  timestamp: Date;
  url: string;
  userAgent: string;
}

interface RetryConfig {
  maxRetries: number;
  backoffMs: number;
  maxBackoffMs: number;
}

interface ErrorBoundaryProps {
  children: import('svelte').Snippet;
  fallback?: import('svelte').Snippet<[MappedError, () => void]>;
  onError?: (error: Error, errorInfo: ErrorInfo) => void;
  errorBoundaryId?: string;
  featureArea?: 'core' | 'non-critical';
}

const defaultRetryConfig: RetryConfig = {
  maxRetries: 3,
  backoffMs: 1000,
  maxBackoffMs: 10000,
};

const errorMapping: Record<string, MappedError> = {
  'NETWORK_ERROR': {
    userMessage: 'Unable to connect. Please check your internet connection.',
    severity: 'warning',
    showRetry: true,
    isRecoverable: true,
  },
  'TIMEOUT_ERROR': {
    userMessage: 'The request timed out. Please try again.',
    severity: 'warning',
    showRetry: true,
    isRecoverable: true,
  },
  'AUTH_ERROR': {
    userMessage: 'Your session has expired. Please log in again.',
    severity: 'critical',
    showRetry: false,
    isRecoverable: true,
  },
  'PERMISSION_DENIED': {
    userMessage: 'You do not have permission to perform this action.',
    severity: 'critical',
    showRetry: false,
    isRecoverable: false,
  },
  'VALIDATION_ERROR': {
    userMessage: 'The information you provided is invalid. Please check and try again.',
    severity: 'warning',
    showRetry: false,
    isRecoverable: false,
  },
  'SERVER_ERROR': {
    userMessage: 'Something went wrong on our end. Please try again later.',
    severity: 'warning',
    showRetry: true,
    isRecoverable: true,
  },
  'NOT_FOUND': {
    userMessage: 'The requested resource was not found.',
    severity: 'warning',
    showRetry: false,
    isRecoverable: false,
  },
  'UNKNOWN_ERROR': {
    userMessage: 'An unexpected error occurred. Please try again.',
    severity: 'warning',
    showRetry: true,
    isRecoverable: true,
  },
};

class ErrorBoundaryStore {
  hasError = $state(false);
  error = $state<Error | null>(null);
  errorInfo = $state<ErrorInfo | null>(null);
  isRetrying = $state(false);
  retryCount = $state(0);
  errorCode = $state<string | null>(null);

  reset() {
    this.hasError = false;
    this.error = null;
    this.errorInfo = null;
    this.isRetrying = false;
    this.retryCount = 0;
    this.errorCode = null;
  }
}
```

## 2. Logic & Algorithms (Step-by-Step)

```
ALGORITHM: ErrorBoundary Constructor and Initialization
1. Initialize ErrorBoundaryStore with default values
2. Set up signal tracking for child component updates
3. Register error boundary in global error registry (if errorBoundaryId provided)

ALGORITHM: Error Detection and Capture
1. Child component throws error during render/update
2. Catch error in error boundary's error handler
3. Extract error message and stack trace
4. Generate timestamp and capture URL/userAgent
5. Classify error type based on error characteristics:
   IF error.message contains 'fetch' OR 'network' -> NETWORK_ERROR
   IF error.message contains 'timeout' OR 'Timed out' -> TIMEOUT_ERROR
   IF error.code === 'ECONNREFUSED' OR error.message contains 'Connection refused' -> SERVER_ERROR
   IF error.message contains '401' OR 'Unauthorized' -> AUTH_ERROR
   IF error.message contains '403' OR 'Forbidden' -> PERMISSION_DENIED
   IF error.message contains '400' OR 'Validation' -> VALIDATION_ERROR
   IF error.message contains '404' OR 'Not found' -> NOT_FOUND
   ELSE -> UNKNOWN_ERROR
6. Store error details in ErrorBoundaryStore
7. Set hasError = true

ALGORITHM: Error Mapping and User Message Generation
1. Lookup error code in errorMapping dictionary
2. IF error code exists:
   RETURN mapped error object with userMessage, severity, showRetry, isRecoverable
3. ELSE:
   RETURN UNKNOWN_ERROR mapping
4. Store mapped error for fallback component rendering

ALGORITHM: Retry Logic with Exponential Backoff
1. User clicks retry button OR automatic retry triggered
2. IF retryCount >= maxRetries:
   SET isRecoverable = false
   RETURN (do not retry)
3. Calculate backoff delay:
   delay = min(backoffMs * (2 ^ retryCount), maxBackoffMs)
4. SET isRetrying = true
5. WAIT for delay milliseconds
6. RESET error state (hasError = false, error = null)
7. INCREMENT retryCount
8. TRIGGER re-render of child components
9. IF render succeeds:
   SET isRetrying = false
   RESET retryCount to 0
   RETURN success
10. IF render fails:
    REPEAT from step 2 (error caught in error handler)

ALGORITHM: Graceful Degradation Handler
1. IF featureArea === 'non-critical' AND error occurs:
   SET degradedMode = true
   DISPLAY reduced-functionality fallback UI
   LOG error for analytics but DO NOT interrupt main flow
2. IF featureArea === 'core' AND error occurs:
   DISPLAY full error UI with retry options
   PRESERVE application state
   ALLOW navigation to unaffected areas
3. ISOLATE error to prevent propagation to parent boundaries

ALGORITHM: Network Status Integration
1. LISTEN for online/offline events from navigator.onLine
2. IF coming back online AND hasError AND error was NETWORK_ERROR:
   AUTOMATICALLY trigger retry
3. IF going offline AND hasError:
   DISPLAY offline indicator in error UI
4. SYNC retry state with TanStack Query cache invalidation

ALGORITHM: Timeout Handling
1. SET timeout timer when request initiated (10 seconds)
2. IF timer expires BEFORE response:
   THROW TIMEOUT_ERROR
   DISPLAY timeout notification
   OFFER manual retry button
3. IF response arrives before timer:
   CLEAR timeout timer
   CONTINUE normal flow
```

## 3. State Management & Error Handling

### Error States

| State | Condition | Transition |
|-------|-----------|------------|
| `hasError: false` | No error captured | Normal operation |
| `hasError: true` | Error caught in catch block | Error occurred, display fallback |
| `isRetrying: true` | Retry initiated, waiting for backoff | Backoff in progress |
| `isRetrying: false` | Retry complete or not retrying | Ready for next action |
| `retryCount: 0-3` | Current retry attempt number | Incremented on each retry attempt |

### Error Transitions

```
STATE TRANSITION DIAGRAM

[IDLE] --child renders successfully--> [IDLE]
[IDLE] --child throws error---------> [ERROR_DETECTED]
                                    |
                                    v
                            [CLASSIFY_ERROR]
                                    |
                    +---------------+---------------+
                    |               |               |
                    v               v               v
            [NETWORK_ERROR]  [TIMEOUT_ERROR]  [CRITICAL_ERROR]
                    |               |               |
                    v               v               v
            [SHOW_RETRY_UI]  [SHOW_RETRY_UI]  [SHOW_FATAL_UI]
                    |               |               |
          +----+----+               |               |
          |         |               |               |
          v         v               |               |
    [RETRY]    [AUTO_RETRY]         |               |
          |         |               |               |
          +----+----+               |               |
                    |               |               |
                    v               v               v
            [BACKOFF_WAIT]  [BACKOFF_WAIT]     [TERMINAL]
                    |               |               |
          +----+----+               |               |
          |         |               |               |
          v         v               v               v
    [SUCCESS]  [MAX_RETRIES]  [SUCCESS/FAIL]   [REFRESH_PAGE]
```

### Error Classification Matrix

| Error Type | User Message | Retry Option | Auto-Retry | Severity |
|------------|--------------|--------------|------------|----------|
| Network Failure | "Unable to connect" | Yes | On connectivity restore | Warning |
| Timeout (10s) | "Request timed out" | Yes | No | Warning |
| Auth Expiry | "Session expired" | No (redirect to login) | No | Critical |
| Permission Denied | "Access denied" | No | No | Critical |
| Validation Error | "Invalid input" | No | No | Warning |
| Server Error | "Server error" | Yes | Exponential backoff | Warning |
| Not Found | "Resource not found" | No | No | Warning |
| Unknown Error | "Unexpected error" | Yes | No | Warning |

### Error Handling Strategy

- **Network Failure:** Preserve all application state, display retry button, automatically retry when `navigator.onLine` becomes true
- **Timeout Handling:** Start 10-second timer on request initiation, show timeout notification with manual retry option when timer expires
- **Graceful Degradation:** Non-critical feature failures (history sync, recommendations) display reduced UI without affecting core functionality (search, auth)
- **Error Classification:** Map technical errors to user-friendly messages; never expose system internals, stack traces, or error codes to users

## 4. Component Interfaces

```typescript
function ErrorBoundary(
  props: ErrorBoundaryProps
): import('svelte').Component;

interface ErrorBoundaryAPI {
  reset: () => void;
  getState: () => ErrorBoundaryState;
  retry: () => Promise<void>;
  getMappedError: () => MappedError | null;
}

function createErrorBoundary(
  config?: {
    fallback?: import('svelte').Snippet<[MappedError, () => void]>;
    onError?: (error: Error, errorInfo: ErrorInfo) => void;
    errorBoundaryId?: string;
    featureArea?: 'core' | 'non-critical';
    retryConfig?: Partial<RetryConfig>;
  }
): {
  component: import('svelte').Component;
  api: ErrorBoundaryAPI;
};

function useErrorHandler(): {
  captureError: (error: Error, context?: Record<string, unknown>) => void;
  clearError: () => void;
};

function mapErrorToUserMessage(error: Error): MappedError;

async function retryWithBackoff(
  operation: () => Promise<void>,
  config?: Partial<RetryConfig>
): Promise<{ success: boolean; attempts: number }>;

function registerErrorBoundary(
  id: string,
  api: ErrorBoundaryAPI
): void;

function unregisterErrorBoundary(id: string): void;

function getErrorBoundary(id: string): ErrorBoundaryAPI | null;

function isOnline(): boolean;

function setupNetworkListener(
  onOnline: () => void,
  onOffline: () => void
): () => void;
```

### Usage Example

```svelte
<script lang="ts">
import { ErrorBoundary } from '@/components/ErrorBoundary';

function handleCoreError(error: Error, info: ErrorInfo) {
  console.error('Core feature error:', error);
  // Send to analytics
}

function handleNonCriticalError(error: Error, info: ErrorInfo) {
  console.warn('Non-critical feature failed:', error);
  // Log but don't interrupt user flow
}
</script>

<ErrorBoundary
  featureArea="core"
  onError={handleCoreError}
>
  <SearchComponent />
</ErrorBoundary>

<ErrorBoundary
  featureArea="non-critical"
  onError={handleNonCriticalError}
>
  <RecommendationsComponent />
</ErrorBoundary>
```
