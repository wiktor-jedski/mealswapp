## FILE: CORSHandler.md
**Traceability:** ARCH-010

### 1. Data Structures & Types

```go
package middleware

type CORSConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	MaxAge           int
	AllowCredentials bool
	AllowOriginFunc  func(origin string) bool
}

type CORSHandler struct {
	config CORSConfig
}
```

### 2. Logic & Algorithms (Step-by-Step)

**Algorithm: HandleCORSRequest**

```
1. Extract the "Origin" header from the incoming HTTP request
   - If no Origin header exists, skip CORS processing and continue to next handler

2. Validate the origin against allowed origins
   a. If AllowOriginFunc is defined, call it with the origin value
      - If returns false, skip CORS processing and continue to next handler
   b. Otherwise, check if origin exists in AllowOrigins slice
      - If "*" in AllowOrigins, allow any origin
      - If origin not found in AllowOrigins, skip CORS processing and continue

3. Set "Access-Control-Allow-Origin" response header to the validated origin

4. If request method is OPTIONS (preflight request):
   a. Extract "Access-Control-Request-Method" header from request
   b. Extract "Access-Control-Request-Headers" header from request
   c. Validate the requested method against AllowMethods
      - If requested method not in AllowMethods, return 403 Forbidden
   d. Validate requested headers against AllowHeaders
      - If any requested header not in AllowHeaders, return 403 Forbidden
   e. Set "Access-Control-Allow-Methods" header to AllowMethods joined by comma
   f. Set "Access-Control-Allow-Headers" header to AllowHeaders joined by comma
   g. If MaxAge > 0, set "Access-Control-Max-Age" header to MaxAge as string
   h. If AllowCredentials is true, set "Access-Control-Allow-Credentials" to "true"
   i. Return 204 No Content with CORS headers, skip further processing

5. For non-OPTIONS requests:
   a. If AllowCredentials is true, set "Access-Control-Allow-Credentials" to "true"
   b. If ExposeHeaders is non-empty, set "Access-Control-Expose-Headers" header
   c. Continue to next handler in the chain
```

**Algorithm: InitializeDefaultConfig**

```
1. Set AllowOrigins to []string{"*"} (wildcard, allow all)
2. Set AllowMethods to default HTTP methods:
   - GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD
3. Set AllowHeaders to default headers:
   - Origin, Content-Type, Accept, Authorization, X-Requested-With
4. Set ExposeHeaders to empty slice (no headers exposed by default)
5. Set MaxAge to 0 (browser default)
6. Set AllowCredentials to false
7. Set AllowOriginFunc to nil
```

### 3. State Management & Error Handling

**Error States:**

| Error Condition | HTTP Status | Response Body | Transition |
| :--- | :--- | :--- | :--- |
| Preflight with invalid method | 403 Forbidden | JSON error response | Request terminated, no further processing |
| Preflight with invalid headers | 403 Forbidden | JSON error response | Request terminated, no further processing |
| Invalid origin (no match) | 200 OK (no CORS headers) | Normal response | Continue to next handler without CORS headers |
| Missing Origin header | 200 OK (no CORS headers) | Normal response | Continue to next handler without CORS headers |

**State Transitions:**

```
State: Idle
  └── Incoming request arrives
      ├── Has Origin header?
      │   ├── No → Transition to: NoCORS (continue chain without CORS)
      │   └── Yes → Transition to: OriginValidation
          
State: OriginValidation
  └── Validate origin against config
      ├── Origin invalid → Transition to: NoCORS (continue chain without CORS)
      ├── Origin valid → Transition to: HeaderProcessing
          
State: HeaderProcessing
  └── Check request method
      ├── OPTIONS (preflight) → Transition to: PreflightProcessing
      └── Other methods → Transition to: SimpleRequestProcessing
          
State: PreflightProcessing
  └── Validate preflight headers
      ├── Method invalid → Return 403 Forbidden
      ├── Headers invalid → Return 403 Forbidden
      └── All valid → Set headers, return 204 No Content
          
State: SimpleRequestProcessing
  └── Set response headers
      └── Continue to next handler
          
State: NoCORS
  └── Continue to next handler (no CORS headers set)
```

**Logging:**

- Log when origin validation fails (debug level)
- Log preflight requests (info level)
- Log preflight validation failures (warn level)
- Log CORS configuration errors at initialization (error level)

### 4. Component Interfaces

**Function: NewCORSHandler**

```go
func NewCORSHandler(config CORSConfig) *CORSHandler
```

**Parameters:**
- `config CORSConfig` - Configuration for CORS behavior

**Returns:**
- `*CORSHandler` - Initialized CORS handler instance

**Behavior:**
- Validates config values
- Sets default values for any zero-value fields
- Returns configured handler ready for middleware registration

---

**Function: Handle**

```go
func (h *CORSHandler) Handle(c *fiber.Ctx) error
```

**Parameters:**
- `c *fiber.Ctx` - Fiber context containing the HTTP request and response

**Returns:**
- `error` - nil on success, fiber error on failure

**Behavior:**
- Implements the HandleCORSRequest algorithm
- Returns appropriate HTTP responses for preflight requests
- Continues to next handler for non-CORS or simple requests

---

**Function: Config**

```go
func (h *CORSHandler) Config() CORSConfig
```

**Returns:**
- `CORSConfig` - Current configuration of the handler

**Behavior:**
- Returns a copy of the current configuration for inspection or modification

---

**Function: DefaultConfig**

```go
func DefaultConfig() CORSConfig
```

**Returns:**
- `CORSConfig` - Default CORS configuration with permissive settings

**Behavior:**
- Returns a CORSConfig with AllowOrigins set to wildcard, standard methods and headers configured

---

**Middleware Registration:**

```go
app.Use(middleware.NewCORSHandler(middleware.DefaultConfig()))
```

**Example Usage with Custom Configuration:**

```go
config := middleware.CORSConfig{
    AllowOrigins: []string{
        "https://mealswapp.com",
        "https://www.mealswapp.com",
    },
    AllowMethods: []string{"GET", "POST", "PUT", "DELETE"},
    AllowHeaders: []string{"Origin", "Content-Type", "Authorization"},
    AllowCredentials: true,
    ExposeHeaders: []string{"Content-Length"},
    MaxAge: 3600,
}

app.Use(middleware.NewCORSHandler(config))
```
