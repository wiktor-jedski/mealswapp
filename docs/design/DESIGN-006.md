## FILE: DESIGN-006.md
**Traceability:** ARCH-006

**Static aspects covered:** AuthController, PasswordHasher, JWTManager, OAuthHandler, SessionManager, AccountLockoutTracker.

### 0. Static Aspect Responsibilities
- `AuthController`: owns registration, login, logout, refresh, verification, and password reset HTTP handlers.
- `PasswordHasher`: owns Argon2 salt generation, hashing, and verification.
- `JWTManager`: owns access token creation, validation, expiry, and refresh-token rotation metadata.
- `OAuthHandler`: owns goth provider flow, profile normalization, and account linking.
- `SessionManager`: owns Fiber session creation, cookie settings, and session invalidation.
- `AccountLockoutTracker`: owns failed-attempt counters, lockout windows, and retry timing.

### 1. Data Structures & Types
- `interface Credentials { email: string; password: string }`
- `interface AuthUser { id: UUID; email: string; emailVerified: boolean; role: "user" | "admin"; passwordHash?: string; oauthProvider?: string }`
- `interface SessionTokens { accessToken: string; refreshToken: string; accessExpiresAt: time.Time; refreshExpiresAt: time.Time }`
- `interface LockoutState { accountFailures: number; ipFailures: number; lockedUntil?: time.Time }`
- `interface OAuthProfile { provider: "google" | "apple"; providerUserId: string; email: string; displayName?: string }`
- `interface PasswordResetToken { tokenHash: string; userId: UUID; expiresAt: time.Time; usedAt?: time.Time }`

### 2. Logic & Algorithms (Step-by-Step)
1. Registration validates email format, password policy, and uniqueness through ARCH-005.
2. Generate a unique salt and hash the password with Argon2 from `golang.org/x/crypto/argon2`.
3. Persist the user as unverified and send a verification email through the configured email provider.
4. Login checks IP and account lockout state before password verification.
5. On successful login, reset failure counters, create a Fiber session, issue 15-minute access and 7-day refresh tokens in HttpOnly/Secure/SameSite=Strict cookies.
6. On failed login, increment account and IP counters; lock account for 15 minutes after 5 account failures and enforce IP limits after 10 failures per 10 minutes.
7. Refresh token flow validates the current refresh token, rotates it, and invalidates the previous token.
8. OAuth flow uses `github.com/markbates/goth`, creates or links accounts, and activates a first-login trial through ARCH-007.
9. Password reset stores only a hash of a random token, enforces 1-hour expiry, and marks the token used after password change.

### 3. State Management & Error Handling
- `unverified`: user can authenticate but paid features remain blocked.
- `authenticated`: active session and valid token cookies exist.
- `refresh_required`: access token expired but refresh token can rotate.
- `locked`: account or IP lockout is active; return retry time.
- `oauth_link_required`: OAuth email matches an existing account and needs explicit linking.
- `invalid_credentials`: return generic message without revealing which field failed.
- `session_expired`: clear cookies and require login.
- `token_reuse_detected`: revoke session family and require reauthentication.

### 4. Component Interfaces
- `func (c *AuthController) Register(ctx *fiber.Ctx) error`
- `func (c *AuthController) Login(ctx *fiber.Ctx) error`
- `func (c *AuthController) Logout(ctx *fiber.Ctx) error`
- `func (c *AuthController) Refresh(ctx *fiber.Ctx) error`
- `func HashPassword(password string) (hash string, salt string, error error)`
- `func VerifyPassword(password string, hash string, salt string) bool`
- `func StartOAuth(ctx *fiber.Ctx, provider string) error`
- `func CompleteOAuth(ctx *fiber.Ctx, provider string) error`
- `func CreatePasswordReset(userID UUID) (plainToken string, error error)`
- `func ConsumePasswordReset(plainToken string, newPassword string) error`
