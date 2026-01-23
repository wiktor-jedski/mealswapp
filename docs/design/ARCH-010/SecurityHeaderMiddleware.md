# SecurityHeaderMiddleware

**Traceability:** ARCH-010

## 1. Data Structures & Types

```go
package middleware

type SecurityHeaderConfig struct {
    CSP           string
    XFrameOptions string
    XContentTypeOptions string
    ReferrerPolicy string
    PermissionsPolicy string
}

type SecurityHeaderMiddleware struct {
    config SecurityHeaderConfig
}

var DefaultSecurityHeaderConfig = SecurityHeaderConfig{
    CSP:                "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'",
    XFrameOptions:      "DENY",
    XContentTypeOptions: "nosniff",
    ReferrerPolicy:     "strict-origin-when-cross-origin",
    PermissionsPolicy:   "camera=(), microphone=(), geolocation=()",
}
```

## 2. Logic & Algorithms

**Algorithm: InjectSecurityHeaders**

```
1. Receive HTTP request context (fiber.Ctx)
2. Retrieve current SecurityHeaderMiddleware configuration
3. Set Content-Security-Policy header with configured CSP value
4. Set X-Frame-Options header with configured value
5. Set X-Content-Type-Options header with configured value
6. Set Referrer-Policy header with configured value
7. Set Permissions-Policy header with configured value
8. Set X-XSS-Protection header (legacy support) to "1; mode=block"
9. Set Strict-Transport-Security header (HSTS) with max-age=31536000; includeSubDomains
10. Call next middleware handler in chain
11. Return response to client with injected headers
```

**Detailed Flow:**

1. Initialize SecurityHeaderMiddleware with optional custom config or defaults
2. Register middleware in Fiber app: app.Use(SecurityHeaderMiddleware())
3. Middleware intercepts each incoming request before routing
4. For each request, inject all security headers into response writer
5. Headers are set on response before any body is written
6. Call c.Next() to pass control to subsequent middleware/handlers
7. Headers remain immutable once response headers are committed

## 3. State Management & Error Handling

**Error States:**

| Error Condition | Handling |
| :--- | :--- |
| Header already set by upstream middleware | Skip injection for that specific header, log warning |
| Response already committed | Skip all header injection, continue to next |
| Invalid CSP directive | Use default CSP config, log error |
| Configuration parse error | Fall back to defaults, log error |
| Memory allocation failure | Continue without security headers, log critical error |

**State Transitions:**

```
UNINITIALIZED -> CONFIGURED -> ACTIVE -> COMPLETED
         |                |            |
    Load defaults    Apply config    Process request
```

**Retry Logic:** None. Security headers are best-effort per request.

## 4. Component Interfaces

```go
func NewSecurityHeaderMiddleware(config ...SecurityHeaderConfig) fiber.Handler

func (m *SecurityHeaderMiddleware) Handler() fiber.Handler

func SecurityHeaderMiddleware() fiber.Handler
```

**Function Signature Details:**

```
NewSecurityHeaderMiddleware(config ...SecurityHeaderConfig) fiber.Handler
Parameters:
    - config: Optional SecurityHeaderConfig. If nil, uses DefaultSecurityHeaderConfig
Returns:
    - fiber.Handler: Middleware function for Fiber app registration

Handler() fiber.Handler
Returns:
    - fiber.Handler: Closure that processes requests and injects headers

SecurityHeaderMiddleware() fiber.Handler
Returns:
    - fiber.Handler: Default middleware with built-in configuration
```

**Usage Example:**

```go
app := fiber.New()

customConfig := middleware.SecurityHeaderConfig{
    CSP: "default-src 'self'",
    XFrameOptions: "DENY",
}

app.Use(middleware.NewSecurityHeaderMiddleware(customConfig))
```

**Header List:**

| Header | Default Value | Can Override |
| :--- | :--- | :--- |
| Content-Security-Policy | See DefaultSecurityHeaderConfig | Yes |
| X-Frame-Options | DENY | Yes |
| X-Content-Type-Options | nosniff | Yes |
| Referrer-Policy | strict-origin-when-cross-origin | Yes |
| Permissions-Policy | camera=(), microphone=(), geolocation=() | Yes |
| X-XSS-Protection | 1; mode=block | No |
| Strict-Transport-Security | max-age=31536000; includeSubDomains | No |
