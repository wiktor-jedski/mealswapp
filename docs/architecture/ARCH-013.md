# [ARCH-013] - Security Middleware

**Description:** Cross-cutting security services implementing encryption, input validation, and audit logging across all components.

| Attribute | Value |
| :--- | :--- |
| **Type** | Middleware |
| **Static Aspects** | EncryptionService (AES-256 via crypto/aes), InputSanitizer, AuditLogger, TLSEnforcer, RateLimiter (Fiber built-in limiter), CSRFValidator (Fiber csrf middleware) |
| **Dependencies** | All services |
| **Traceability** | SW-REQ-059, SW-REQ-068, SW-REQ-075, SW-REQ-084 |

**Dynamic Behavior:**

- **Encryption at Rest:** AES-256 encryption (crypto/aes) for PII fields in database.
- **Encryption in Transit:** TLS 1.3 enforced for all connections. HTTP redirects to HTTPS.
- **Input Validation:** Sanitizes all user inputs to prevent XSS, SQL injection, and command injection.
- **Rate Limiting:** Enforces rate limits using Fiber built-in limiter middleware.
- **CSRF Protection:** Validates synchronizer tokens on all state-changing requests using Fiber csrf middleware.
- **Audit Logging:** Logs all authentication events, API requests, errors, and admin actions with timestamps and user IDs.

**Interface Definition:**

- `Input`: Raw data for encryption, user inputs for validation
- `Output`: Encrypted data, sanitized inputs, audit log entries

**Alternative Analysis (BP6):**

- *Chosen Approach:* Application-level encryption with database-native TDE as backup
- *Alternative Considered:* Full database-level Transparent Data Encryption (TDE) only
- *Trade-off:* Application-level encryption provides field-level control over which data is encrypted and allows encryption keys to be managed separately from database. TDE-only would encrypt entire database but not protect against application-level data leaks. Layered approach provides defense in depth.
