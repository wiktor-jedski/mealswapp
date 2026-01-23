# CSRFValidator (Fiber CSRF Middleware)

**Traceability:** ARCH-013

## 1. Data Structures & Types

### 1.1 Configuration Structure

```go
type CSRFConfig struct {
    // TokenLookup is the method to extract token from request
    // Default: "header:Authorization"
    TokenLookup string

    // CookieName is the name of the CSRF cookie
    // Default: "_csrf"
    CookieName string

    // CookieDomain is the domain that is allowed to receive the cookie
    CookieDomain string

    // CookiePath is the path to set on the cookie
    // Default: "/"
    CookiePath string

    // CookieSecure sets the Secure attribute on the cookie
    // Default: true (when TLS is enabled)
    CookieSecure bool

    // CookieHTTPOnly sets the HttpOnly attribute on the cookie
    // Default: true
    CookieHTTPOnly bool

    // CookieSameSite configures the SameSite attribute
    // Default: fiber.CookieSameSiteStrictMode
    CookieSameSite fiber.CookieSameSiteMode

    // Expiration is the time the CSRF token expires
    // Default: 1 hour
    Expiration time.Duration

    // SkipFailedAuthentication bypasses CSRF validation for failed auth attempts
    // Default: false
    SkipFailedAuthentication bool

    // CookieExtractor extracts CSRF token from cookies
    CookieExtractor func(c *fiber.Ctx) (string, error)
}
```

### 1.2 Context Keys

```go
const (
    // CSRFTokenKey is the context key for storing CSRF token
    CSRFTokenKey = "csrf_token"

    // CSRFTokenValidKey is the context key for CSRF validation result
    CSRFTokenValidKey = "csrf_token_valid"
)
```

### 1.3 Token Structure

```go
// CSRFToken represents a synchronizer token for CSRF protection
type CSRFToken struct {
    Value     string    // The random token value (256-bit random)
    CreatedAt time.Time // Token creation timestamp
    ExpiresAt time.Time // Token expiration timestamp
}
```

## 2. Logic & Algorithms (Step-by-Step)

### 2.1 Middleware Initialization Flow

```
1. Load configuration from Fiber app config or custom config
2. Set default values for unset configuration options:
   - TokenLookup: "header:X-CSRF-Token" or "form:_csrf"
   - CookieName: "_csrf"
   - Expiration: 1 hour
3. Validate configuration parameters
4. Register middleware in Fiber middleware chain
```

### 2.2 Token Generation Algorithm

```
1. Generate cryptographically secure random bytes (32 bytes = 256 bits)
2. Encode to base64 URL-safe string
3. Create CSRFToken struct with:
   - Value: encoded token
   - CreatedAt: current time
   - ExpiresAt: CreatedAt + Expiration
4. Store token in secure session or temporary storage
5. Return token for client consumption
```

### 2.3 Request Validation Flow

```
1. Check if request method is safe (GET, HEAD, OPTIONS):
   - If safe method: skip validation, continue to next handler
   - If unsafe method: proceed to step 2

2. Extract token from request using configured TokenLookup:
   a. Try header extraction (e.g., "X-CSRF-Token" or "Authorization: Bearer <token>")
   b. Try cookie extraction (CookieName)
   c. Try form field extraction (form:_csrf)
   d. If no token found: reject request with 403 Forbidden

3. Validate token structure and format:
   a. Check token is valid base64 URL-encoded string
   b. Verify token length (expected ~44 characters for base64 of 32 bytes)
   c. If invalid format: reject request with 403 Forbidden

4. Retrieve stored token from session or token store:
   a. Get session ID from session cookie
   b. Fetch associated CSRF token from Redis store
   c. If no stored token: reject request with 403 Forbidden

5. Perform token comparison (constant-time):
   a. Use crypto/subtle.ConstantTimeCompare
   b. Compare submitted token with stored token
   c. If mismatch: reject request with 403 Forbidden

6. Check token expiration:
   a. Compare current time with token.ExpiresAt
   b. If token expired: reject request with 403 Forbidden
   c. Optionally: issue new token and continue

7. If all validations pass:
   a. Mark request as CSRF validated in context
   b. Store token in context for downstream handlers
   c. Continue to next handler

8. On response, if token was not validated and request was safe:
   a. Optionally generate new CSRF token
   b. Set CSRF cookie in response
```

### 2.4 Token Storage Flow

```
1. Create unique session identifier (session ID)
2. Store token mapping in Redis:
   Key: "csrf:<session_id>"
   Value: JSON-encoded CSRFToken
   TTL: Configured Expiration duration
3. Set session cookie on client with session identifier
4. Return token to client via response (cookie or header)
```

## 3. State Management & Error Handling

### 3.1 Possible Error States

| Error Condition | HTTP Status | Error Code | User Message |
| :--- | :--- | :--- | :--- |
| Missing CSRF token | 403 | ERR_CSRF_MISSING_TOKEN | "Missing CSRF token" |
| Invalid token format | 403 | ERR_CSRF_INVALID_FORMAT | "Invalid CSRF token format" |
| Token mismatch | 403 | ERR_CSRF_TOKEN_MISMATCH | "CSRF token mismatch" |
| Token expired | 403 | ERR_CSRF_TOKEN_EXPIRED | "CSRF token expired" |
| Token not found in store | 403 | ERR_CSRF_TOKEN_NOT_FOUND | "CSRF token not found" |
| Session not found | 403 | ERR_CSRF_NO_SESSION | "Session not found" |
| Redis connection failure | 500 | ERR_CSRF_STORAGE_ERROR | "CSRF storage unavailable" |
| Token generation failure | 500 | ERR_CSRF_GENERATION_FAILED | "Failed to generate CSRF token" |
| Cookie setting failure | 500 | ERR_CSRF_COOKIE_SET_FAILED | "Failed to set CSRF cookie" |

### 3.2 State Transitions

```
Initial State: NEW_REQUEST

NEW_REQUEST -> CHECK_METHOD
    Transition: Request received, start processing

CHECK_METHOD -> SAFE_METHOD_PROCESSED
    Condition: Request method is GET, HEAD, or OPTIONS
    Action: Skip CSRF validation, continue to handler

CHECK_METHOD -> EXTRACT_TOKEN
    Condition: Request method is POST, PUT, PATCH, DELETE
    Action: Proceed to token extraction

EXTRACT_TOKEN -> TOKEN_MISSING
    Condition: No token found in request
    Action: Return 403 Forbidden

EXTRACT_TOKEN -> VALIDATE_TOKEN
    Condition: Token extracted successfully
    Action: Proceed to token validation

VALIDATE_TOKEN -> TOKEN_INVALID
    Condition: Token format invalid
    Action: Return 403 Forbidden

VALIDATE_TOKEN -> RETRIEVE_STORED_TOKEN
    Condition: Token format valid
    Action: Fetch stored token from Redis

RETRIEVE_STORED_TOKEN -> TOKEN_NOT_FOUND
    Condition: Stored token not found
    Action: Return 403 Forbidden

RETRIEVE_STORED_TOKEN -> COMPARE_TOKENS
    Condition: Stored token retrieved
    Action: Perform constant-time comparison

COMPARE_TOKENS -> TOKEN_MISMATCH
    Condition: Tokens do not match
    Action: Return 403 Forbidden

COMPARE_TOKENS -> CHECK_EXPIRATION
    Condition: Tokens match
    Action: Check token expiration

CHECK_EXPIRATION -> TOKEN_EXPIRED
    Condition: Token is expired
    Action: Return 403 Forbidden (or optionally regenerate)

CHECK_EXPIRATION -> VALIDATED
    Condition: Token is valid
    Action: Mark request as validated, continue to handler

TOKEN_MISSING, TOKEN_INVALID, TOKEN_NOT_FOUND, TOKEN_MISMATCH, TOKEN_EXPIRED
    -> Return 403 Forbidden response
    -> Log security event
    -> Increment CSRF failure counter

VALIDATED -> Response generation
    Action: Continue to next handler middleware
    On completion: Optionally refresh token and set cookie
```

### 3.3 Error Recovery

```
1. Missing Token Recovery:
   - Client can request a new token via GET /csrf-token endpoint
   - Server generates new token and returns in response

2. Expired Token Recovery:
   - Client must obtain new token before making state-changing request
   - Automatic token refresh on safe requests is optional

3. Storage Failure Recovery:
   - Fallback to in-memory token storage (limited scalability)
   - Return 503 Service Unavailable with retry-after header
   - Log critical error for monitoring

4. Token Theft Detection:
   - If token mismatch detected for valid session
   - Invalidate entire session
   - Log security incident
   - Require re-authentication
```

## 4. Component Interfaces

### 4.1 Public Functions

```go
// New creates a new CSRF middleware handler with default configuration
func New() fiber.Handler

// NewWithConfig creates a CSRF middleware handler with custom configuration
func NewWithConfig(config CSRFConfig) fiber.Handler

// ExtractToken extracts CSRF token from request using configured lookup method
func (c *CSRFConfig) ExtractToken(ctx *fiber.Ctx) (string, error)

// GenerateToken generates a new cryptographically secure CSRF token
func (c *CSRFConfig) GenerateToken(ctx *fiber.Ctx) (string, error)

// ValidateToken validates a CSRF token against stored token
func (c *CSRFConfig) ValidateToken(ctx *fiber.Ctx, token string) bool

// Protected returns a middleware that requires CSRF validation
func Protected() fiber.Handler

// Skip returns a middleware that skips CSRF validation for specific paths
func Skip(paths ...string) fiber.Handler
```

### 4.2 Handler Signature

```go
// Handler is the fiber.Handler function signature for CSRF middleware
// Returns error on validation failure (handled by Fiber error handler)
type Handler = func(*fiber.Ctx) error
```

### 4.3 Configuration Example

```go
// Default configuration
var DefaultConfig = CSRFConfig{
    TokenLookup:              "header:X-CSRF-Token",
    CookieName:               "_csrf",
    CookiePath:               "/",
    CookieDomain:             "",
    CookieSecure:             true,
    CookieHTTPOnly:           true,
    CookieSameSite:           fiber.CookieSameSiteStrictMode,
    Expiration:               1 * time.Hour,
    SkipFailedAuthentication: false,
    CookieExtractor:          nil,
}

// Production configuration
var ProductionConfig = CSRFConfig{
    TokenLookup:              "header:X-CSRF-Token",
    CookieName:               "_csrf",
    CookiePath:               "/",
    CookieDomain:             "mealswapp.com",
    CookieSecure:             true,
    CookieHTTPOnly:           true,
    CookieSameSite:           fiber.CookieSameSiteStrictMode,
    Expiration:               30 * time.Minute,
    SkipFailedAuthentication: false,
}
```

### 4.4 Usage Example

```go
// Basic usage
app.Use(csrf.New())

// Custom configuration
app.Use(csrf.NewWithConfig(CSRFConfig{
    TokenLookup:   "header:X-CSRF-Token",
    CookieName:    "_csrf",
    Expiration:    45 * time.Minute,
    CookieSecure:  true,
    CookieHTTPOnly: true,
    CookieSameSite: fiber.CookieSameSiteStrictMode,
}))

// Skip CSRF for specific paths
app.Use("/api/public/", csrf.New())
app.Use("/api/public/", func(c *fiber.Ctx) error {
    return c.Next()
})

// Get CSRF token in handler
app.Get("/csrf-token", func(c *fiber.Ctx) error {
    token := c.Locals(csrf.CSRFTokenKey).(string)
    return c.JSON(fiber.Map{"token": token})
})
```

### 4.5 Dependencies

```
External:
- github.com/gofiber/fiber/v2 (v2.x)
- github.com/gofiber/csrf (Fiber CSRF middleware package)

Internal:
- Session management via Fiber session middleware
- Redis storage for token persistence
- Logging via Fiber logger middleware
- Error handling via Fiber error handler
```
