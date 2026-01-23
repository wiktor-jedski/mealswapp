# OAuthHandler (goth)

**Traceability:** ARCH-006

---

## 1. Data Structures & Types

### 1.1 OAuth Provider Configuration

```go
type OAuthProvider string

const (
    OAuthProviderGoogle OAuthProvider = "google"
    OAuthProviderApple  OAuthProvider = "apple"
)

type OAuthConfig struct {
    Provider       OAuthProvider
    ClientID       string
    ClientSecret   string
    RedirectURL    string
    Scopes         []string
    AuthURL        string
    TokenURL       string
    UserInfoURL    string
}
```

### 1.2 OAuth Session State

```go
type OAuthSessionState struct {
    ID          string            `json:"id"`
    Provider    OAuthProvider     `json:"provider"`
    State       string            `json:"state"`
    Nonce       string            `json:"nonce"`
    RedirectURL string            `json:"redirect_url"`
    ExpiresAt   time.Time         `json:"expires_at"`
    CreatedAt   time.Time         `json:"created_at"`
}
```

### 1.3 Goth User Data

```go
type GothUser struct {
    ID            string         `json:"id"`
    Email         string         `json:"email"`
    EmailVerified bool           `json:"email_verified"`
    Name          string         `json:"name"`
    FirstName     string         `json:"first_name"`
    LastName      string         `json:"last_name"`
    AvatarURL     string         `json:"avatar_url"`
    Provider      OAuthProvider  `json:"provider"`
    RawData       map[string]any `json:"raw_data"`
}
```

### 1.4 OAuth Callback Request

```go
type OAuthCallbackRequest struct {
    Provider   OAuthProvider `query:"provider" validate:"required"`
    Code       string        `query:"code" validate:"required"`
    State      string        `query:"state" validate:"required"`
    Error      string        `query:"error,omitempty"`
    ErrorDesc  string        `query:"error_description,omitempty"`
    ErrorURI   string        `query:"error_uri,omitempty"`
}
```

### 1.5 OAuth Token Response

```go
type OAuthTokenResponse struct {
    AccessToken  string    `json:"access_token"`
    TokenType    string    `json:"token_type"`
    ExpiresIn    int       `json:"expires_in"`
    RefreshToken string    `json:"refresh_token,omitempty"`
    Scope        string    `json:"scope,omitempty"`
    IDToken      string    `json:"id_token,omitempty"`
}
```

### 1.6 User Account Link Result

```go
type OAuthAccountLinkResult struct {
    UserID          string    `json:"user_id"`
    IsNewUser       bool      `json:"is_new_user"`
    TrialGranted    bool      `json:"trial_granted"`
    TrialExpiresAt  time.Time `json:"trial_expires_at"`
    SessionID       string    `json:"session_id"`
}
```

### 1.7 OAuth Flow Result

```go
type OAuthFlowResult struct {
    Success       bool                   `json:"success"`
    RedirectURL   string                 `json:"redirect_url,omitempty"`
    Error         string                 `json:"error,omitempty"`
    ErrorCode     string                 `json:"error_code,omitempty"`
    User          *GothUser              `json:"user,omitempty"`
    Tokens        *TokenSet              `json:"tokens,omitempty"`
    AccountLink   *OAuthAccountLinkResult `json:"account_link,omitempty"`
}
```

### 1.8 Token Set (Internal)

```go
type TokenSet struct {
    AccessToken  string    `json:"access_token"`
    RefreshToken string    `json:"refresh_token,omitempty"`
    ExpiresAt    time.Time `json:"expires_at"`
}
```

---

## 2. Logic & Algorithms (Step-by-Step)

### 2.1 Initiate OAuth Flow

**Purpose:** Redirect user to OAuth provider for authentication.

```
FUNCTION InitiateOAuthFlow(provider: OAuthProvider, redirectURL: string): OAuthFlowResult

1. Validate provider is supported (google or apple)
   IF provider NOT IN [google, apple] THEN
       RETURN OAuthFlowResult{Success: false, Error: "UNSUPPORTED_PROVIDER", ErrorCode: "ERR_OAUTH_001"}
   END IF

2. Generate cryptographically secure state token (32 bytes, URL-safe)
   state := generateSecureToken(32)

3. Generate nonce for PKCE (if required by provider)
   nonce := generateSecureToken(32)

4. Retrieve OAuth configuration for provider
   config := getOAuthConfig(provider)
   IF config IS nil THEN
       RETURN OAuthFlowResult{Success: false, Error: "CONFIG_NOT_FOUND", ErrorCode: "ERR_OAUTH_002"}
   END IF

5. Store session state in Redis with 10-minute expiration
   sessionState := OAuthSessionState{
       ID: generateUUID(),
       Provider: provider,
       State: state,
       Nonce: nonce,
       RedirectURL: redirectURL,
       ExpiresAt: now.Add(10 * time.Minute),
       CreatedAt: now,
   }
   redis.Set(ctx, "oauth_state:"+state, sessionState, 10*time.Minute)

6. Construct authorization URL with parameters
   authURL := buildAuthURL(config, state, nonce)
   - client_id
   - redirect_uri (from config)
   - response_type: "code"
   - scope: join(config.Scopes, " ")
   - state: state
   - nonce: nonce (for ID token validation)

7. RETURN OAuthFlowResult{Success: true, RedirectURL: authURL}
END FUNCTION
```

### 2.2 Handle OAuth Callback

**Purpose:** Process OAuth callback, exchange code for tokens, retrieve user info.

```
FUNCTION HandleOAuthCallback(req: OAuthCallbackRequest): OAuthFlowResult

1. Validate callback request parameters
   IF req.Error IS NOT empty THEN
       RETURN handleOAuthError(req.Error, req.ErrorDesc, req.Provider)
   END IF

   IF req.Code IS empty OR req.State IS empty THEN
       RETURN OAuthFlowResult{Success: false, Error: "INVALID_CALLBACK_PARAMS", ErrorCode: "ERR_OAUTH_003"}
   END IF

2. Retrieve and validate session state from Redis
   storedState := redis.Get(ctx, "oauth_state:"+req.State)
   IF storedState IS nil THEN
       RETURN OAuthFlowResult{Success: false, Error: "STATE_NOT_FOUND", ErrorCode: "ERR_OAUTH_004"}
   END IF

   IF storedState.ExpiresAt.Before(now) THEN
       RETURN OAuthFlowResult{Success: false, Error: "STATE_EXPIRED", ErrorCode: "ERR_OAUTH_005"}
   END IF

   IF storedState.Provider != req.Provider THEN
       RETURN OAuthFlowResult{Success: false, Error: "PROVIDER_MISMATCH", ErrorCode: "ERR_OAUTH_006"}
   END IF

3. Delete used state from Redis (one-time use)
   redis.Del(ctx, "oauth_state:"+req.State)

4. Retrieve OAuth configuration for provider
   config := getOAuthConfig(req.Provider)
   IF config IS nil THEN
       RETURN OAuthFlowResult{Success: false, Error: "CONFIG_NOT_FOUND", ErrorCode: "ERR_OAUTH_002"}
   END IF

5. Exchange authorization code for access token
   tokenResponse := exchangeCodeForToken(config, req.Code)
   IF tokenResponse.Error IS NOT nil THEN
       RETURN OAuthFlowResult{Success: false, Error: tokenResponse.Error, ErrorCode: "ERR_OAUTH_007", ErrorDesc: tokenResponse.ErrorDescription}
   END IF

6. Retrieve user info from provider using access token
   userInfo := fetchUserInfo(config, tokenResponse.AccessToken)
   IF userInfo.Error IS NOT nil THEN
       RETURN OAuthFlowResult{Success: false, Error: userInfo.Error, ErrorCode: "ERR_OAUTH_008"}
   END IF

7. Validate ID token if present (Google/Apple)
   IF tokenResponse.IDToken IS NOT empty THEN
       idTokenValid := validateIDToken(tokenResponse.IDToken, config, userInfo.Nonce)
       IF NOT idTokenValid THEN
           RETURN OAuthFlowResult{Success: false, Error: "INVALID_ID_TOKEN", ErrorCode: "ERR_OAUTH_009"}
       END IF
   END IF

8. Create or link user account
   linkResult := linkOAuthAccount(userInfo, req.Provider)
   IF linkResult.Error IS NOT nil THEN
       RETURN OAuthFlowResult{Success: false, Error: linkResult.Error, ErrorCode: linkResult.ErrorCode}
   END IF

9. Create application session
   sessionID := createSession(linkResult.UserID, storedState.RedirectURL)

10. RETURN OAuthFlowResult{
       Success: true,
       User: userInfo,
       Tokens: tokenResponse,
       AccountLink: linkResult,
       RedirectURL: storedState.RedirectURL + "?session=" + sessionID
   }
END FUNCTION
```

### 2.3 Link OAuth Account

**Purpose:** Create new user or link to existing account.

```
FUNCTION linkOAuthAccount(user: GothUser, provider: OAuthProvider): OAuthAccountLinkResult

1. Check if email already exists in database
   existingUser := userRepository.FindByEmail(user.Email)
   IF existingUser IS nil THEN
       // Create new user account
       newUser := User{
           Email: user.Email,
           EmailVerified: user.EmailVerified,
           Name: user.Name,
           AvatarURL: user.AvatarURL,
           OAuthProvider: provider,
           OAuthID: user.ID,
           CreatedAt: now,
           UpdatedAt: now,
       }
       userID := userRepository.Create(newUser)
       
       // Record OAuth account link
       oauthAccount := OAuthAccount{
           UserID: userID,
           Provider: provider,
           ProviderUserID: user.ID,
           Email: user.Email,
           AccessToken: encryptedAccessToken,
           RefreshToken: encryptedRefreshToken,
           TokenExpiresAt: tokenExpiresAt,
           CreatedAt: now,
       }
       oauthRepository.Create(oauthAccount)

       // Grant 7-day trial
       trialExpiresAt := now.Add(7 * 24 * time.Hour)
       subscriptionRepository.GrantTrial(userID, trialExpiresAt)

       RETURN OAuthAccountLinkResult{
           UserID: userID,
           IsNewUser: true,
           TrialGranted: true,
           TrialExpiresAt: trialExpiresAt,
           SessionID: sessionID,
       }
   END IF

2. User exists - check if already linked to this provider
   existingOAuthAccount := oauthRepository.FindByUserAndProvider(existingUser.ID, provider)
   IF existingOAuthAccount IS NOT nil THEN
       // Update tokens
       oauthRepository.UpdateTokens(existingOAuthAccount.ID, encryptedAccessToken, encryptedRefreshToken, tokenExpiresAt)
       RETURN OAuthAccountLinkResult{
           UserID: existingUser.ID,
           IsNewUser: false,
           TrialGranted: false,
           SessionID: sessionID,
       }
   END IF

3. User exists but not linked to this provider
   // Check if user has password set
   IF existingUser.PasswordHash IS NOT empty THEN
       // User has password - offer to link account
       RETURN OAuthAccountLinkResult{
           UserID: existingUser.ID,
           IsNewUser: false,
           TrialGranted: false,
           Error: "ACCOUNT_EXISTS_WITH_PASSWORD",
           ErrorCode: "ERR_OAUTH_010",
       }
   END IF

4. Link OAuth account to existing user
   oauthAccount := OAuthAccount{
       UserID: existingUser.ID,
       Provider: provider,
       ProviderUserID: user.ID,
       Email: user.Email,
       AccessToken: encryptedAccessToken,
       RefreshToken: encryptedRefreshToken,
       TokenExpiresAt: tokenExpiresAt,
       CreatedAt: now,
   }
   oauthRepository.Create(oauthAccount)

   // Update user profile with OAuth data
   userRepository.UpdateProfile(existingUser.ID, user.Name, user.AvatarURL, provider, user.ID)

   RETURN OAuthAccountLinkResult{
       UserID: existingUser.ID,
       IsNewUser: false,
       TrialGranted: false,
       SessionID: sessionID,
   }
END FUNCTION
```

### 2.4 Token Exchange (HTTP)

**Purpose:** Exchange authorization code for tokens.

```
FUNCTION exchangeCodeForToken(config: OAuthConfig, code: string): OAuthTokenResponse

1. Construct token request body (application/x-www-form-urlencoded)
   body := url.Values{
       "grant_type": {"authorization_code"},
       "code": {code},
       "client_id": {config.ClientID},
       "client_secret": {config.ClientSecret},
       "redirect_uri": {config.RedirectURL},
   }

2. Make POST request to token URL
   response := http.PostForm(config.TokenURL, body)
   IF response.StatusCode != 200 THEN
       errorBody := parseErrorResponse(response.Body)
       RETURN OAuthTokenResponse{Error: errorBody.Error, ErrorDescription: errorBody.ErrorDescription}
   END IF

3. Parse token response
   tokenResp := parseTokenResponse(response.Body)

4. Encrypt refresh token before storage
   encryptedRefreshToken := encryptToken(tokenResp.RefreshToken)

5. RETURN tokenResp with encrypted refresh token
END FUNCTION
```

### 2.5 Fetch User Info

**Purpose:** Retrieve user profile from OAuth provider.

```
FUNCTION fetchUserInfo(config: OAuthConfig, accessToken: string): GothUser

1. Make authenticated GET request to user info URL
   req := http.NewRequest("GET", config.UserInfoURL, nil)
   req.Header.Set("Authorization", "Bearer "+accessToken)
   response := http.Do(req)

2. Parse user info response
   userInfo := parseUserInfoResponse(response.Body)

3. Normalize user data
   userInfo.Provider = config.Provider

4. RETURN userInfo
END FUNCTION
```

### 2.6 Create Application Session

**Purpose:** Create Fiber session for authenticated user.

```
FUNCTION createSession(userID: string, redirectURL: string): string

1. Generate session ID
   sessionID := generateSecureToken(32)

2. Create session data
   sessionData := SessionData{
       UserID: userID,
       CreatedAt: now,
       ExpiresAt: now.Add(7 * 24 * time.Hour), // 7-day session
       IPAddress: getClientIP(),
       UserAgent: getClientUserAgent(),
   }

3. Store in Redis with session middleware
   fiberSession := sessionManager.Get(sessionID)
   fiberSession.Set("user_id", userID)
   fiberSession.Set("created_at", now.Unix())
   fiberSession.Set("redirect_url", redirectURL)
   fiberSession.Save()

4. RETURN sessionID
END FUNCTION
```

---

## 3. State Management & Error Handling

### 3.1 State Machine

```
                    ┌─────────────────────────────────────────┐
                    │              IDLE                       │
                    │   Waiting for OAuth flow initiation     │
                    └─────────────────┬───────────────────────┘
                                      │ InitiateOAuthFlow()
                                      ▼
                    ┌─────────────────────────────────────────┐
                    │         STATE_GENERATED                 │
                    │   State token stored, awaiting callback │
                    └─────────────────┬───────────────────────┘
                                      │ HandleOAuthCallback()
                                      ▼
                    ┌─────────────────────────────────────────┐
                    │          TOKEN_EXCHANGE                 │
                    │   Exchanging code for access token      │
                    └─────────────────┬───────────────────────┘
                                      │ Token received
                                      ▼
                    ┌─────────────────────────────────────────┐
                    │           USER_FETCH                    │
                    │   Fetching user info from provider      │
                    └─────────────────┬───────────────────────┘
                                      │ User info received
                                      ▼
                    ┌─────────────────────────────────────────┐
                    │         ACCOUNT_LINK                    │
                    │   Creating/linking user account         │
                    └─────────────────┬───────────────────────┘
                                      │ Account linked
                                      ▼
                    ┌─────────────────────────────────────────┐
                    │           SESSION_CREATE                │
                    │   Creating application session          │
                    └─────────────────┬───────────────────────┘
                                      │ Session created
                                      ▼
                    ┌─────────────────────────────────────────┐
                    │          COMPLETED                      │
                    │   OAuth flow complete, redirecting      │
                    └─────────────────────────────────────────┘
```

### 3.2 Error States and Transitions

| Error Code | Error Condition | HTTP Status | User Message | Recovery Action |
|------------|-----------------|-------------|--------------|-----------------|
| ERR_OAUTH_001 | Unsupported OAuth provider | 400 | "Authentication provider not supported" | Use supported provider (google, apple) |
| ERR_OAUTH_002 | OAuth config not found | 500 | "Service temporarily unavailable" | Retry later, contact support if persists |
| ERR_OAUTH_003 | Missing callback parameters | 400 | "Invalid authentication response" | Restart OAuth flow |
| ERR_OAUTH_004 | State token not found | 400 | "Authentication session expired" | Restart OAuth flow |
| ERR_OAUTH_005 | State token expired | 400 | "Authentication session expired" | Restart OAuth flow |
| ERR_OAUTH_006 | Provider mismatch | 400 | "Authentication provider mismatch" | Restart OAuth flow with correct provider |
| ERR_OAUTH_007 | Token exchange failed | 502 | "Failed to complete authentication" | Restart OAuth flow |
| ERR_OAUTH_008 | User info fetch failed | 502 | "Failed to retrieve user information" | Retry authentication |
| ERR_OAUTH_009 | ID token validation failed | 401 | "Authentication verification failed" | Retry authentication |
| ERR_OAUTH_010 | Account exists with password | 409 | "An account with this email already exists" | Login with password to link accounts |
| ERR_OAUTH_011 | Network error during token exchange | 503 | "Network error, please try again" | Retry authentication |
| ERR_OAUTH_012 | Rate limit exceeded | 429 | "Too many authentication attempts" | Wait 15 minutes before retry |

### 3.3 State Persistence

**Redis Keys:**
- `oauth_state:{state}` - OAuth session state (10-minute TTL)
- `oauth_rate_limit:{ip}` - Rate limiting counter (15-minute TTL)
- `session:{session_id}` - Application session data (7-day TTL)

**Database Tables:**
- `users` - User accounts
- `oauth_accounts` - Linked OAuth accounts
- `user_subscriptions` - Trial/subscription status

### 3.4 Retry and Recovery

1. **Automatic Retry (Network Errors):**
   - Token exchange: Max 2 retries with exponential backoff (100ms, 500ms)
   - User info fetch: Max 2 retries with exponential backoff

2. **User-Initiated Recovery:**
   - Expired state: User must restart OAuth flow
   - Rate limited: User must wait 15 minutes
   - Account conflict: User can login with password first, then link OAuth

---

## 4. Component Interfaces

### 4.1 Public Functions

```go
// InitiateOAuthFlow initiates OAuth authentication for a provider
// Parameters:
//   - ctx: Fiber context for request handling
//   - provider: OAuth provider (google or apple)
//   - redirectURL: URL to redirect after successful auth
// Returns:
//   - *OAuthFlowResult: Contains redirect URL or error
func InitiateOAuthFlow(ctx *fiber.Ctx, provider OAuthProvider, redirectURL string) *OAuthFlowResult

// HandleOAuthCallback processes OAuth callback from provider
// Parameters:
//   - ctx: Fiber context for request handling
// Returns:
//   - *OAuthFlowResult: Contains user data, tokens, and session info
func HandleOAuthCallback(ctx *fiber.Ctx) *OAuthFlowResult

// RevokeOAuthToken revokes OAuth access token
// Parameters:
//   - ctx: Fiber context
//   - provider: OAuth provider
// Returns:
//   - error: nil on success
func RevokeOAuthToken(ctx *fiber.Ctx, provider OAuthProvider) error

// RefreshOAuthToken refreshes OAuth access token using refresh token
// Parameters:
//   - ctx: Fiber context
//   - provider: OAuth provider
// Returns:
//   - *TokenSet: New tokens or error
func RefreshOAuthToken(ctx *fiber.Ctx, provider OAuthProvider) (*TokenSet, error)
```

### 4.2 Internal Functions

```go
// getOAuthConfig retrieves OAuth configuration for provider
// Parameters:
//   - provider: OAuth provider
// Returns:
//   - *OAuthConfig: Configuration or nil if not found
func getOAuthConfig(provider OAuthProvider) *OAuthConfig

// generateStateToken generates cryptographically secure state token
// Parameters:
//   - length: Token length in bytes
// Returns:
//   - string: URL-safe state token
func generateStateToken(length int) string

// storeOAuthState stores OAuth state in Redis
// Parameters:
//   - ctx: Context
//   - state: OAuth session state
// Returns:
//   - error: Redis error
func storeOAuthState(ctx context.Context, state *OAuthSessionState) error

// getOAuthState retrieves OAuth state from Redis
// Parameters:
//   - ctx: Context
//   - state: State token
// Returns:
//   - *OAuthSessionState: Session state or nil
func getOAuthState(ctx context.Context, state string) *OAuthSessionState

// exchangeCodeForTokens exchanges authorization code for tokens
// Parameters:
//   - config: OAuth configuration
//   - code: Authorization code
// Returns:
//   - *OAuthTokenResponse: Token response or error
func exchangeCodeForTokens(config *OAuthConfig, code string) (*OAuthTokenResponse, error)

// fetchUserInfo fetches user information from OAuth provider
// Parameters:
//   - config: OAuth configuration
//   - accessToken: Access token
// Returns:
//   - *GothUser: User information or error
func fetchUserInfo(config *OAuthConfig, accessToken string) (*GothUser, error)

// linkOAuthAccount creates or links OAuth account
// Parameters:
//   - user: User info from OAuth provider
//   - provider: OAuth provider
// Returns:
//   - *OAuthAccountLinkResult: Result or error
func linkOAuthAccount(user *GothUser, provider OAuthProvider) *OAuthAccountLinkResult

// validateIDToken validates OAuth ID token
// Parameters:
//   - idToken: JWT token
//   - config: OAuth configuration
//   - nonce: Expected nonce value
// Returns:
//   - bool: Validation result
func validateIDToken(idToken string, config *OAuthConfig, nonce string) bool

// createApplicationSession creates Fiber session for authenticated user
// Parameters:
//   - ctx: Fiber context
//   - userID: User ID
//   - redirectURL: Original redirect URL
// Returns:
//   - string: Session ID
func createApplicationSession(ctx *fiber.Ctx, userID string, redirectURL string) string

// handleOAuthError processes OAuth error response
// Parameters:
//   - error: Error code from provider
//   - errorDesc: Error description
//   - provider: OAuth provider
// Returns:
//   - *OAuthFlowResult: Error result
func handleOAuthError(error string, errorDesc string, provider OAuthProvider) *OAuthFlowResult

// encryptToken encrypts token for storage
// Parameters:
//   - token: Plain text token
// Returns:
//   - string: Encrypted token
func encryptToken(token string) string

// decryptToken decrypts token from storage
// Parameters:
//   - encrypted: Encrypted token
// Returns:
//   - string: Plain text token
func decryptToken(encrypted string) string
```

### 4.3 Configuration Interface

```go
// OAuthConfigLoader loads OAuth configuration from environment
type OAuthConfigLoader interface {
    Load(provider OAuthProvider) *OAuthConfig
}

// DefaultOAuthConfigLoader implements OAuthConfigLoader
type DefaultOAuthConfigLoader struct{}

// Load returns OAuth configuration from environment variables
// Environment variables:
//   - GOOGLE_CLIENT_ID
//   - GOOGLE_CLIENT_SECRET
//   - GOOGLE_REDIRECT_URL
//   - APPLE_CLIENT_ID
//   - APPLE_CLIENT_SECRET
//   - APPLE_REDIRECT_URL
func (l *DefaultOAuthConfigLoader) Load(provider OAuthProvider) *OAuthConfig
```

### 4.4 Fiber Routes

```go
// RegisterRoutes registers OAuth handler routes
// Route: /auth/oauth/:provider
// Method: GET - Initiate OAuth flow
// Route: /auth/oauth/:provider/callback
// Method: GET - Handle OAuth callback
// Route: /auth/oauth/:provider/revoke
// Method: POST - Revoke OAuth token
func RegisterRoutes(app *fiber.App, handler *OAuthHandler)
```

### 4.5 Error Types

```go
var (
    ErrUnsupportedProvider    = errors.New("unsupported OAuth provider")
    ErrStateNotFound          = errors.New("OAuth state not found")
    ErrStateExpired           = errors.New("OAuth state expired")
    ErrProviderMismatch       = errors.New("OAuth provider mismatch")
    ErrTokenExchangeFailed    = errors.New("token exchange failed")
    ErrUserInfoFetchFailed    = errors.New("user info fetch failed")
    ErrIDTokenValidationFailed = errors.New("ID token validation failed")
    ErrAccountExistsWithPassword = errors.New("account exists with password")
    ErrRateLimitExceeded      = errors.New("rate limit exceeded")
)
```
