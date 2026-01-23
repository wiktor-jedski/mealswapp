# SessionManager (Fiber Session Middleware)

**Traceability:** ARCH-006

---

## 1. Data Structures & Types

### 1.1 Session Data Structure

```go
package auth

type SessionData struct {
    UserID          string    `json:"user_id"`
    SessionID       string    `json:"session_id"`
    AccessToken     string    `json:"access_token"`
    RefreshToken    string    `json:"refresh_token"`
    CreatedAt       time.Time `json:"created_at"`
    LastAccessedAt  time.Time `json:"last_accessed_at"`
    ExpiresAt       time.Time `json:"expires_at"`
    RefreshTokenRotatedAt time.Time `json:"refresh_token_rotated_at"`
    IPAddress       string    `json:"ip_address"`
    UserAgent       string    `json:"user_agent"`
    IsOAuthUser     bool      `json:"is_oauth_user"`
    TrialActive     bool      `json:"trial_active"`
    TrialExpiresAt  *time.Time `json:"trial_expires_at,omitempty"`
}
```

### 1.2 Session Configuration

```go
package auth

import (
    "github.com/gofiber/fiber/v2"
    "github.com/gofiber/session/v2"
)

type SessionConfig struct {
    CookieName           string
    CookieDomain         string
    CookieHTTPOnly       bool
    CookieSecure         bool
    CookieSameSite       string
    SessionExpiry        time.Duration
    RefreshTokenExpiry   time.Duration
    Storage              fiber.Storage
    KeyGenerator         func() string
}
```

### 1.3 Session Store Interface

```go
package auth

type SessionStore interface {
    Create(ctx *fiber.Ctx, data *SessionData) error
    Get(ctx *fiber.Ctx) (*SessionData, error)
    Update(ctx *fiber.Ctx, data *SessionData) error
    Refresh(ctx *fiber.Ctx) error
    Destroy(ctx *fiber.Ctx) error
    Regenerate(ctx *fiber.Ctx) error
    Validate(ctx *fiber.Ctx) (bool, error)
    ExtendExpiry(ctx *fiber.Ctx) error
}
```

### 1.4 Token Claims Structure

```go
package auth

import "github.com/golang-jwt/jwt/v5"

type AccessTokenClaims struct {
    UserID      string `json:"user_id"`
    SessionID   string `json:"session_id"`
    IsOAuthUser bool   `json:"is_oauth_user"`
    TrialActive bool   `json:"trial_active"`
    jwt.RegisteredClaims
}

type RefreshTokenClaims struct {
    UserID       string `json:"user_id"`
    SessionID    string `json:"session_id"`
    TokenVersion int    `json:"token_version"`
    jwt.RegisteredClaims
}
```

### 1.5 Session State Enumeration

```go
package auth

type SessionState string

const (
    SessionStateActive    SessionState = "active"
    SessionStateExpired   SessionState = "expired"
    SessionStateRevoked   SessionState = "revoked"
    SessionStateInvalid   SessionState = "invalid"
    SessionStateLocked    SessionState = "locked"
)
```

### 1.6 Error Types

```go
package auth

var (
    ErrSessionNotFound       = errors.New("session not found")
    ErrSessionExpired        = errors.New("session expired")
    ErrSessionInvalid        = errors.New("session invalid")
    ErrSessionRevoked        = errors.New("session revoked")
    ErrRefreshTokenInvalid   = errors.New("refresh token invalid")
    ErrRefreshTokenExpired   = errors.New("refresh token expired")
    ErrAccessTokenInvalid    = errors.New("access token invalid")
    ErrAccessTokenExpired    = errors.New("access token expired")
    ErrSessionLocked         = errors.New("session locked due to security policy")
    ErrCookieNotFound        = errors.New("cookie not found")
    ErrCookieInvalid         = errors.New("cookie invalid")
)
```

---

## 2. Logic & Algorithms

### 2.1 Session Creation Flow

```
ALGORITHM: CreateSession
INPUT: ctx (fiber.Ctx), userID (string), isOAuthUser (bool), trialActive (bool)
OUTPUT: *SessionData, error

1.  BEGIN
2.      sessionID ← KeyGenerator()
3.      accessToken ← GenerateAccessToken(userID, sessionID, isOAuthUser, trialActive)
4.      refreshToken ← GenerateRefreshToken(userID, sessionID)
5.      currentTime ← time.Now()
6.      expiresAt ← currentTime + config.AccessTokenExpiry
7.      refreshExpiresAt ← currentTime + config.RefreshTokenExpiry
8.      
9.      sessionData ← SessionData{
10.         UserID: userID,
11.         SessionID: sessionID,
12.         AccessToken: accessToken,
13.         RefreshToken: refreshToken,
14.         CreatedAt: currentTime,
15.         LastAccessedAt: currentTime,
16.         ExpiresAt: expiresAt,
17.         RefreshTokenRotatedAt: time.Time{},
18.         IPAddress: GetClientIP(ctx),
19.         UserAgent: GetClientUserAgent(ctx),
20.         IsOAuthUser: isOAuthUser,
21.         TrialActive: trialActive,
22.         TrialExpiresAt: nil
23.     }
24.     
25.     store ← GetSessionStore()
26.     IF store.Create(ctx, &sessionData) != nil THEN
27.         RETURN nil, ErrSessionCreationFailed
28.     END IF
29.     
30.     SetAuthCookies(ctx, accessToken, refreshToken)
31.     RETURN &sessionData, nil
32. END
```

### 2.2 Session Validation Flow

```
ALGORITHM: ValidateSession
INPUT: ctx (fiber.Ctx)
OUTPUT: (bool, *SessionData, error)

1.  BEGIN
2.     accessToken ← ExtractAccessToken(ctx)
3.     IF accessToken == nil THEN
4.         RETURN false, nil, ErrAccessTokenInvalid
5.     END IF
6.     
7.     claims ← ParseAccessToken(accessToken)
8.     IF claims == nil THEN
9.         RETURN false, nil, ErrAccessTokenInvalid
10.    END IF
11.    
12.    IF claims.ExpiresAt.Before(time.Now()) THEN
13.        RETURN false, nil, ErrAccessTokenExpired
14.    END IF
15.    
16.    sessionID ← claims.SessionID
17.    sessionData ← GetSessionFromStore(ctx, sessionID)
18.    IF sessionData == nil THEN
19.        RETURN false, nil, ErrSessionNotFound
20.    END IF
21.    
22.    IF sessionData.ExpiresAt.Before(time.Now()) THEN
23.        RETURN false, nil, ErrSessionExpired
24.    END IF
25.    
26.    IF sessionData.IPAddress != GetClientIP(ctx) THEN
27.        LogSecurityEvent(ctx, "IP_MISMATCH", sessionID)
28.        RETURN false, nil, ErrSessionInvalid
29.    END IF
30.    
31.    IF sessionData.RefreshTokenRotatedAt.After(claims.IssuedAt) THEN
32.        RETURN false, nil, ErrRefreshTokenInvalid
33.    END IF
34.    
35.    RETURN true, sessionData, nil
36. END
```

### 2.3 Refresh Token Rotation Flow

```
ALGORITHM: RotateRefreshToken
INPUT: ctx (fiber.Ctx)
OUTPUT: (string, string, error)

1.  BEGIN
2.     refreshToken ← ExtractRefreshToken(ctx)
3.     IF refreshToken == nil THEN
4.         RETURN nil, nil, ErrRefreshTokenInvalid
5.     END IF
6.     
7.     claims ← ParseRefreshToken(refreshToken)
8.     IF claims == nil THEN
9.         RETURN nil, nil, ErrRefreshTokenInvalid
10.    END IF
11.    
12.    IF claims.ExpiresAt.Before(time.Now()) THEN
13.        RETURN nil, nil, ErrRefreshTokenExpired
14.    END IF
15.    
16.     sessionData ← GetSessionFromStore(ctx, claims.SessionID)
17.     IF sessionData == nil THEN
18.         RETURN nil, nil, ErrSessionNotFound
19.     END IF
20.     
21.     IF sessionData.RefreshToken != refreshToken THEN
22.         LogSecurityEvent(ctx, "REFRESH_TOKEN_MISMATCH", claims.SessionID)
23.         DestroySession(ctx)
24.         RETURN nil, nil, ErrRefreshTokenInvalid
25.     END IF
26.     
27.     newAccessToken ← GenerateAccessToken(
28.         sessionData.UserID,
29.         sessionData.SessionID,
30.         sessionData.IsOAuthUser,
31.         sessionData.TrialActive
32.     )
33.     
34.     newRefreshToken ← GenerateRefreshToken(
35.         sessionData.UserID,
36.         sessionData.SessionID
37.     )
38.     
39.     currentTime ← time.Now()
40.     sessionData.AccessToken ← newAccessToken
41.     sessionData.RefreshToken ← newRefreshToken
42.     sessionData.LastAccessedAt ← currentTime
43.     sessionData.RefreshTokenRotatedAt ← currentTime
44.     sessionData.ExpiresAt ← currentTime + config.AccessTokenExpiry
45.     
46.     store ← GetSessionStore()
47.     store.Update(ctx, sessionData)
48.     
49.     SetAuthCookies(ctx, newAccessToken, newRefreshToken)
50.     RETURN newAccessToken, newRefreshToken, nil
51. END
```

### 2.4 Session Destruction Flow

```
ALGORITHM: DestroySession
INPUT: ctx (fiber.Ctx)
OUTPUT: error

1.  BEGIN
2.     accessToken ← ExtractAccessToken(ctx)
3.     IF accessToken == nil THEN
4.         RETURN nil
5.     END IF
6.     
7.     claims ← ParseAccessToken(accessToken)
8.     IF claims == nil THEN
9.         RETURN nil
10.    END IF
11.    
12.     sessionID ← claims.SessionID
13.     store ← GetSessionStore()
14.     store.Destroy(ctx, sessionID)
15.     
16.     ClearAuthCookies(ctx)
17.     RETURN nil
18. END
```

### 2.5 Session Middleware Handler

```
ALGORITHM: SessionMiddleware
INPUT: ctx (fiber.Ctx) → next (fiber.Handler)
OUTPUT: error

1.  BEGIN
2.     isValid, sessionData, err ← ValidateSession(ctx)
3.     
4.     IF err == ErrAccessTokenExpired THEN
5.         newAccessToken, newRefreshToken, refreshErr ← RotateRefreshToken(ctx)
6.         IF refreshErr == nil THEN
7.             SetAuthCookies(ctx, newAccessToken, newRefreshToken)
8.             AttachSessionToContext(ctx, sessionData)
9.             RETURN next(ctx)
10.        END IF
11.        
12.        ClearAuthCookies(ctx)
13.        RETURN next(ctx)
14.    END IF
15.    
16.    IF isValid THEN
17.        IF time.Since(sessionData.LastAccessedAt) > config.SessionExpiry/2 THEN
18.            ExtendExpiry(ctx)
19.        END IF
20.        AttachSessionToContext(ctx, sessionData)
21.        RETURN next(ctx)
22.    END IF
23.    
24.    IF err == ErrSessionNotFound OR err == ErrSessionExpired THEN
25.         ClearAuthCookies(ctx)
26.    END IF
27.    
28.     RETURN next(ctx)
29. END
```

### 2.6 Access Token Generation Algorithm

```
ALGORITHM: GenerateAccessToken
INPUT: userID (string), sessionID (string), isOAuthUser (bool), trialActive (bool)
OUTPUT: string

1.  BEGIN
2.     currentTime ← time.Now()
3.     expiresAt ← currentTime + 15 * time.Minute
4.     
5.     claims ← AccessTokenClaims{
6.         UserID: userID,
7.         SessionID: sessionID,
8.         IsOAuthUser: isOAuthUser,
9.         TrialActive: trialActive,
10.        RegisteredClaims: jwt.RegisteredClaims{
11.            Issuer: "mealswapp",
12.            Subject: userID,
13.            Audience: jwt.ClaimStrings{"mealswapp"},
14.            IssuedAt: jwt.NewNumericDate(currentTime),
15.            ExpiresAt: jwt.NewNumericDate(expiresAt),
16.            NotBefore: jwt.NewNumericDate(currentTime),
17.            ID: sessionID
18.        }
19.     }
20.     
21.     token ← jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
22.     signedToken ← token.SignedString(GetJWTSecret())
23.     RETURN signedToken
24. END
```

### 2.7 Refresh Token Generation Algorithm

```
ALGORITHM: GenerateRefreshToken
INPUT: userID (string), sessionID (string)
OUTPUT: string

1.  BEGIN
2.     currentTime ← time.Now()
3.     expiresAt ← currentTime + 7 * 24 * time.Hour
4.     tokenVersion ← atomic.AddInt32(&globalTokenVersion, 1)
5.     
6.     claims ← RefreshTokenClaims{
7.         UserID: userID,
8.         SessionID: sessionID,
9.         TokenVersion: int(tokenVersion),
10.        RegisteredClaims: jwt.RegisteredClaims{
11.            Issuer: "mealswapp",
12.            Subject: userID,
13.            Audience: jwt.ClaimStrings{"mealswapp"},
14.            IssuedAt: jwt.NewNumericDate(currentTime),
15.            ExpiresAt: jwt.NewNumericDate(expiresAt),
16.            NotBefore: jwt.NewNumericDate(currentTime),
17.             ID: GenerateUUID()
18.        }
19.     }
20.     
21.     token ← jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
22.     signedToken ← token.SignedString(GetJWTSecret())
23.     RETURN signedToken
24. END
```

---

## 3. State Management & Error Handling

### 3.1 Session State Transitions

```
STATE: Initial → Active
TRIGGER: CreateSession() successful
ACTION: Store session in Redis with expiry of 7 days
GUARD: User must be authenticated, IP must be valid

STATE: Active → Active
TRIGGER: ValidateSession() successful, ExtendExpiry() called
ACTION: Update LastAccessedAt, extend ExpiresAt by 15 minutes
GUARD: Session must not be expired, IP must match

STATE: Active → Expired
TRIGGER: Access token expiry reached (15 minutes)
ACTION: Access token becomes invalid, refresh token still valid
RECOVERY: RotateRefreshToken() can restore active state

STATE: Active → Revoked
TRIGGER: Logout, Account lockout, Security violation detected
ACTION: Delete session from Redis, clear cookies
GUARD: User request or security policy violation

STATE: Active → Invalid
TRIGGER: IP mismatch, User-Agent mismatch, Token replay detected
ACTION: Destroy session, log security event
RECOVERY: User must re-authenticate

STATE: Expired → Active
TRIGGER: RotateRefreshToken() with valid refresh token
ACTION: Generate new access token, new refresh token, reset expiry
GUARD: Refresh token must be valid and not expired (7 days)

STATE: Expired → Revoked
TRIGGER: Refresh token expiry reached (7 days)
ACTION: Delete session from Redis
RECOVERY: User must re-authenticate
```

### 3.2 Error States and Recovery

| Error State | Description | Recovery Action | User Impact |
|-------------|-------------|-----------------|-------------|
| `ErrSessionNotFound` | Session deleted from store | Clear cookies, redirect to login | User must log in |
| `ErrSessionExpired` | Session TTL exceeded | Attempt token rotation | Transparent if refresh succeeds |
| `ErrSessionInvalid` | Security validation failed | Destroy session, log event | User must log in |
| `ErrSessionLocked` | Account lockout active | Reject request | User blocked until lockout expires |
| `ErrRefreshTokenInvalid` | Token mismatch or replay | Destroy session | User must log in |
| `ErrRefreshTokenExpired` | Refresh token TTL exceeded | Destroy session | User must log in |
| `ErrAccessTokenInvalid` | Token signature invalid | Attempt rotation | Transparent if refresh succeeds |
| `ErrAccessTokenExpired` | Access token TTL exceeded | Attempt rotation | Transparent if refresh succeeds |

### 3.3 Security Event Logging

```go
package auth

type SecurityEvent struct {
    EventType   string
    SessionID   string
    UserID      string
    IPAddress   string
    UserAgent   string
    Timestamp   time.Time
    Details     string
}

func LogSecurityEvent(ctx *fiber.Ctx, eventType, sessionID string) {
    event := SecurityEvent{
        EventType:  eventType,
        SessionID:  sessionID,
        IPAddress:  GetClientIP(ctx),
        UserAgent:  GetClientUserAgent(ctx),
        Timestamp:  time.Now(),
    }
    
    switch eventType {
    case "IP_MISMATCH":
        event.UserID = GetUserIDFromSession(ctx)
        event.Details = fmt.Sprintf("IP address mismatch: stored=%s, current=%s", 
            GetStoredIP(ctx), event.IPAddress)
    case "REFRESH_TOKEN_MISMATCH":
        event.UserID = GetUserIDFromSession(ctx)
        event.Details = "Refresh token does not match stored token"
    case "UNAUTHORIZED_ACCESS_ATTEMPT":
        event.Details = "Attempted access without valid session"
    }
    
    go SendToSecurityLogger(event)
}
```

### 3.4 Concurrent Session Management

```go
package auth

type SessionManager struct {
    store           fiber.Storage
    mutex           sync.RWMutex
    activeSessions  map[string]int
    config          *SessionConfig
}

func (sm *SessionManager) GetActiveSessionCount(userID string) int {
    sm.mutex.RLock()
    defer sm.mutex.RUnlock()
    return sm.activeSessions[userID]
}

func (sm *SessionManager) RegisterSession(userID, sessionID string) {
    sm.mutex.Lock()
    defer sm.mutex.Unlock()
    sm.activeSessions[userID]++
}

func (sm *SessionManager) UnregisterSession(userID, sessionID string) {
    sm.mutex.Lock()
    defer sm.mutex.Unlock()
    if sm.activeSessions[userID] > 0 {
        sm.activeSessions[userID]--
    }
}

func (sm *SessionManager) RevokeAllUserSessions(userID string) error {
    sm.mutex.Lock()
    defer sm.mutex.Unlock()
    
    pattern := fmt.Sprintf("session:%s:*", userID)
    keys, _ := sm.store.Scan(0, pattern, 100)
    
    for _, key := range keys {
        sm.store.Delete(key)
    }
    
    sm.activeSessions[userID] = 0
    return nil
}
```

### 3.5 Session Validation Middleware Chain

```
MIDDLEWARE: ValidateSession
PRIORITY: High (before business logic)
ERROR HANDLING: Pass to error handler if critical security failure

1. Extract cookies from request
2. Validate access token format and signature
3. Check access token expiry
4. Retrieve session from Redis
5. Validate session state (not expired, not revoked)
6. Validate IP consistency
7. Validate user agent consistency
8. Update last accessed timestamp
9. Attach session to request context
10. Pass to next middleware or handler
```

### 3.6 Token Rotation Safety

```go
package auth

type TokenRotationPolicy struct {
    MaxRotationsPerSession     int
    RotationCooldown           time.Duration
    LastRotationTime           time.Time
}

func (trp *TokenRotationPolicy) CanRotate() bool {
    return time.Since(trp.LastRotationTime) > trp.RotationCooldown
}

func (trp *TokenRotationPolicy) RecordRotation() {
    trp.LastRotationTime = time.Now()
}

func (sm *SessionManager) SafeRotateRefreshToken(ctx *fiber.Ctx) (string, string, error) {
    policy := GetRotationPolicy(ctx)
    
    if !policy.CanRotate() {
        return "", "", ErrRotationCooldownNotElapsed
    }
    
    accessToken, refreshToken, err := sm.RotateRefreshToken(ctx)
    if err == nil {
        policy.RecordRotation()
    }
    
    return accessToken, refreshToken, err
}
```

---

## 4. Component Interfaces

### 4.1 SessionManager Public API

```go
package auth

type SessionManager interface {
    CreateSession(ctx *fiber.Ctx, userID string, isOAuthUser bool, trialActive bool) (*SessionData, error)
    ValidateSession(ctx *fiber.Ctx) (bool, *SessionData, error)
    RefreshSession(ctx *fiber.Ctx) (string, string, error)
    DestroySession(ctx *fiber.Ctx) error
    RegenerateSession(ctx *fiber.Ctx) (*SessionData, error)
    GetSession(ctx *fiber.Ctx) (*SessionData, error)
    ExtendSessionExpiry(ctx *fiber.Ctx) error
    RevokeUserSessions(userID string) error
    RevokeSession(sessionID string) error
    GetActiveSessionCount(userID string) int
}
```

### 4.2 SessionManager Implementation

```go
package auth

type sessionManager struct {
    config     *SessionConfig
    store      fiber.Storage
    jwtSecret  []byte
    policy     *TokenRotationPolicy
}

func NewSessionManager(config *SessionConfig, store fiber.Storage, jwtSecret []byte) SessionManager {
    return &sessionManager{
        config:    config,
        store:     store,
        jwtSecret: jwtSecret,
        policy: &TokenRotationPolicy{
            MaxRotationsPerSession: 100,
            RotationCooldown:       5 * time.Second,
        },
    }
}
```

### 4.3 CreateSession Method Signature

```go
func (sm *sessionManager) CreateSession(
    ctx *fiber.Ctx,
    userID string,
    isOAuthUser bool,
    trialActive bool,
) (*SessionData, error)
```

**Parameters:**
- `ctx`: Fiber request context containing client IP and user agent
- `userID`: UUID of the authenticated user
- `isOAuthUser`: Boolean indicating if user authenticated via OAuth
- `trialActive`: Boolean indicating if user has active trial

**Returns:**
- `*SessionData`: Complete session data including tokens
- `error`: Error if session creation fails

**Preconditions:**
- User must be authenticated via credentials or OAuth
- User account must not be locked
- Client IP must be valid and not blacklisted

**Postconditions:**
- Session stored in Redis with 7-day TTL
- HttpOnly cookies set with access and refresh tokens
- Session attached to request context

### 4.4 ValidateSession Method Signature

```go
func (sm *sessionManager) ValidateSession(ctx *fiber.Ctx) (bool, *SessionData, error)
```

**Parameters:**
- `ctx`: Fiber request context

**Returns:**
- `bool`: True if session is valid
- `*SessionData`: Session data if valid
- `error`: Error if validation fails

**Preconditions:**
- Access token cookie must be present
- Session must exist in Redis store

**Postconditions:**
- LastAccessedAt updated if session is valid
- Session attached to request context

### 4.5 RefreshSession Method Signature

```go
func (sm *sessionManager) RefreshSession(ctx *fiber.Ctx) (string, string, error)
```

**Parameters:**
- `ctx`: Fiber request context

**Returns:**
- `string`: New access token
- `string`: New refresh token
- `error`: Error if refresh fails

**Preconditions:**
- Refresh token cookie must be present and valid
- Refresh token must not be expired
- Session must exist in Redis store

**Postconditions:**
- New access token issued with 15-minute expiry
- Refresh token rotated with new 7-day expiry
- Old refresh token invalidated
- HttpOnly cookies updated with new tokens

### 4.6 DestroySession Method Signature

```go
func (sm *sessionManager) DestroySession(ctx *fiber.Ctx) error
```

**Parameters:**
- `ctx`: Fiber request context

**Returns:**
- `error`: Error if destruction fails

**Preconditions:**
- Session must exist in Redis store

**Postconditions:**
- Session deleted from Redis
- Access token and refresh token cookies cleared
- Session removed from request context

### 4.7 Fiber Middleware Registration

```go
package auth

func RegisterSessionMiddleware(app *fiber.App, manager SessionManager) {
    app.Use(func(c *fiber.Ctx) error {
        isValid, sessionData, err := manager.ValidateSession(c)
        
        if err == nil && isValid {
            c.Locals("session", sessionData)
            c.Locals("user_id", sessionData.UserID)
        }
        
        if err == ErrAccessTokenExpired {
            newAccess, newRefresh, refreshErr := manager.RefreshSession(c)
            if refreshErr == nil {
                SetAuthCookies(c, newAccess, newRefresh)
                c.Locals("session", sessionData)
                c.Locals("user_id", sessionData.UserID)
                return c.Next()
            }
            DestroySession(c)
            return c.Next()
        }
        
        return c.Next()
    })
}
```

### 4.8 Cookie Management Functions

```go
package auth

func SetAuthCookies(ctx *fiber.Ctx, accessToken, refreshToken string) {
    ctx.Cookie(&fiber.Cookie{
        Name:     "at",
        Value:    accessToken,
        Expires:  time.Now().Add(15 * time.Minute),
        HTTPOnly: true,
        Secure:   true,
        SameSite: "strict",
        Domain:   GetCookieDomain(),
    })
    
    ctx.Cookie(&fiber.Cookie{
        Name:     "rt",
        Value:    refreshToken,
        Expires:  time.Now().Add(7 * 24 * time.Hour),
        HTTPOnly: true,
        Secure:   true,
        SameSite: "strict",
        Domain:   GetCookieDomain(),
    })
}

func ClearAuthCookies(ctx *fiber.Ctx) {
    ctx.Cookie(&fiber.Cookie{
        Name:     "at",
        Value:    "",
        Expires:  time.Now().Add(-time.Hour),
        HTTPOnly: true,
        Secure:   true,
        SameSite: "strict",
        Domain:   GetCookieDomain(),
    })
    
    ctx.Cookie(&fiber.Cookie{
        Name:     "rt",
        Value:    "",
        Expires:  time.Now().Add(-time.Hour),
        HTTPOnly: true,
        Secure:   true,
        SameSite: "strict",
        Domain:   GetCookieDomain(),
    })
}

func ExtractAccessToken(ctx *fiber.Ctx) (string, error) {
    token := ctx.Cookies("at")
    if token == "" {
        return "", ErrCookieNotFound
    }
    return token, nil
}

func ExtractRefreshToken(ctx *fiber.Ctx) (string, error) {
    token := ctx.Cookies("rt")
    if token == "" {
        return "", ErrCookieNotFound
    }
    return token, nil
}
```

### 4.9 Redis Session Storage Implementation

```go
package auth

type redisSessionStore struct {
    client    *redis.Client
    keyPrefix string
}

func NewRedisSessionStore(client *redis.Client) *redisSessionStore {
    return &redisSessionStore{
        client:    client,
        keyPrefix: "session:",
    }
}

func (rs *redisSessionStore) key(sessionID string) string {
    return rs.keyPrefix + sessionID
}

func (rs *redisSessionStore) Create(ctx *fiber.Ctx, data *SessionData) error {
    key := rs.key(data.SessionID)
    jsonData, _ := json.Marshal(data)
    
    return rs.client.Set(ctx.Context(), key, jsonData, 7*24*time.Hour).Err()
}

func (rs *redisSessionStore) Get(ctx *fiber.Ctx, sessionID string) (*SessionData, error) {
    key := rs.key(sessionID)
    result, err := rs.client.Get(ctx.Context(), key).Bytes()
    
    if err == redis.Nil {
        return nil, ErrSessionNotFound
    }
    if err != nil {
        return nil, err
    }
    
    var data SessionData
    if err := json.Unmarshal(result, &data); err != nil {
        return nil, err
    }
    
    return &data, nil
}

func (rs *redisSessionStore) Update(ctx *fiber.Ctx, data *SessionData) error {
    key := rs.key(data.SessionID)
    jsonData, _ := json.Marshal(data)
    
    return rs.client.Set(ctx.Context(), key, jsonData, 7*24*time.Hour).Err()
}

func (rs *redisSessionStore) Delete(ctx *fiber.Ctx, sessionID string) error {
    key := rs.key(sessionID)
    return rs.client.Del(ctx.Context(), key).Err()
}
```

### 4.10 JWT Token Methods

```go
package auth

func (sm *sessionManager) GenerateAccessToken(userID, sessionID string, isOAuthUser, trialActive bool) string {
    claims := &AccessTokenClaims{
        UserID:      userID,
        SessionID:   sessionID,
        IsOAuthUser: isOAuthUser,
        TrialActive: trialActive,
        RegisteredClaims: jwt.RegisteredClaims{
            Issuer:    "mealswapp",
            Subject:   userID,
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
            ID:        sessionID,
        },
    }
    
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    signedToken, _ := token.SignedString(sm.jwtSecret)
    return signedToken
}

func (sm *sessionManager) ParseAccessToken(tokenString string) (*AccessTokenClaims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &AccessTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return sm.jwtSecret, nil
    })
    
    if err != nil {
        return nil, err
    }
    
    if claims, ok := token.Claims.(*AccessTokenClaims); ok && token.Valid {
        return claims, nil
    }
    
    return nil, ErrAccessTokenInvalid
}

func (sm *sessionManager) GenerateRefreshToken(userID, sessionID string) string {
    claims := &RefreshTokenClaims{
        UserID:       userID,
        SessionID:    sessionID,
        TokenVersion: atomic.AddInt32(&globalTokenVersion, 1),
        RegisteredClaims: jwt.RegisteredClaims{
            Issuer:    "mealswapp",
            Subject:   userID,
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
            ID:        GenerateUUID(),
        },
    }
    
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    signedToken, _ := token.SignedString(sm.jwtSecret)
    return signedToken
}

func (sm *sessionManager) ParseRefreshToken(tokenString string) (*RefreshTokenClaims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &RefreshTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return sm.jwtSecret, nil
    })
    
    if err != nil {
        return nil, err
    }
    
    if claims, ok := token.Claims.(*RefreshTokenClaims); ok && token.Valid {
        return claims, nil
    }
    
    return nil, ErrRefreshTokenInvalid
}
```
