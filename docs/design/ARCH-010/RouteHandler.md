## FILE: RouteHandler.md
**Traceability:** ARCH-010

### 1. Data Structures & Types

```go
package handler

import (
	"github.com/gofiber/fiber/v2"
)

// RouteConfig holds configuration for a single route
type RouteConfig struct {
	Method      string
	Path        string
	Handler     fiber.Handler
	Middlewares []fiber.Handler
	RateLimit   RateLimitConfig
	Timeout     time.Duration
}

// RateLimitConfig defines rate limiting parameters per route
type RateLimitConfig struct {
	MaxRequests int
	Window      time.Duration
}

// APIRouteRegistry stores all registered API routes
type APIRouteRegistry struct {
	v1Routes map[string]RouteConfig
	v2Routes map[string]RouteConfig
}

// RouteMatchResult contains the result of route matching
type RouteMatchResult struct {
	Found      bool
	Version    string
	Handler    fiber.Handler
	Middlewares []fiber.Handler
	Timeout    time.Duration
	RateLimit  RateLimitConfig
}

// RouteHandler is the main router component
type RouteHandler struct {
	app             *fiber.App
	registry        *APIRouteRegistry
	middlewareChain *MiddlewareChain
	timeoutConfig   time.Duration
}
```

### 2. Logic & Algorithms (Step-by-Step)

**Initialize RouteHandler:**
1. Create new Fiber app instance with custom config
2. Initialize empty APIRouteRegistry for v1 and v2 routes
3. Register built-in middleware: CORS, SecurityHeaders, CSRF, Timeout
4. Register API version prefixes (/api/v1/, /api/v2/)
5. Return configured RouteHandler instance

**RegisterRoute(config RouteConfig):**
1. Validate route config (method, path, handler required)
2. Determine API version from path prefix
3. Store route config in appropriate registry (v1Routes or v2Routes)
4. Apply rate limiting middleware if configured
5. Register route with Fiber app using combined handler chain
6. Return error if registration fails

**Route Matching Process:**
1. Extract request path and method from incoming HTTP request
2. Remove API version prefix from path for matching
3. Look up route in version-specific registry using method + path key
4. If not found in v1, check v2 registry
5. If found, construct handler chain: rateLimiter → timeout → routeHandler
6. If not found, return 404 response

**Handler Chain Execution:**
1. Execute rate limiter middleware (reject if exceeded)
2. Execute timeout middleware (abort with 504 if exceeded)
3. Execute route-specific middlewares
4. Execute final route handler
5. Return response through chain back to client

**API Version Routing:**
```
Request: GET /api/v1/users/123
  → Extract version: v1
  → Extract path: /users/123
  → Match against v1Routes["GET:/users/123"]
  → Execute handler chain

Request: GET /api/v2/orders
  → Extract version: v2
  → Extract path: /orders
  → Match against v2Routes["GET:/orders"]
  → Execute handler chain
```

### 3. State Management & Error Handling

**Error States:**

| Error | Condition | Response | Handling |
| :--- | :--- | :--- | :--- |
| RouteNotFound | No matching route in registry | 404 | Return error response |
| RateLimitExceeded | Requests exceed limit | 429 | Reject with Retry-After header |
| RouteTimeout | Handler exceeds timeout | 504 | Return timeout error |
| InvalidRouteConfig | Missing required fields | panic | Validate during registration |
| MiddlewarePanic | Handler/middleware panics | 500 | Fiber recovery middleware |

**State Transitions:**

```
Idle → RouteRegistered → RequestReceived → (RateLimitCheck) → (TimeoutCheck) → HandlerExecution → ResponseSent

RateLimitExceeded: RequestReceived → Rejected(429)
Timeout: HandlerExecution → Timeout(504)
Panic: AnyState → Recovered(500)
```

**Recovery Handling:**
1. Register Fiber recovery middleware at top of chain
2. Catch all panics during handler execution
3. Log error with request context
4. Return standardized 500 response
5. Continue chain execution for next request

### 4. Component Interfaces

```go
// NewRouteHandler creates a new RouteHandler instance
func NewRouteHandler(timeout time.Duration, globalRateLimit RateLimitConfig) *RouteHandler

// RegisterRoute adds a new route to the registry
func (rh *RouteHandler) RegisterRoute(config RouteConfig) error

// RegisterRoutes registers multiple routes at once
func (rh *RouteHandler) RegisterRoutes(configs []RouteConfig) []error

// GetRoute returns route configuration for a given path and method
func (rh *RouteHandler) GetRoute(version, method, path string) RouteMatchResult

// ListRoutes returns all registered routes for a version
func (rh *RouteHandler) ListRoutes(version string) []RouteConfig

// ServeHTTP implements http.Handler interface
func (rh *RouteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request)

// Start initializes the Fiber app and begins listening
func (rh *RouteHandler) Start(addr string) error

// Shutdown gracefully shuts down the router
func (rh *RouteHandler) Shutdown() error
```

**Middleware Registration Functions:**

```go
// RegisterGlobalMiddleware adds middleware to all routes
func (rh *RouteHandler) RegisterGlobalMiddleware(middleware fiber.Handler)

// RegisterVersionMiddleware adds middleware to specific API version
func (rh *RouteHandler) RegisterVersionMiddleware(version string, middleware fiber.Handler)

// RegisterRateLimiter configures rate limiting for a route
func (rh *RouteHandler) RegisterRateLimiter(routeKey string, config RateLimitConfig) error
```

**Error Response Factory:**

```go
// NewErrorResponse creates a standardized error response
func NewErrorResponse(ctx *fiber.Ctx, status int, code string, message string) error
```
