## FILE: DESIGN-010.md
**Traceability:** ARCH-010

**Static aspects covered:** RouteHandler, RateLimiter, SecurityHeaderMiddleware, CSRFValidator, RequestValidator, CORSHandler.

### 0. Static Aspect Responsibilities
- `RouteHandler`: owns versioned route registration and direct service-handler dispatch.
- `RateLimiter`: owns IP, user, and endpoint rate limits using Fiber middleware.
- `SecurityHeaderMiddleware`: owns CSP, frame, content type, referrer, and permissions headers.
- `CSRFValidator`: owns synchronizer token validation on state-changing requests.
- `RequestValidator`: owns route-specific body, query, and path validation.
- `CORSHandler`: owns allowed origins, methods, headers, and credential behavior.

### 1. Data Structures & Types
- `interface RouteDefinition { method: string; path: string; version: "v1"; handler: fiber.Handler; middlewares: fiber.Handler[] }`
- `interface RateLimitRule { keyScope: "ip" | "user" | "endpoint"; maxRequests: number; windowSeconds: number }`
- `interface RequestValidationRule { route: string; bodySchema?: string; querySchema?: string; requiresCSRF: boolean; requiresAuth: boolean }`
- `interface GatewayContext { requestId: string; userId?: UUID; startedAt: time.Time; deadline: time.Time }`
- `interface SecurityHeaders { contentSecurityPolicy: string; frameOptions: "DENY"; contentTypeOptions: "nosniff"; referrerPolicy: string; permissionsPolicy: string }`

### 2. Logic & Algorithms (Step-by-Step)
1. Build Fiber app with global request ID, logger, recovery, timeout, security header, CORS, CSRF, and rate-limit middleware.
2. Route only versioned paths such as `/api/v1/search`, `/api/v1/auth/login`, and `/api/v1/jobs/:id`.
3. For every request, attach `GatewayContext` with a 10-second deadline.
4. Apply CORS rules before auth but after request ID assignment.
5. Validate CSRF synchronizer token for POST, PUT, PATCH, and DELETE routes.
6. Enforce endpoint-specific rate limits, including failed login limits per IP.
7. Validate request bodies and query parameters before calling service handlers.
8. Route valid requests to backend service controllers by direct function call.
9. On timeout, cancel request context and return 504 with security headers still attached.

### 3. State Management & Error Handling
- `accepted`: request passed gateway checks and is routed.
- `bad_request`: validation failed; return 400.
- `unauthorized`: missing or invalid auth context; return 401.
- `forbidden`: authenticated but not allowed; return 403.
- `csrf_failed`: return 403 and audit event.
- `rate_limited`: return 429 with retry metadata.
- `timeout`: return 504 after 10 seconds.
- `panic_recovered`: return 500, log stack internally, expose generic message.

### 4. Component Interfaces
- `func NewRouter(deps ServiceDependencies) (*fiber.App, error)`
- `func RegisterV1Routes(app *fiber.App, deps ServiceDependencies)`
- `func SecurityHeadersMiddleware(config SecurityHeaders) fiber.Handler`
- `func TimeoutMiddleware(timeout time.Duration) fiber.Handler`
- `func ValidateRequest(rule RequestValidationRule) fiber.Handler`
- `func CORSHandler(config CORSConfig) fiber.Handler`
- `func RateLimiter(rule RateLimitRule) fiber.Handler`
- `func ExtractGatewayContext(ctx *fiber.Ctx) GatewayContext`
