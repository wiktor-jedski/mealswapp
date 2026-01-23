# InputSanitizer

**Traceability:** ARCH-013

---

## 1. Data Structures & Types

### 1.1 SanitizationConfig

```go
type SanitizationConfig struct {
    AllowHTML          bool
    AllowedTags        []string
    AllowedAttributes  map[string][]string
    MaxInputLength     int
    StripNullBytes     bool
    EscapeSQL          bool
    EscapeShell        bool
}
```

### 1.2 SanitizationResult

```go
type SanitizationResult struct {
    Value      interface{}
    IsValid    bool
    Sanitized  bool
    Warnings   []string
    Errors     []SanitizationError
}
```

### 1.3 SanitizationError

```go
type SanitizationError struct {
    Field     string
    Code      string
    Message   string
    Value     interface{}
}
```

### 1.4 Sanitizer

```go
type Sanitizer struct {
    config    SanitizationConfig
    logger    *log.Logger
    meter     metric.Meter
}
```

### 1.5 InputType

```go
type InputType string

const (
    InputTypeString  InputType = "string"
    InputTypeNumber  InputType = "number"
    InputTypeBool    InputType = "bool"
    InputTypeArray   InputType = "array"
    InputTypeObject  InputType = "object"
    InputTypeEmail   InputType = "email"
    InputTypeURL     InputType = "url"
    InputTypeHTML    InputType = "html"
)
```

### 1.6 ValidationRule

```go
type ValidationRule struct {
    Field       string
    InputType   InputType
    Required    bool
    MinLength   *int
    MaxLength   *int
    MinValue    *float64
    MaxValue    *float64
    Pattern     *regexp.Regexp
    Custom      func(interface{}) bool
}
```

---

## 2. Logic & Algorithms (Step-by-Step)

### 2.1 Sanitize Input Flow

```
1. Receive raw input (interface{})
2. Validate input type matches expected type
3. Apply length constraints
4. Strip null bytes if configured
5. Escape SQL characters if configured
6. Escape shell characters if configured
7. Process based on input type:
   - STRING: HTML escape, apply tag/attribute rules
   - NUMBER: Parse and validate range
   - BOOL: Validate boolean value
   - ARRAY: Recursively sanitize each element
   - OBJECT: Recursively sanitize each field
   - EMAIL: Validate email format
   - URL: Validate URL format, ensure safe protocol
8. Return SanitizationResult with sanitized value
```

### 2.2 XSS Prevention Algorithm

```
1. Check if HTML is allowed
2. If not allowed:
   - HTML escape all special characters:
     & → &amp;
     < → &lt;
     > → &gt;
     " → &quot;
     ' → &#x27;
     / → &#x2F;
   - Return escaped string
3. If allowed:
   - Parse HTML using goquery or bluemonday
   - Remove all tags except allowed list
   - Remove all attributes except allowed list
   - Remove event handlers (onclick, onerror, etc.)
   - Remove javascript: and data: protocols
   - Return sanitized HTML
```

### 2.3 SQL Injection Prevention Algorithm

```
1. Identify string values in input
2. Apply escaping to dangerous characters:
   - ' → ''
   - " → \"
   - \ → \\
   - ; → \; (comment delimiter)
   - -- → \- \- (comment delimiter)
   - /* → \/* (comment delimiter)
   - UNION → sanitized
   - SELECT → sanitized
   - DROP → sanitized
   - INSERT → sanitized
   - UPDATE → sanitized
   - DELETE → sanitized
3. If pattern matching detects SQL keywords, reject input
4. Use parameterized query patterns as alternative
```

### 2.4 Shell Escape Algorithm

```
1. Identify string values
2. Apply shell escape:
   - ; → \;
   - | → \|
   - & → \&
   - $ → \$
   - ` → \`
   - " → \"
   - ' → \'
   - \ → \\
   - ( → \(
   - ) → \)
   - { → \{
   - } → \}
   - [ → \[
   - ] → \]
   - * → \*
   - ? → \?
   - < → \<
   - > → \>
   - # → \#
   - ~ → \~
   - ! → \!
   - % → \%
   - ^ → \^
   - Newline → \n
3. Reject input containing dangerous commands:
   - rm, del, erase, unlink, mkfs, format
   - wget, curl, nc, netcat
   - sh, bash, powershell, cmd
   - sudo, su, passwd
```

### 2.5 Fiber Middleware Flow

```
1. Extract request body/params/headers
2. For each input field:
   - Detect input type from schema/validation rules
   - Apply appropriate sanitization
   - Track sanitization events (sanitized, rejected)
3. If any validation fails:
   - Return 400 Bad Request with sanitization errors
4. If all inputs valid:
   - Continue to next handler
5. Log sanitization metrics:
   - Inputs sanitized count
   - Inputs rejected count
   - XSS attempts blocked
   - SQL injection attempts blocked
   - Shell injection attempts blocked
```

### 2.6 Recursive Object Sanitization

```
1. Iterate through object fields
2. For each field value:
   - If primitive type: sanitize directly
   - If array: recursively sanitize each element
   - If object: recursively sanitize each field
3. Return sanitized object
```

---

## 3. State Management & Error Handling

### 3.1 Error States

| Error State | Condition | Transition |
| :--- | :--- | :--- |
| `InputTooLong` | Input exceeds MaxLength | Reject with 400, log warning |
| `InvalidType` | Input type doesn't match expected | Reject with 400, log error |
| `XSSDetected` | XSS patterns detected in input | Reject with 400, log security event |
| `SQLInjectionDetected` | SQL injection patterns detected | Reject with 400, log security event |
| `ShellInjectionDetected` | Shell injection patterns detected | Reject with 400, log security event |
| `RequiredFieldMissing` | Required field is empty/null | Reject with 400, log validation error |
| `InvalidEmailFormat` | Email validation fails | Reject with 400, log validation error |
| `InvalidURLFormat` | URL validation fails | Reject with 400, log validation error |
| `ValueOutOfRange` | Number outside min/max bounds | Reject with 400, log validation error |
| `PatternMismatch` | Input doesn't match regex pattern | Reject with 400, log validation error |
| `NullByteInjection` | Null bytes detected in input | Strip null bytes or reject |

### 3.2 State Transitions

```
Initial State: Ready

Ready → Sanitizing: On input received
Sanitizing → Validated: On successful sanitization
Sanitizing → Error: On sanitization failure

Validated → Ready: On completion
Error → Ready: After error response sent
```

### 3.3 Panic Recovery

```
1. Recover from panics in sanitize functions
2. Log panic with stack trace
3. Return sanitized value with error
4. Emit metrics for panic count
5. Never expose internal errors to caller
```

### 3.4 Audit Logging Events

```
- SANITIZATION_STARTED: Sanitization process began
- SANITIZATION_COMPLETED: Sanitization completed successfully
- SANITIZATION_FAILED: Sanitization failed with errors
- XSS_ATTEMPT_BLOCKED: XSS attack detected and blocked
- SQL_INJECTION_BLOCKED: SQL injection detected and blocked
- SHELL_INJECTION_BLOCKED: Shell injection detected and blocked
- INPUT_REJECTED: Input failed validation rules
- FIELD_SANITIZED: Individual field was sanitized
```

### 3.5 Metrics

```
- input_sanitizer_requests_total: Total sanitization requests
- input_sanitizer_sanitized_total: Total inputs sanitized
- input_sanitizer_rejected_total: Total inputs rejected
- input_sanitizer_xss_blocked_total: XSS attempts blocked
- input_sanitizer_sql_blocked_total: SQL injection attempts blocked
- input_sanitizer_shell_blocked_total: Shell injection attempts blocked
- input_sanitizer_latency_seconds: Sanitization latency histogram
```

---

## 4. Component Interfaces

### 4.1 NewSanitizer

```go
func NewSanitizer(config SanitizationConfig, logger *log.Logger) *Sanitizer
```

**Parameters:**
- `config`: SanitizationConfig with sanitization rules
- `logger`: Logger instance for audit logging

**Returns:**
- Sanitizer instance

**Behavior:**
- Initializes sanitizer with provided configuration
- Sets up default config if none provided
- Initializes metrics meter

### 4.2 Sanitize

```go
func (s *Sanitizer) Sanitize(input interface{}, inputType InputType, fieldName string) SanitizationResult
```

**Parameters:**
- `input`: Raw input value to sanitize
- `inputType`: Expected type of input
- `fieldName`: Name of field (for error reporting)

**Returns:**
- SanitizationResult with sanitized value or errors

**Behavior:**
- Validates input type matches expected type
- Applies length constraints
- Strips null bytes if configured
- Escapes SQL if configured
- Escapes shell if configured
- Applies type-specific sanitization
- Returns result with sanitized value

### 4.3 SanitizeString

```go
func (s *Sanitizer) SanitizeString(value string, allowHTML bool) (string, []SanitizationError)
```

**Parameters:**
- `value`: String value to sanitize
- `allowHTML`: Whether to allow HTML tags

**Returns:**
- Sanitized string
- List of sanitization errors

**Behavior:**
- Applies HTML escaping or sanitization
- Applies SQL escaping if configured
- Applies shell escaping if configured
- Strips null bytes

### 4.4 SanitizeNumber

```go
func (s *Sanitizer) SanitizeNumber(value interface{}, min, max *float64) (float64, []SanitizationError)
```

**Parameters:**
- `value`: Numeric value (float64, int, or string representation)
- `min`: Minimum allowed value (optional)
- `max`: Maximum allowed value (optional)

**Returns:**
- Sanitized numeric value
- List of sanitization errors

**Behavior:**
- Parses string to float64 if needed
- Validates numeric range
- Returns errors if parsing fails or out of range

### 4.5 SanitizeArray

```go
func (s *Sanitizer) SanitizeArray(arr []interface{}, itemType InputType) ([]interface{}, []SanitizationError)
```

**Parameters:**
- `arr`: Array to sanitize
- `itemType`: Expected type of array items

**Returns:**
- Sanitized array
- List of sanitization errors

**Behavior:**
- Recursively sanitizes each array element
- Collects all errors from items

### 4.6 SanitizeObject

```go
func (s *Sanitizer) SanitizeObject(obj map[string]interface{}, rules []ValidationRule) (map[string]interface{}, []SanitizationError)
```

**Parameters:**
- `obj`: Object to sanitize
- `rules`: Validation rules for each field

**Returns:**
- Sanitized object
- List of sanitization errors

**Behavior:**
- Validates required fields
- Applies type-specific sanitization per field
- Collects all errors

### 4.7 ValidateEmail

```go
func (s *Sanitizer) ValidateEmail(email string) (bool, []SanitizationError)
```

**Parameters:**
- `email`: Email address to validate

**Returns:**
- True if valid email
- List of validation errors

**Behavior:**
- Validates email format using regex
- Strips potentially dangerous characters

### 4.8 ValidateURL

```go
func (s *Sanitizer) ValidateURL(url string) (bool, []SanitizationError)
```

**Parameters:**
- `url`: URL to validate

**Returns:**
- True if valid URL
- List of validation errors

**Behavior:**
- Validates URL format
- Ensures safe protocols (http, https only)
- Rejects javascript:, data:, etc.

### 4.9 CreateFiberMiddleware

```go
func (s *Sanitizer) CreateFiberMiddleware(config FiberMiddlewareConfig) fiber.Handler
```

**Parameters:**
- `config`: Fiber middleware configuration

**Returns:**
- Fiber middleware handler

**Behavior:**
- Extracts inputs from request (body, query, params, headers)
- Applies sanitization based on field definitions
- Returns 400 on sanitization failure
- Continues to next handler on success
- Logs all sanitization events

### 4.10 FiberMiddlewareConfig

```go
type FiberMiddlewareConfig struct {
    BodyFields     map[string]ValidationRule
    QueryFields    map[string]ValidationRule
    ParamsFields   map[string]ValidationRule
    HeadersFields  map[string]ValidationRule
    OnError        func(c *fiber.Ctx, errors []SanitizationError) error
    Skipper        func(c *fiber.Ctx) bool
}
```

### 4.11 BlockXSSPatterns

```go
func (s *Sanitizer) BlockXSSPatterns(value string) (string, bool)
```

**Parameters:**
- `value`: String to check for XSS patterns

**Returns:**
- Potentially modified value
- True if XSS patterns were detected

**Behavior:**
- Checks for common XSS patterns:
  - <script> tags
  - javascript: protocol
  - event handlers (onclick, onerror, etc.)
  - <iframe>, <object>, <embed>
  - SVG malicious content
  - Data exfiltration patterns

### 4.12 BlockSQLInjection

```go
func (s *Sanitizer) BlockSQLInjection(value string) (string, bool)
```

**Parameters:**
- `value`: String to check for SQL injection

**Returns:**
- Potentially modified value
- True if SQL injection patterns detected

**Behavior:**
- Checks for SQL keywords and patterns:
  - UNION, SELECT, INSERT, UPDATE, DELETE
  - DROP, TRUNCATE, ALTER
  - Comment patterns (--, /*, */)
  - OR 1=1, AND 1=1
  - SLEEP(), BENCHMARK()
  -information_schema queries

### 4.13 BlockShellInjection

```go
func (s *Sanitizer) BlockShellInjection(value string) (string, bool)
```

**Parameters:**
- `value`: String to check for shell injection

**Returns:**
- Potentially modified value
- True if shell injection patterns detected

**Behavior:**
- Checks for shell metacharacters and commands
- Detects dangerous command patterns
- Rejects input with shell commands

### 4.14 DefaultConfig

```go
func DefaultConfig() SanitizationConfig
```

**Returns:**
- Default sanitization configuration

**Behavior:**
- AllowHTML: false
- AllowedTags: []string{} (none)
- AllowedAttributes: map[string][]string{} (none)
- MaxInputLength: 10000
- StripNullBytes: true
- EscapeSQL: true
- EscapeShell: true

### 4.15 StrictConfig

```go
func StrictConfig() SanitizationConfig
```

**Returns:**
- Strict sanitization configuration for high-security contexts

**Behavior:**
- AllowHTML: false
- MaxInputLength: 1000
- StripNullBytes: true
- EscapeSQL: true
- EscapeShell: true
- Additional pattern blocking enabled

### 4.16 HTMLPermissiveConfig

```go
func HTMLPermissiveConfig(allowedTags, allowedAttributes map[string][]string) SanitizationConfig
```

**Parameters:**
- `allowedTags`: List of allowed HTML tags
- `allowedAttributes`: Map of allowed attributes per tag

**Returns:**
- Configuration allowing specific HTML

**Behavior:**
- AllowHTML: true
- Uses whitelist approach for tags/attributes
- All other sanitization enabled
