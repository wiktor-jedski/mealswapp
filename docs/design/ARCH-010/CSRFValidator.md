## FILE: CSRFValidator.md
**Traceability:** ARCH-010

### 1. Data Structures & Types

```go
package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/csrf"
)

type CSRFConfig struct {
	CookieName     string
	CookieDomain   string
	CookiePath     string
	CookieSecure   bool
	CookieHTTPOnly bool
	CookieSameSite string
	TokenLength    uint8
	Expiration     time.Duration
	Skipper        func(c *fiber.Ctx) bool
	Extractor      func(c *fiber.Ctx) (string, error)
}

type CSRFValidator struct {
	config CSRFConfig
	store  csrf.Store
}

type CSRFToken struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

type ValidationResult struct {
	Valid   bool
	Reason  string
	Code    int
}
```

### 2. Logic & Algorithms (Step-by-Step)

**Algorithm: ValidateCSRFToken**

```
1. INPUT: HTTP Context (c), Request Method (GET, POST, PUT, DELETE, PATCH)
2. OUTPUT: ValidationResult

3. BEGIN
4. IF Skipper(c) returns true THEN
5.     RETURN ValidationResult{Valid: true, Reason: "skipped"}
6. END IF

7. IF Request Method IS in {GET, HEAD, OPTIONS} THEN
8.     RETURN ValidationResult{Valid: true, Reason: "safe method"}
9. END IF

10. token ← ExtractToken(c)
11. IF token IS empty THEN
12.     RETURN ValidationResult{Valid: false, Reason: "missing token", Code: 403}
13. END IF

14. IF ValidateTokenFormat(token) IS false THEN
15.     RETURN ValidationResult{Valid: false, Reason: "invalid token format", Code: 403}
16. END IF

17. IF store.VerifyToken(c, token) IS false THEN
18.     LogCSRFViolation(c, token)
19.     RETURN ValidationResult{Valid: false, Reason: "token mismatch", Code: 403}
20. END IF

21. IF IsTokenExpired(c) THEN
22.     RETURN ValidationResult{Valid: false, Reason: "token expired", Code: 403}
23. END IF

24. RETURN ValidationResult{Valid: true, Reason: "valid"}
25. END
```

**Algorithm: GenerateCSRFToken**

```
1. INPUT: HTTP Context (c)
2. OUTPUT: CSRFToken

3. BEGIN
4. token ← GenerateRandomBytes(config.TokenLength)
5. expiresAt ← Now() + config.Expiration

6. store.SaveToken(c, token, expiresAt)

7. RETURN CSRFToken{
8.     Token:     token,
9.     ExpiresAt: expiresAt
10. }
11. END
```

**Algorithm: ExtractToken**

```
1. INPUT: HTTP Context (c)
2. OUTPUT: string (token) or error

3. BEGIN
4. IF config.Extractor IS NOT nil THEN
5.     RETURN config.Extractor(c)
6. END IF

7. header ← c.Get("X-CSRF-Token")
8. IF header IS NOT empty THEN
9.     RETURN header, nil
10. END IF

11. formToken ← c.FormValue("csrf_token")
12. IF formToken IS NOT empty THEN
13.     RETURN formToken, nil
14. END IF

15. cookie ← c.Cookies(config.CookieName)
16. IF cookie IS NOT empty THEN
17.     RETURN cookie, nil
18. END IF

19. RETURN "", ErrCSRFTokenNotFound
20. END
```

### 3. State Management & Error Handling

**State Transitions:**

| Current State | Event | Next State | Action |
|--------------|-------|------------|--------|
| Initial | Request received | TokenExtraction | Call ExtractToken |
| TokenExtraction | Token missing | Invalid | Return 403, log warning |
| TokenExtraction | Token extracted | TokenValidation | Proceed to validation |
| TokenValidation | Format invalid | Invalid | Return 403, increment failure counter |
| TokenValidation | Store mismatch | Invalid | Return 403, log security event |
| TokenValidation | Token expired | Invalid | Return 403, trigger token refresh |
| TokenValidation | Valid token | Valid | Allow request, update session |

**Error States:**

| Error | HTTP Code | User Message | Logging Level |
|-------|-----------|--------------|---------------|
| ErrCSRFTokenNotFound | 403 | "Missing CSRF token" | Warn |
| ErrCSRFTokenInvalid | 403 | "Invalid CSRF token" | Warn |
| ErrCSRFTokenExpired | 403 | "CSRF token expired" | Warn |
| ErrCSRFTokenMismatch | 403 | "Token validation failed" | Error |
| ErrCSRFExtractorFailed | 500 | "Internal server error" | Error |

**Security Events to Log:**
- Repeated validation failures from same IP
- Token extraction failures
- Expired token attempts
- Missing token on state-changing requests

### 4. Component Interfaces

```go
func NewCSRFValidator(config CSRFConfig) *CSRFValidator

func (v *CSRFValidator) Validate(c *fiber.Ctx) error
// Validates CSRF token on incoming requests
// Returns error if validation fails, nil if passed

func (v *CSRFValidator) GenerateToken(c *fiber.Ctx) error
// Generates and sets new CSRF token
// Returns token in response body or header

func (v *CSRFValidator) RefreshToken(c *fiber.Ctx) error
// Invalidates current token and generates new one
// Useful after successful authentication

func (v *CSRFValidator) ExtractToken(c *fiber.Ctx) (string, error)
// Extracts token from request (header, form, or cookie)

func (v *CSRFValidator) Skipper(c *fiber.Ctx) bool
// Determines if CSRF validation should be skipped for request

func (v *CSRFValidator) Middleware() fiber.Handler
// Returns Fiber middleware handler for easy integration
```

**Fiber Middleware Integration:**

```go
app.Use(middleware.NewCSRFValidator(CSRFConfig{
	CookieName:     "_csrf",
	CookieDomain:   "mealswapp.com",
	CookiePath:     "/",
	CookieSecure:   true,
	CookieHTTPOnly: true,
	CookieSameSite: "Strict",
	TokenLength:    32,
	Expiration:     1 * time.Hour,
}).Middleware())
```

**Configuration Options:**

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| CookieName | string | "_csrf" | Name of CSRF cookie |
| CookieDomain | string | "" | Domain for cookie |
| CookiePath | string | "/" | Path for cookie |
| CookieSecure | bool | true | Require HTTPS |
| CookieHTTPOnly | bool | true | Prevent JS access |
| CookieSameSite | string | "Strict" | SameSite policy |
| TokenLength | uint8 | 32 | Bytes in token |
| Expiration | time.Duration | 1h | Token lifetime |
| Skipper | func | nil | Custom skip logic |
| Extractor | func | nil | Custom token extractor |
