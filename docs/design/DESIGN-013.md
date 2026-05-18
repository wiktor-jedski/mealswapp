## FILE: DESIGN-013.md
**Traceability:** ARCH-013

**Static aspects covered:** EncryptionService, InputSanitizer, AuditLogger, TLSEnforcer, RateLimiter, CSRFValidator.

### 0. Static Aspect Responsibilities
- `EncryptionService`: owns AES-256-GCM envelope encryption and decryption for PII fields.
- `InputSanitizer`: owns field-level validation, escaping, rejection, and sanitization metadata.
- `AuditLogger`: owns structured security and admin audit event persistence.
- `TLSEnforcer`: owns TLS 1.3 policy, HTTP redirects, and HSTS configuration.
- `RateLimiter`: owns security-focused rate limits using Fiber built-in limiter.
- `CSRFValidator`: owns Fiber csrf middleware integration and token validation.

### 1. Data Structures & Types
- `interface EncryptionEnvelope { keyVersion: string; nonce: []byte; ciphertext: []byte; algorithm: "AES-256-GCM" }`
- `interface SanitizationResult { value: string; changed: boolean; violations: string[] }`
- `interface AuditLogEntry { requestId: string; userId?: UUID; action: string; resource: string; outcome: "success" | "failure"; ip: string; userAgent: string; createdAt: time.Time }`
- `interface RateLimitDecision { allowed: boolean; retryAfterSeconds?: number; key: string }`
- `interface CSRFTokenPair { cookieToken: string; formToken: string; expiresAt: time.Time }`
- `interface TLSPolicy { minVersion: "1.3"; redirectHTTP: boolean; hstsMaxAgeSeconds: number }`

### 2. Logic & Algorithms (Step-by-Step)
1. Load encryption keys from GCP Secret Manager at process start; identify active key by version.
2. Encrypt PII fields with AES-256-GCM before repository persistence.
3. Decrypt only at service boundaries that need plaintext; never log plaintext PII.
4. Sanitize string inputs by validating expected shape first, then escaping or rejecting dangerous content based on field type.
5. Use parameterized SQL in ARCH-005 as the primary SQL injection defense.
6. Enforce TLS 1.3 and redirect HTTP to HTTPS in deployed environments.
7. Apply Fiber limiter middleware using IP, user, or endpoint scoped keys.
8. Validate CSRF synchronizer tokens for state-changing requests.
9. Write structured audit logs for auth events, API requests, errors, and admin actions.

### 3. State Management & Error Handling
- `encrypted`: field stored as envelope.
- `decryption_failed`: return internal error and alert because data or key state is inconsistent.
- `input_rejected`: validation failed before sanitization; return 400.
- `sanitized`: accepted value differs from input; log only metadata.
- `rate_limited`: return 429.
- `csrf_invalid`: return 403.
- `tls_required`: redirect or reject non-TLS traffic.
- `audit_unavailable`: continue low-risk reads but fail security-sensitive mutations if audit cannot be recorded.

### 4. Component Interfaces
- `func EncryptPII(ctx context.Context, plaintext []byte) (EncryptionEnvelope, error)`
- `func DecryptPII(ctx context.Context, envelope EncryptionEnvelope) ([]byte, error)`
- `func SanitizeInput(field string, value string) (SanitizationResult, error)`
- `func Audit(ctx context.Context, entry AuditLogEntry) error`
- `func EnforceTLS(policy TLSPolicy) fiber.Handler`
- `func SecurityRateLimiter(scope string, max int, window time.Duration) fiber.Handler`
- `func GenerateCSRFToken(ctx *fiber.Ctx) (CSRFTokenPair, error)`
- `func ValidateCSRFToken(ctx *fiber.Ctx) error`
