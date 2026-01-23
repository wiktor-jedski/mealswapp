# RequestValidator

**Traceability:** ARCH-010

## 1. Data Structures & Types

```go
package middleware

import (
    "regexp"
    "strconv"
    "strings"

    "github.com/gofiber/fiber/v2"
)

// ValidationRule defines a single validation rule for a parameter
type ValidationRule struct {
    FieldName   string
    FieldType   FieldType
    Required    bool
    MinLength   *int
    MaxLength   *int
    MinValue    *float64
    MaxValue    *float64
    Pattern     *regexp.Regexp
    EnumValues  []string
}

// FieldType specifies the expected type of a field
type FieldType string

const (
    FieldTypeString  FieldType = "string"
    FieldTypeInt     FieldType = "int"
    FieldTypeFloat   FieldType = "float"
    FieldTypeBool    FieldType = "bool"
    FieldTypeEmail   FieldType = "email"
    FieldTypeUUID    FieldType = "uuid"
    FieldTypeDate    FieldType = "date"
    FieldTypeEnum    FieldType = "enum"
)

// ValidationSchema contains all rules for validating a request
type ValidationSchema struct {
    PathParams   []ValidationRule
    QueryParams  []ValidationRule
    BodyParams   []ValidationRule
    Headers      []ValidationRule
}

// ValidationError represents a single field validation error
type ValidationError struct {
    Field   string      `json:"field"`
    Value   interface{} `json:"value"`
    Message string      `json:"message"`
    Code    string      `json:"code"`
}

// ValidationResult contains the outcome of request validation
type ValidationResult struct {
    Valid  bool             `json:"valid"`
    Errors []ValidationError `json:"errors,omitempty"`
}

// RequestValidatorConfig holds configuration for the validator middleware
type RequestValidatorConfig struct {
    // SchemaRegistry maps route patterns to validation schemas
    SchemaRegistry map[string]ValidationSchema
    // SkipPaths contains routes that should skip validation
    SkipPaths []string
    // ErrorHandler custom handler for validation errors
    ErrorHandler fiber.ErrorHandler
    // EnableDebug enables detailed error logging
    EnableDebug bool
}

// RequestContext holds extracted and validated request data
type RequestContext struct {
    PathParams  map[string]interface{}
    QueryParams map[string]interface{}
    BodyParams  map[string]interface{}
    Headers     map[string]interface{}
    RawBody     []byte
}
```

## 2. Logic & Algorithms

### 2.1 RequestValidator Middleware Flow

```
1. Receive HTTP request (c *fiber.Ctx)
2. Extract request path and HTTP method
3. Check if path is in SkipPaths → if yes, call Next() and return
4. Look up ValidationSchema in SchemaRegistry using route pattern
5. If no schema found → call Next() and return
6. Run validation pipeline:
   a. Validate Path Parameters
   b. Validate Query Parameters
   c. Validate Headers
   d. Validate Body (if POST/PUT/PATCH)
7. If any validation fails:
   a. Collect all errors
   b. Build ValidationResult with errors
   c. Return 400 Bad Request with error details
8. If all validations pass:
   a. Store validated data in RequestContext
   c.Locals("validated", ctx)
   b. Call Next()
```

### 2.2 Path Parameter Validation Algorithm

```
FOR each ValidationRule in schema.PathParams:
    1. Extract value from c.Params(rule.FieldName)
    2. IF value is empty AND rule.Required:
        ADD ValidationError(field=rule.FieldName, code="REQUIRED")
        CONTINUE to next rule
    3. IF value is empty AND NOT rule.Required:
        CONTINUE to next rule
    4. Switch on rule.FieldType:
        CASE string:
            IF rule.MinLength AND len(value) < *rule.MinLength:
                ADD ValidationError(field=rule.FieldName, code="MIN_LENGTH")
            IF rule.MaxLength AND len(value) > *rule.MaxLength:
                ADD ValidationError(field=rule.FieldName, code="MAX_LENGTH")
            IF rule.Pattern AND NOT rule.Pattern.MatchString(value):
                ADD ValidationError(field=rule.FieldName, code="PATTERN_MISMATCH")
        CASE int:
            parsed, err := strconv.Atoi(value)
            IF err:
                ADD ValidationError(field=rule.FieldName, code="INVALID_TYPE")
            ELSE:
                IF rule.MinValue AND float64(parsed) < *rule.MinValue:
                    ADD ValidationError(field=rule.FieldName, code="MIN_VALUE")
                IF rule.MaxValue AND float64(parsed) > *rule.MaxValue:
                    ADD ValidationError(field=rule.FieldName, code="MAX_VALUE")
        CASE uuid:
            IF NOT isValidUUID(value):
                ADD ValidationError(field=rule.FieldName, code="INVALID_UUID")
    5. Store validated value in context.PathParams
```

### 2.3 Query Parameter Validation Algorithm

```
FOR each ValidationRule in schema.QueryParams:
    1. Extract value from c.Query(rule.FieldName)
    2. IF value is empty AND rule.Required:
        ADD ValidationError(field=rule.FieldName, code="REQUIRED")
        CONTINUE to next rule
    3. IF value is empty AND NOT rule.Required:
        CONTINUE to next rule
    4. Apply type-specific validation (same as path parameters)
    5. Handle comma-separated values for array types:
        IF rule.FieldType indicates array:
            values := strings.Split(value, ",")
            Validate each element recursively
    6. Store validated value in context.QueryParams
```

### 2.4 Header Validation Algorithm

```
FOR each ValidationRule in schema.Headers:
    1. Extract value from c.Get(rule.FieldName)
    2. Normalize header name (case-insensitive lookup):
        normalizedName := strings.ToLower(rule.FieldName)
    3. IF value is empty AND rule.Required:
        ADD ValidationError(field=rule.FieldName, code="REQUIRED")
        CONTINUE to next rule
    4. IF value is empty AND NOT rule.Required:
        CONTINUE to next rule
    5. Apply type-specific validation
    6. Special handling for Content-Type:
        Validate against allowed content types
    7. Special handling for Authorization:
        Validate Bearer token format if present
    8. Store validated value in context.Headers
```

### 2.5 Body Validation Algorithm

```
1. IF HTTP method is GET/HEAD AND schema has BodyParams:
    ADD ValidationError(field="body", code="UNEXPECTED_BODY")
    RETURN false
2. IF HTTP method is not in [POST, PUT, PATCH] AND BodyParams exist:
    CONTINUE (body validation skipped)
3. Parse request body:
    rawBody := c.Body()
    IF rawBody is nil AND schema.BodyParams has required fields:
        ADD ValidationError(field="body", code="MISSING_BODY")
        RETURN false
4. Determine content type:
    contentType := c.Get("Content-Type")
    SWITCH:
        CASE strings.Contains(contentType, "application/json"):
            bodyData := make(map[string]interface{})
            err := json.Unmarshal(rawBody, &bodyData)
            IF err:
                ADD ValidationError(field="body", code="INVALID_JSON")
                RETURN false
        CASE strings.Contains(contentType, "application/x-www-form-urlencoded"):
            Parse body using c.BodyParser()
        CASE strings.Contains(contentType, "multipart/form-data"):
            Parse body using c.Context().FormValue()
        DEFAULT:
            ADD ValidationError(field="body", code="UNSUPPORTED_CONTENT_TYPE")
            RETURN false
5. FOR each ValidationRule in schema.BodyParams:
    a. Extract value from bodyData using dot notation:
        value := getNestedValue(bodyData, rule.FieldName)
    b. IF value is nil AND rule.Required:
        ADD ValidationError(field=rule.FieldName, code="REQUIRED")
        CONTINUE
    c. IF value is nil AND NOT rule.Required:
        CONTINUE
    d. Apply type validation and constraint checks
    e. Store validated value in context.BodyParams
```

### 2.6 Date Format Validation

```
VALID_DATE_FORMATS = [
    "2006-01-02",           // ISO date
    "2006-01-02T15:04:05Z", // ISO datetime
    "01/02/2006",           // US format
    "02/01/2006",           // EU format
]

FOR each date validation:
    parsedDate := tryParseWithFormats(value, VALID_DATE_FORMATS)
    IF parsedDate is nil:
        ADD ValidationError(field, code="INVALID_DATE")
```

### 2.7 Email Validation

```
EMAIL_PATTERN := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

FOR each email validation:
    IF NOT EMAIL_PATTERN.MatchString(value):
        ADD ValidationError(field, code="INVALID_EMAIL")
    IF domain in blockedDomains:
        ADD ValidationError(field, code="BLOCKED_DOMAIN")
```

## 3. State Management & Error Handling

### 3.1 Error States

| Error Code | HTTP Status | Description | Recovery Action |
| :--- | :--- | :--- | :--- |
| REQUIRED | 400 | Required field missing | Client provides the missing field |
| INVALID_TYPE | 400 | Field type mismatch | Client corrects the data type |
| MIN_LENGTH | 400 | String shorter than minimum | Client provides longer string |
| MAX_LENGTH | 400 | String longer than maximum | Client provides shorter string |
| MIN_VALUE | 400 | Numeric value too small | Client provides larger number |
| MAX_VALUE | 400 | Numeric value too large | Client provides smaller number |
| PATTERN_MISMATCH | 400 | String doesn't match pattern | Client corrects format |
| INVALID_UUID | 400 | Invalid UUID format | Client provides valid UUID |
| INVALID_EMAIL | 400 | Invalid email format | Client corrects email |
| INVALID_DATE | 400 | Invalid date format | Client uses supported format |
| INVALID_JSON | 400 | Malformed JSON body | Client fixes JSON syntax |
| MISSING_BODY | 400 | Expected body but none provided | Client includes body |
| UNSUPPORTED_CONTENT_TYPE | 415 | Unsupported Content-Type | Client uses supported type |
| UNEXPECTED_BODY | 400 | Body on GET/HEAD request | Client removes body |
| BLOCKED_DOMAIN | 400 | Email domain blocked | Client uses different email |
| ENUM_VALUE | 400 | Value not in allowed enum | Client uses valid enum value |

### 3.2 State Transitions

```
Initial State: NEW_REQUEST
    ↓
[Extract Route Pattern]
    ↓
Lookup Schema (SchemaRegistry)
    ↓
    ├─→ Schema Found → VALIDATION_PIPELINE
    └─→ Schema Not Found → SKIP_VALIDATION → CALL_NEXT

VALIDATION_PIPELINE:
    ├─→ Path Params Invalid → ERROR_COLLECTION
    ├─→ Query Params Invalid → ERROR_COLLECTION
    ├─→ Headers Invalid → ERROR_COLLECTION
    ├─→ Body Invalid → ERROR_COLLECTION
    └─→ All Valid → STORE_CONTEXT → CALL_NEXT

ERROR_COLLECTION:
    ├─→ Collect all errors (fail-fast: false)
    ├─→ Build ValidationResult
    └─→ ERROR_RESPONSE

ERROR_RESPONSE:
    ├─→ Format error response JSON
    ├─→ Set status 400/415
    ├─→ Call ErrorHandler (custom or default)
    └─→ End request (do not call Next)
```

### 3.3 Error Response Format

```json
{
    "error": {
        "code": "VALIDATION_FAILED",
        "message": "Request validation failed",
        "details": [
            {
                "field": "email",
                "value": "invalid-email",
                "message": "must be a valid email address",
                "code": "INVALID_EMAIL"
            },
            {
                "field": "age",
                "value": "150",
                "message": "must be less than or equal to 120",
                "code": "MAX_VALUE"
            }
        ]
    },
    "request_id": "req_abc123",
    "timestamp": "2026-01-23T10:30:00Z"
}
```

### 3.4 Panic Recovery

```
RECOVERY_FUNCTION():
    IF recover() is not nil:
        Log panic with stack trace
        Return 500 Internal Server Error
        "An unexpected error occurred during validation"
```

## 4. Component Interfaces

### 4.1 Public Middleware Function

```go
// NewRequestValidator creates a new RequestValidator middleware with the given configuration
func NewRequestValidator(config RequestValidatorConfig) fiber.Handler {
    return func(c *fiber.Ctx) error {
        // Implementation
    }
}
```

### 4.2 Schema Builder Helper

```go
// NewValidationSchema creates a ValidationSchema with fluent API
func NewValidationSchema() *ValidationSchema {
    return &ValidationSchema{
        PathParams:  make([]ValidationRule, 0),
        QueryParams: make([]ValidationRule, 0),
        BodyParams:  make([]ValidationRule, 0),
        Headers:     make([]ValidationRule, 0),
    }
}

// WithPathParam adds a path parameter validation rule
func (s *ValidationSchema) WithPathParam(name string, fieldType FieldType, required bool) *ValidationSchema

// WithQueryParam adds a query parameter validation rule
func (s *ValidationSchema) WithQueryParam(name string, fieldType FieldType, required bool) *ValidationSchema

// WithBodyParam adds a body parameter validation rule
func (s *ValidationSchema) WithBodyParam(name string, fieldType FieldType, required bool) *ValidationSchema

// WithHeader adds a header validation rule
func (s *ValidationSchema) WithHeader(name string, fieldType FieldType, required bool) *ValidationSchema

// MinLength sets the minimum length constraint (for string types)
func (r *ValidationRule) MinLength(length int) *ValidationRule

// MaxLength sets the maximum length constraint (for string types)
func (r *ValidationRule) MaxLength(length int) *ValidationRule

// MinValue sets the minimum value constraint (for numeric types)
func (r *ValidationRule) MinValue(value float64) *ValidationRule

// MaxValue sets the maximum value constraint (for numeric types)
func (r *ValidationRule) MaxValue(value float64) *ValidationRule

// Pattern sets a regex pattern constraint
func (r *ValidationRule) Pattern(pattern string) *ValidationRule

// EnumValues sets allowed values for enum types
func (r *ValidationRule) EnumValues(values ...string) *ValidationRule
```

### 4.3 Validation Functions

```go
// ValidatePathParams validates path parameters against the schema
func ValidatePathParams(c *fiber.Ctx, rules []ValidationRule) ([]ValidationError, map[string]interface{})

// ValidateQueryParams validates query parameters against the schema
func ValidateQueryParams(c *fiber.Ctx, rules []ValidationRule) ([]ValidationError, map[string]interface{})

// ValidateHeaders validates request headers against the schema
func ValidateHeaders(c *fiber.Ctx, rules []ValidationRule) ([]ValidationError, map[string]interface{})

// ValidateBody validates request body against the schema
func ValidateBody(c *fiber.Ctx, rules []ValidationRule) ([]ValidationError, map[string]interface{})
```

### 4.4 Type Validation Functions

```go
// ValidateString validates a string value against rules
func ValidateString(value string, rule ValidationRule) []ValidationError

// ValidateInt validates an integer value against rules
func ValidateInt(value string, rule ValidationRule) []ValidationError

// ValidateFloat validates a float value against rules
func ValidateFloat(value string, rule ValidationRule) []ValidationError

// ValidateBool validates a boolean value against rules
func ValidateBool(value string, rule ValidationRule) []ValidationError

// ValidateEmail validates an email value
func ValidateEmail(value string, rule ValidationRule) []ValidationError

// ValidateUUID validates a UUID value
func ValidateUUID(value string, rule ValidationRule) []ValidationError

// ValidateDate validates a date value against allowed formats
func ValidateDate(value string, rule ValidationRule) []ValidationError

// ValidateEnum validates a value against allowed enum values
func ValidateEnum(value string, rule ValidationRule) []ValidationError
```

### 4.5 Utility Functions

```go
// GetNestedValue extracts a value from a nested map using dot notation
func GetNestedValue(data map[string]interface{}, path string) interface{}

// SchemaRegistry returns a pre-configured schema registry for common patterns
func DefaultSchemaRegistry() map[string]ValidationSchema

// IsValidUUID validates a UUID string
func IsValidUUID(uuid string) bool

// IsSkipPath checks if a path should skip validation
func IsSkipPath(path string, skipPaths []string) bool
```

### 4.6 Configuration Defaults

```go
var DefaultConfig = RequestValidatorConfig{
    SkipPaths: []string{
        "/health",
        "/ready",
        "/metrics",
        "/favicon.ico",
    },
    ErrorHandler: func(c *fiber.Ctx, err error) error {
        code := fiber.StatusBadRequest
        if e, ok := err.(*fiber.Error); ok {
            code = e.Code
        }
        return c.Status(code).JSON(fiber.Map{
            "error": fiber.Map{
                "code":    "VALIDATION_FAILED",
                "message": err.Error(),
            },
        })
    },
    EnableDebug: false,
}
```
