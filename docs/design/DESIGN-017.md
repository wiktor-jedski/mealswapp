## FILE: DESIGN-017.md
**Traceability:** ARCH-017

**Static aspects covered:** ErrorBoundary, GlobalExceptionHandler, RetryManager, ErrorMessageMapper.

### 0. Static Aspect Responsibilities
- `ErrorBoundary`: owns client-side feature isolation, fallback rendering, and state preservation.
- `GlobalExceptionHandler`: owns server panic/error conversion to consistent HTTP envelopes.
- `RetryManager`: owns retry policies, retry timing, and connectivity-restoration retries.
- `ErrorMessageMapper`: owns user-safe text for technical errors and log severity mapping.

### 1. Data Structures & Types
- `type ErrorCategory = "validation" | "auth" | "entitlement" | "network" | "timeout" | "server" | "dependency" | "unknown"`
- `interface AppError { category: ErrorCategory; code: string; message: string; retryable: boolean; requestId?: string; cause?: unknown }`
- `interface RetryPolicy { maxAttempts: number; baseDelayMs: number; maxDelayMs: number; jitterMs: number; retryableCategories: ErrorCategory[] }`
- `interface DegradedFeature { name: string; component: string; reason: string; active: boolean; since: time.Time }`
- `interface ErrorMessageRule { code: string; category: ErrorCategory; userMessage: string; logLevel: "info" | "warn" | "error" }`
- `interface ErrorBoundaryState { hasError: boolean; error?: AppError; degradedFeatures: DegradedFeature[] }`

### 2. Logic & Algorithms (Step-by-Step)
1. Client `ErrorBoundary` catches rendering and async action failures from ARCH-001.
2. Server `GlobalExceptionHandler` converts panics and returned errors into consistent HTTP JSON envelopes.
3. Classify errors by source: validation, auth, entitlement, network, timeout, server, dependency, or unknown.
4. Map technical error codes to user-safe messages without stack traces or infrastructure names.
5. For network failures, preserve current client state and register retry on connectivity restoration.
6. For 10-second timeouts, show timeout notification and expose manual retry.
7. Retry external API calls according to Appendix A policies; fail fast for database and Redis operations where fallback exists.
8. Mark non-critical features as degraded when history sync, recommendations, LP optimization, or similarity indicators fail.
9. Log all server errors through ARCH-014 and security-relevant errors through ARCH-013 audit logging.

### 3. State Management & Error Handling
- `normal`: no active error.
- `recoverable_error`: user can retry; state is preserved.
- `degraded`: feature flag disables non-critical behavior while core search/auth remains available.
- `fatal_client_error`: boundary renders fallback UI for the failed view only.
- `fatal_server_error`: return generic 500 envelope with request ID.
- `timeout`: retryable with preserved request payload where safe.
- `offline_waiting`: retry is queued until browser online event fires.
- `unknown_error`: generic message, error-level log, no technical detail in response.

### 4. Component Interfaces
- `function classifyClientError(error: unknown): AppError`
- `function mapErrorMessage(error: AppError): string`
- `function shouldRetry(error: AppError, policy: RetryPolicy, attempt: number): boolean`
- `function nextRetryDelay(policy: RetryPolicy, attempt: number): number`
- `function markFeatureDegraded(feature: DegradedFeature): void`
- `func GlobalExceptionHandler(ctx *fiber.Ctx, err error) error`
- `func ClassifyServerError(err error) AppError`
- `func WriteErrorResponse(ctx *fiber.Ctx, err AppError) error`
- `func RetryExternal(ctx context.Context, policy RetryPolicy, fn func() error) error`
