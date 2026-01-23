---
## FILE: AuthController.md
**Traceability:** ARCH-006

### 1. Data Structures & Types

```go
package auth

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/markbates/goth"
)

// User represents the core user entity from ARCH-005
type User struct {
	ID            string    `json:"id" db:"id"`
	Email         string    `json:"email" db:"email"`
	PasswordHash  string    `json:"-" db:"password_hash"`
	Provider      string    `json:"provider,omitempty" db:"provider"` // "email", "google", "apple"
	ProviderID    string    `json:"-" db:"provider_id"`               // External OAuth ID
	IsVerified    bool      `json:"is_verified" db:"is_verified"`
	HasTrial      bool      `json:"has_trial" db:"has_trial"`
	FailedLogins  int       `json:"-" db:"failed_logins"`
	LockedUntil   *time.Time `json:"-" db:"locked_until"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// AuthTokens represents JWT token pair
type AuthTokens struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"-"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
}

// LoginRequest represents email/password login input
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

// RegisterRequest represents user registration input
type RegisterRequest struct {
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=8,password_strength"`
	FirstName string `json:"first_name" validate:"required,min=1,max=50"`
	LastName  string `json:"last_name" validate:"required,min=1,max=50"`
}

// TokenClaims represents JWT payload structure
type TokenClaims struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	TokenType string `json:"token_type"` // "access" or "refresh"
}

// RefreshTokenRecord represents stored refresh token in Redis
type RefreshTokenRecord struct {
	UserID       string    `json:"user_id"`
	TokenHash    string    `json:"token_hash"`
	DeviceID     string    `json:"device_id"`
	IPAddress    string    `json:"ip_address"`
	CreatedAt    time.Time `json:"created_at"`
	ExpiresAt    time.Time `json:"expires_at"`
	RotatedAt    time.Time `json:"rotated_at"`
}

// AccountLockoutState tracks login failure state per account
type AccountLockoutState struct {
	UserID         string    `json:"user_id"`
	FailedAttempts int       `json:"failed_attempts"`
	LockedUntil    *time.Time `json:"locked_until,omitempty"`
}

// IPLockoutState tracks login failure state per IP address
type IPLockoutState struct {
	IPAddress      string    `json:"ip_address"`
	FailedAttempts int       `json:"failed_attempts"`
	LockedUntil    *time.Time `json:"locked_until,omitempty"`
}

// OAuthSession stores Goth OAuth state during flow
type OAuthSession struct {
	State         string    `json:"state"`
	Provider      string    `json:"provider"`
	RedirectURL   string    `json:"redirect_url"`
	ExpiresAt     time.Time `json:"expires_at"`
}

// PasswordResetToken represents single-use reset token
type PasswordResetToken struct {
	UserID     string    `json:"user_id"`
	TokenHash  string    `json:"token_hash"`
	ExpiresAt  time.Time `json:"expires_at"`
	UsedAt     *time.Time `json:"used_at,omitempty"`
}

// AuthError represents authentication failure details
type AuthError struct {
	Code        string `json:"code"`
	Message     string `json:"message"`
	RetryAfter  *int   `json:"retry_after,omitempty"` // Seconds until retry
}

// AuthController handles all authentication operations
type AuthController struct {
	userRepo        UserRepository
	jwtManager      *JWTManager
	passwordHasher  *PasswordHasher
	sessionManager  *SessionManager
	oauthHandler    *OAuthHandler
	lockoutTracker  *AccountLockoutTracker
	redisClient     *redis.Client
	config          *Config
}

// Config holds AuthController configuration
type Config struct {
	AccessTokenTTL   time.Duration // 15 minutes
	RefreshTokenTTL  time.Duration // 7 days
	MaxLoginAttempts int           // 5
	LockoutDuration  time.Duration // 15 minutes
	IPMaxAttempts    int           // 10
	IPLockoutWindow  time.Duration // 10 minutes
	ResetTokenTTL    time.Duration // 1 hour
	CookieDomain     string
	CookieSecure     bool
	CookieSameSite   string
}

// UserRepository interface (ARCH-005 dependency)
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
	GetByProvider(ctx context.Context, provider, providerID string) (*User, error)
	Update(ctx context.Context, user *User) error
	IncrementFailedLogins(ctx context.Context, userID string) error
	ResetFailedLogins(ctx context.Context, userID string) error
	UpdatePasswordHash(ctx context.Context, userID, hash string) error
}
```

### 2. Logic & Algorithms (Step-by-Step)

#### 2.1 Registration Flow

```
REGISTER(email, password, first_name, last_name)
1. Validate RegisterRequest fields
2. Check if email already exists via userRepo.GetByEmail()
3. IF email exists:
   a. IF user.provider != "email": RETURN error "Email registered via OAuth"
   b. ELSE: RETURN error "Email already registered"
4. Generate cryptographically random salt (32 bytes)
5. Call passwordHasher.Hash(password, salt) -> hash
6. Create User struct with:
   - ID: uuidv7()
   - Email: validated email
   - PasswordHash: hash
   - Provider: "email"
   - IsVerified: false
   - HasTrial: true
7. Call userRepo.Create(ctx, user)
8. Generate email verification token via verificationTokenGenerator.Generate(user.ID, email)
9. Send verification email via emailService.SendVerification(user.Email, verificationToken)
10. Create session with unverified state
11. RETURN success response with session
```

#### 2.2 Email/Password Login Flow

```
LOGIN(email, password)
1. Validate LoginRequest fields
2. Extract client IP from Fiber context (c.IPs()[0] or c.IP())
3. Check IP lockout state via lockoutTracker.CheckIPLockout(ipAddress)
4. IF IP is locked:
   a. Calculate retry_after_seconds = locked_until - now
   b. RETURN AuthError{code: "ip_locked", retry_after: seconds}
5. Call userRepo.GetByEmail(email)
6. IF user not found:
   a. Record failed attempt against IP via lockoutTracker.RecordIPFailure(ipAddress)
   b. Add randomized delay (100-500ms) to prevent timing attacks
   c. RETURN AuthError{code: "invalid_credentials"}
7. Check account lockout state via lockoutTracker.CheckAccountLockout(user.ID)
8. IF account is locked:
   a. Calculate retry_after_seconds
   b. RETURN AuthError{code: "account_locked", retry_after: seconds}
9. Call passwordHasher.Verify(user.PasswordHash, password)
10. IF password invalid:
    a. Call userRepo.IncrementFailedLogins(user.ID)
    b. Call lockoutTracker.RecordAccountFailure(user.ID)
    c. IF user.FailedLogins + 1 >= config.MaxLoginAttempts:
       - Set locked_until = now + config.LockoutDuration
    d. Record IP failure
    e. RETURN AuthError{code: "invalid_credentials"}
11. IF password valid:
    a. Call userRepo.ResetFailedLogins(user.ID)
    b. Call lockoutTracker.ClearAccountFailures(user.ID)
    c. IF user.IsVerified == false:
       - Update last_login timestamp
       - RETURN response with unverified session
    d. Generate auth tokens via jwtManager.IssueTokens(user)
    e. Store refresh token via sessionManager.StoreRefreshToken(user.ID, refreshToken)
    f. Set HttpOnly cookies:
       - access_token: expires in 15min
       - refresh_token: expires in 7 days
    g. Update user.LastLoginAt = now
    h. RETURN success with tokens
```

#### 2.3 Token Refresh Flow

```
REFRESH(refresh_token_cookie)
1. Extract refresh_token from cookie
2. IF no refresh token: RETURN AuthError{code: "no_refresh_token"}
3. Parse and validate JWT via jwtManager.ValidateRefresh(refresh_token)
4. IF invalid/expired: RETURN AuthError{code: "invalid_refresh_token"}
5. Look up refresh token record via sessionManager.GetRefreshToken(userID, tokenID)
6. IF not found or used: RETURN AuthError{code: "token_revoked"}
7. Verify token not expired (record.ExpiresAt > now)
8. Generate new access token via jwtManager.IssueAccessToken(user)
9. Generate new refresh token via jwtManager.IssueRefreshToken(user)
10. Rotate tokens:
    a. Delete old refresh token via sessionManager.RevokeRefreshToken(userID, tokenID)
    b. Store new refresh token via sessionManager.StoreRefreshToken(userID, newRefreshToken)
    c. Update token record with new hash and rotated_at timestamp
11. Set new HttpOnly cookies
12. RETURN new access token
```

#### 2.4 Social OAuth Login Flow (Google/Apple)

```
OAUTH_BEGIN(provider, redirect_url)
1. Validate provider is "google" or "apple"
2. Generate state: cryptographically random 32-byte string
3. Create OAuthSession{state, provider, redirect_url, expires_at: now+10min}
4. Store session via sessionManager.StoreOAuthSession(state, session)
5. Call goth.BeginAuth(state, provider, callbackURL)
6. Get auth URL from goth
7. Redirect to provider's OAuth URL with state parameter

OAUTH_CALLBACK(provider, code, state, original_state)
1. Validate state matches stored OAuthSession via sessionManager.GetOAuthSession(state)
2. IF not found or expired: RETURN AuthError{code: "oauth_session_expired"}
3. Delete OAuthSession
4. Call goth.CallBack(provider, code, original_state)
5. Extract goth.User: email, provider_id, name, avatar
6. Attempt userRepo.GetByProvider(provider, provider_id)
7. IF user exists:
    a. Link OAuth provider if user.provider == "email":
       - Update user.Provider = provider
       - Update user.ProviderID = provider_id
       - userRepo.Update(user)
    b. Proceed to login flow
8. IF user doesn't exist:
    a. Check if email exists via userRepo.GetByEmail(goth.User.Email)
    b. IF email exists:
       - Link OAuth to existing account
       - Update user.Provider = provider
       - userRepo.Update(user)
       - Proceed to login
    c. IF email doesn't exist:
       - Create new user:
         * ID: uuidv7()
         * Email: goth.User.Email
         * Provider: provider
         * ProviderID: provider_id
         * IsVerified: true (OAuth emails are verified)
         * HasTrial: true
         * FirstName/LastName: from goth.User.Name
       - userRepo.Create(user)
       - Grant 7-day trial
9. Issue tokens and set cookies
10. Redirect to frontend dashboard
```

#### 2.5 Password Reset Flow

```
REQUEST_PASSWORD_RESET(email)
1. Validate email format
2. Call userRepo.GetByEmail(email)
3. IF user not found: RETURN success (don't reveal email existence)
4. Generate cryptographically random reset token (32 bytes)
5. Hash token via passwordHasher.Hash(token)
6. Create PasswordResetToken record with:
   - UserID: user.ID
   - TokenHash: hashed token
   - ExpiresAt: now + config.ResetTokenTTL
7. Store via sessionManager.StorePasswordResetToken(user.ID, tokenHash, expiresAt)
8. Generate reset link: https://app.mealswapp.com/reset-password?token={raw_token}&user_id={user.ID}
9. Send email via emailService.SendPasswordReset(user.Email, resetLink)
10. RETURN success

RESET_PASSWORD(user_id, raw_token, new_password)
1. Retrieve reset token record via sessionManager.GetPasswordResetToken(user_id)
2. IF not found: RETURN AuthError{code: "reset_token_invalid"}
3. IF record.ExpiresAt < now: RETURN AuthError{code: "reset_token_expired"}
4. IF record.UsedAt != nil: RETURN AuthError{code: "reset_token_used"}
5. Verify token via passwordHasher.Verify(record.TokenHash, raw_token)
6. IF invalid: RETURN AuthError{code: "reset_token_invalid"}
7. Generate new salt and hash via passwordHasher.Hash(new_password, newSalt)
8. Call userRepo.UpdatePasswordHash(userID, newHash)
9. Mark token as used via sessionManager.MarkPasswordResetTokenUsed(userID)
10. Invalidate all existing sessions via sessionManager.RevokeAllUserTokens(userID)
11. Send confirmation email
12. RETURN success
```

#### 2.6 Logout Flow

```
LOGOUT()
1. Extract refresh token from cookie
2. IF refresh token present:
   a. Parse JWT to get userID and tokenID
   b. Revoke refresh token via sessionManager.RevokeRefreshToken(userID, tokenID)
3. Clear cookies:
   - access_token: max_age = -1
   - refresh_token: max_age = -1
4. Destroy Fiber session via sessionManager.Destroy(c)
5. RETURN success
```

### 3. State Management & Error Handling

#### 3.1 Account Lockout States

| State | Condition | Action | Retry After |
|-------|-----------|--------|-------------|
| Account Locked | failed_logins >= 5 | Reject login | 15 minutes |
| IP Locked | IP failures >= 10 | Reject all logins | 10 minutes |
| Verification Required | is_verified == false | Allow session, block paid features | N/A |
| OAuth Linked | provider != "email" | Skip password check | N/A |
| Session Expired | access_token expired | Require refresh token | N/A |
| Refresh Token Revoked | token not in Redis | Force re-login | N/A |

#### 3.2 Error Codes and HTTP Responses

| Code | HTTP Status | Message | Retryable |
|------|-------------|---------|-----------|
| invalid_credentials | 401 | Invalid email or password | Yes (with delay) |
| account_locked | 403 | Account temporarily locked | Yes (after lockout) |
| ip_locked | 429 | Too many login attempts | Yes (after lockout) |
| email_not_verified | 403 | Please verify your email | No |
| no_refresh_token | 401 | Session expired | No |
| invalid_refresh_token | 401 | Invalid session | No |
| token_revoked | 401 | Session invalidated | No |
| reset_token_expired | 400 | Password reset link expired | No |
| reset_token_used | 400 | Password reset link already used | No |
| email_already_registered | 409 | Email already registered | No |
| oauth_session_expired | 400 | OAuth session expired | No |
| provider_not_supported | 400 | OAuth provider not supported | No |

#### 3.3 State Transitions

```
Unverified User
  ├─ Complete email verification → Verified User
  └─ OAuth login → Verified User (OAuth emails verified)

Failed Login
  ├─ Valid password → Logged In (resets counter)
  └─ Invalid password:
     ├─ < 5 failures → Unlocked (can retry)
     └─ 5+ failures → Account Locked (15 min timeout)

Account Locked
  ├─ Timeout expires → Failed Login state
  └─ Successful login → Logged In (resets counter)

Logged In
  ├─ Access token expires → Use refresh token
  ├─ Refresh token expires → Re-authenticate required
  └─ Logout → Not authenticated

OAuth Flow
  ├─ New email → Create account with trial
  ├─ Existing email → Link provider
  └─ Existing provider account → Login
```

#### 3.4 Security Considerations

- All password operations use Argon2id with memory=64MB, time=3, parallelism=4
- Tokens generated using crypto/rand for cryptographic randomness
- Timing attack prevention via constant-time comparisons and randomized delays
- Session fixation prevention via token rotation on each use
- CSRF protection via state parameter in OAuth flows
- Cookie security: HttpOnly, Secure, SameSite=Strict

### 4. Component Interfaces

#### 4.1 AuthController Public Methods

```go
// NewAuthController creates a new AuthController instance
func NewAuthController(
	userRepo UserRepository,
	jwtManager *JWTManager,
	passwordHasher *PasswordHasher,
	sessionManager *SessionManager,
	oauthHandler *OAuthHandler,
	lockoutTracker *AccountLockoutTracker,
	redisClient *redis.Client,
	config *Config,
) *AuthController

// Register handles user registration
// POST /api/auth/register
func (c *AuthController) Register(ctx *fiber.Ctx) error

// Login handles email/password authentication
// POST /api/auth/login
func (c *AuthController) Login(ctx *fiber.Ctx) error

// Refresh handles access token refresh
// POST /api/auth/refresh
func (c *AuthController) Refresh(ctx *fiber.Ctx) error

// Logout handles user logout
// POST /api/auth/logout
func (c *AuthController) Logout(ctx *fiber.Ctx) error

// OAuthBegin initiates OAuth flow
// GET /api/auth/oauth/:provider
func (c *AuthController) OAuthBegin(ctx *fiber.Ctx) error

// OAuthCallback handles OAuth provider callback
// GET /api/auth/oauth/:provider/callback
func (c *AuthController) OAuthCallback(ctx *fiber.Ctx) error

// RequestPasswordReset initiates password reset flow
// POST /api/auth/password-reset/request
func (c *AuthController) RequestPasswordReset(ctx *fiber.Ctx) error

// ResetPassword completes password reset
// POST /api/auth/password-reset/confirm
func (c *AuthController) ResetPassword(ctx *fiber.Ctx) error

// VerifyEmail confirms email verification
// POST /api/auth/verify-email
func (c *AuthController) VerifyEmail(ctx *fiber.Ctx) error

// GetSession returns current session state
// GET /api/auth/session
func (c *AuthController) GetSession(ctx *fiber.Ctx) error
```

#### 4.2 JWTManager Interface

```go
type JWTManager struct {
	secretKey []byte
}

// NewJWTManager creates a new JWT manager
func NewJWTManager(secretKey []byte) *JWTManager

// IssueAccessToken generates a short-lived access token
func (m *JWTManager) IssueAccessToken(user *User) (string, error)

// IssueRefreshToken generates a long-lived refresh token
func (m *JWTManager) IssueRefreshToken(user *User) (string, error)

// IssueTokens generates both access and refresh tokens
func (m *JWTManager) IssueTokens(user *User) (*AuthTokens, error)

// ValidateAccessToken validates an access token
func (m *JWTManager) ValidateAccessToken(tokenString string) (*TokenClaims, error)

// ValidateRefreshToken validates a refresh token
func (m *JWTManager) ValidateRefreshToken(tokenString string) (*TokenClaims, error)

// GetTokenID extracts the unique token ID from a refresh token
func (m *JWTManager) GetTokenID(tokenString string) (string, error)
```

#### 4.3 PasswordHasher Interface

```go
type PasswordHasher struct {
	time    int
	memory  int
	parallelism int
	keyLen  int
	saltLen int
}

// NewPasswordHasher creates a new Argon2 hasher
func NewPasswordHasher() *PasswordHasher

// Hash generates Argon2id hash with random salt
func (h *PasswordHasher) Hash(password string, salt []byte) (string, error)

// Verify compares password against hash using constant-time comparison
func (h *PasswordHasher) Verify(storedHash, password string) (bool, error)

// GenerateSalt creates a cryptographically random salt
func (h *PasswordHasher) GenerateSalt() ([]byte, error)
```

#### 4.4 SessionManager Interface

```go
type SessionManager struct {
	redisClient *redis.Client
	sessionTTL  time.Duration
}

// NewSessionManager creates a new session manager
func NewSessionManager(redisClient *redis.Client) *SessionManager

// StoreRefreshToken stores refresh token in Redis
func (m *SessionManager) StoreRefreshToken(ctx context.Context, userID string, token *RefreshTokenRecord) error

// GetRefreshToken retrieves refresh token record
func (m *SessionManager) GetRefreshToken(ctx context.Context, userID, tokenID string) (*RefreshTokenRecord, error)

// RevokeRefreshToken removes a specific refresh token
func (m *SessionManager) RevokeRefreshToken(ctx context.Context, userID, tokenID string) error

// RevokeAllUserTokens removes all refresh tokens for a user
func (m *SessionManager) RevokeAllUserTokens(ctx context.Context, userID string) error

// StoreOAuthSession stores OAuth state temporarily
func (m *SessionManager) StoreOAuthSession(ctx context.Context, state string, session *OAuthSession) error

// GetOAuthSession retrieves OAuth state
func (m *SessionManager) GetOAuthSession(ctx context.Context, state string) (*OAuthSession, error)

// DeleteOAuthSession removes OAuth state
func (m *SessionManager) DeleteOAuthSession(ctx context.Context, state string) error

// StorePasswordResetToken stores password reset token
func (m *SessionManager) StorePasswordResetToken(ctx context.Context, userID string, token *PasswordResetToken) error

// GetPasswordResetToken retrieves password reset token
func (m *SessionManager) GetPasswordResetToken(ctx context.Context, userID string) (*PasswordResetToken, error)

// MarkPasswordResetTokenUsed marks token as used
func (m *SessionManager) MarkPasswordResetTokenUsed(ctx context.Context, userID string) error
```

#### 4.5 AccountLockoutTracker Interface

```go
type AccountLockoutTracker struct {
	redisClient *redis.Client
	config      *Config
}

// NewAccountLockoutTracker creates a new lockout tracker
func NewAccountLockoutTracker(redisClient *redis.Client, config *Config) *AccountLockoutTracker

// CheckAccountLockout returns lockout state for user account
func (t *AccountLockoutTracker) CheckAccountLockout(ctx context.Context, userID string) (*AccountLockoutState, error)

// RecordAccountFailure records a failed login attempt
func (t *AccountLockoutTracker) RecordAccountFailure(ctx context.Context, userID string) error

// ClearAccountFailures resets failed login counter
func (t *AccountLockoutTracker) ClearAccountFailures(ctx context.Context, userID string) error

// CheckIPLockout returns lockout state for IP address
func (t *AccountLockoutTracker) CheckIPLockout(ctx context.Context, ipAddress string) (*IPLockoutState, error)

// RecordIPFailure records a failed attempt from IP
func (t *AccountLockoutTracker) RecordIPFailure(ctx context.Context, ipAddress string) error
```

#### 4.6 OAuthHandler Interface

```go
type OAuthHandler struct {
	callbackURL string
	providers   map[string]goth.Provider
}

// NewOAuthHandler creates a new OAuth handler
func NewOAuthHandler(callbackURL string) *OAuthHandler

// RegisterProvider registers an OAuth provider
func (h *OAuthHandler) RegisterProvider(provider string, key, secret string, endpoints map[string]string) error

// BeginAuth initiates OAuth flow
func (h *OAuthHandler) BeginAuth(ctx context.Context, state, provider string) (string, error)

// CompleteAuth completes OAuth flow and returns user info
func (h *OAuthHandler) CompleteAuth(ctx context.Context, provider, code, state string) (*goth.User, error)

// GetAuthURL returns OAuth authorization URL
func (h *OAuthHandler) GetAuthURL(provider, state string) (string, error)
```
