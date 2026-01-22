# [ARCH-006] - Authentication Module

**Description:** Security service handling user authentication via email/password and social providers (Google, Apple), session management, and token lifecycle.

| Attribute | Value |
| :--- | :--- |
| **Type** | Service |
| **Static Aspects** | AuthController, PasswordHasher, JWTManager, OAuthHandler, SessionManager, AccountLockoutTracker |
| **Dependencies** | ARCH-005 (Data Repository), ARCH-013 (Security Middleware), External OAuth Providers |
| **Traceability** | SW-REQ-046, SW-REQ-058, SW-REQ-059, SW-REQ-060, SW-REQ-061, SW-REQ-062, SW-REQ-063, SW-REQ-064, SW-REQ-065, SW-REQ-066, SW-REQ-069, SW-REQ-070 |

**Dynamic Behavior:**

- **Registration:** Validates email uniqueness, hashes password with Argon2 (unique salt), sends verification email. Blocks paid features until verified.
- **Login:** Validates credentials, tracks failed attempts per account (5 max -> 15min lockout) and per IP (10 max/10min).
- **Token Lifecycle:** Issues 15-minute access tokens and 7-day refresh tokens in HttpOnly/Secure/SameSite=Strict cookies. Rotates refresh token on use.
- **Social Login:** Handles OAuth2 flows for Google/Apple, creates or links user accounts, grants 7-day trial on first authentication.
- **Password Reset:** Generates cryptographically random single-use tokens valid for 1 hour.

**Interface Definition:**

- `Input`: Credentials (email/password or OAuth tokens), session cookies
- `Output`: JWT tokens (access/refresh), session state, verification emails

**Alternative Analysis (BP6):**

- *Chosen Approach:* Custom JWT-based authentication with HttpOnly cookies
- *Alternative Considered:* Third-party auth service (Auth0, Firebase Auth)
- *Trade-off:* Custom implementation provides full control over security requirements (SW-REQ-062, SW-REQ-063, SW-REQ-065) and avoids vendor lock-in. Third-party services simplify development but may not support exact lockout policies or cookie configurations required. For a subscription-based app with specific security needs, custom implementation ensures compliance.

**Reference Documentation:** 
- 02_APPENDIX_A.md
