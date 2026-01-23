# JWTManager

**Traceability:** ARCH-006

## 1. Data Structures & Types

```go
package jwt

import (
	"time"
	"github.com/golang-jwt/jwt/v5"
)

type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
)

type TokenClaims struct {
	jwt.RegisteredClaims
	UserID      string    `json:"user_id"`
	Email       string    `json:"email"`
	TokenType   TokenType `json:"token_type"`
	SessionID   string    `json:"session_id"`
	Permissions []string  `json:"permissions,omitempty"`
}

type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
}

type TokenPayload struct {
	UserID      string
	Email       string
	SessionID   string
	Permissions []string
}

type JWTManagerConfig struct {
	AccessTokenSecret  string
	RefreshTokenSecret string
	AccessTokenExpiry  time.Duration
	RefreshTokenExpiry time.Duration
	Issuer             string
}

type TokenValidator struct {
	accessSecret  string
	refreshSecret string
	issuer        string
}
```

## 2. Logic & Algorithms

### 2.1 Generate Token Pair

```
FUNCTION GenerateTokenPair(payload TokenPayload) -> TokenPair

	accessClaims := TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(currentTime.Add(config.AccessTokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(currentTime),
			NotBefore: jwt.NewNumericDate(currentTime),
			Issuer:    config.Issuer,
			Subject:   payload.UserID,
		},
		UserID:      payload.UserID,
		Email:       payload.Email,
		TokenType:   TokenTypeAccess,
		SessionID:   payload.SessionID,
		Permissions: payload.Permissions,
	}

	refreshClaims := TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(currentTime.Add(config.RefreshTokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(currentTime),
			NotBefore: jwt.NewNumericDate(currentTime),
			Issuer:    config.Issuer,
			Subject:   payload.UserID,
		},
		UserID:      payload.UserID,
		Email:       payload.Email,
		TokenType:   TokenTypeRefresh,
		SessionID:   payload.SessionID,
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)

	accessSigned, accessError := accessToken.SignedString([]byte(config.AccessTokenSecret))
	refreshSigned, refreshError := refreshToken.SignedString([]byte(config.RefreshTokenSecret))

	IF accessError OR refreshError THEN
		RETURN error
	END IF

	RETURN TokenPair{
		AccessToken:  accessSigned,
		RefreshToken: refreshSigned,
		ExpiresAt:    currentTime.Add(config.AccessTokenExpiry),
		TokenType:    "Bearer",
	}
END FUNCTION
```

### 2.2 Validate Access Token

```
FUNCTION ValidateAccessToken(tokenString string) -> TokenClaims, error

	token, error := jwt.ParseWithClaims(tokenString, &TokenClaims{}, 
		FUNCTION(token *jwt.Token) -> interface{}
			IF method, ok := token.Method.(*jwt.SigningMethodHMAC); ok THEN
				RETURN method.VerifyKey([]byte(config.AccessSecret))
			END IF
			RETURN error: "unexpected signing method"
		END FUNCTION
	)

	IF error != nil THEN
		RETURN nil, error
	END IF

	claims, ok := token.Claims.(*TokenClaims)
	IF NOT ok OR NOT token.Valid THEN
		RETURN nil, error: "invalid token claims"
	END IF

	IF claims.TokenType != TokenTypeAccess THEN
		RETURN nil, error: "invalid token type"
	END IF

	RETURN claims, nil
END FUNCTION
```

### 2.3 Validate Refresh Token

```
FUNCTION ValidateRefreshToken(tokenString string) -> TokenClaims, error

	token, error := jwt.ParseWithClaims(tokenString, &TokenClaims{},
		FUNCTION(token *jwt.Token) -> interface{}
			IF method, ok := token.Method.(*jwt.SigningMethodHMAC); ok THEN
				RETURN method.VerifyKey([]byte(config.RefreshSecret))
			END IF
			RETURN error: "unexpected signing method"
		END FUNCTION
	)

	IF error != nil THEN
		RETURN nil, error
	END IF

	claims, ok := token.Claims.(*TokenClaims)
	IF NOT ok OR NOT token.Valid THEN
		RETURN nil, error: "invalid token claims"
	END IF

	IF claims.TokenType != TokenTypeRefresh THEN
		RETURN nil, error: "invalid token type"
	END IF

	RETURN claims, nil
END FUNCTION
```

### 2.4 Refresh Token Rotation

```
FUNCTION RotateRefreshToken(currentRefreshToken string, payload TokenPayload) -> TokenPair, error

	claims, error := ValidateRefreshToken(currentRefreshToken)
	IF error != nil THEN
		RETURN nil, error
	END IF

	IF claims.ExpiresAt.Before(currentTime) THEN
		RETURN nil, error: "refresh token expired"
	END IF

	newPayload := TokenPayload{
		UserID:      claims.UserID,
		Email:       claims.Email,
		SessionID:   claims.SessionID,
		Permissions: claims.Permissions,
	}

	RETURN GenerateTokenPair(newPayload)
END FUNCTION
```

### 2.5 Extract Token from Cookie

```
FUNCTION ExtractTokenFromCookie(cookie *http.Cookie, expectedType TokenType) -> string, error

	IF cookie == nil THEN
		RETURN "", error: "cookie not found"
	END IF

	IF cookie.HttpOnly != true THEN
		RETURN "", error: "cookie must be HttpOnly"
	END IF

	IF cookie.Secure != true THEN
		RETURN "", error: "cookie must be Secure"
	END IF

	IF cookie.SameSite != http.SameSiteStrictMode THEN
		RETURN "", error: "cookie must use SameSite=Strict"
	END IF

	RETURN cookie.Value, nil
END FUNCTION
```

### 2.6 Blacklist Token (for logout)

```
FUNCTION BlacklistToken(tokenString string, expiry time.Duration) -> error

	claims, _, error := jwt.NewParser().ParseUnverified(tokenString, &TokenClaims{})
	IF error != nil THEN
		RETURN error
	END IF

	jti := uuid.New().String()
	blacklistKey := "blacklist:" + jti

	store.Set(ctx, blacklistKey, "revoked", expiry)

	RETURN nil
END FUNCTION
```

### 2.7 Check Token Blacklist

```
FUNCTION IsTokenBlacklisted(tokenString string) -> bool, error

	claims, _, error := jwt.NewParser().ParseUnverified(tokenString, &TokenClaims{})
	IF error != nil THEN
		RETURN false, error
	END IF

	jti, exists := claims.RegisteredClaims[jwt.RegisteredClaims{}.ID]
	IF NOT exists THEN
		RETURN false, nil
	END IF

	blacklistKey := "blacklist:" + jti
	exists, err := store.Exists(ctx, blacklistKey)

	RETURN exists, err
END FUNCTION
```

## 3. State Management & Error Handling

### 3.1 Error States

| Error | Condition | Recovery Action |
|-------|-----------|-----------------|
| `ErrInvalidToken` | Token is malformed or signature verification fails | Request new token pair via login |
| `ErrExpiredToken` | Token has passed its expiration time | Request new token pair via login or refresh |
| `ErrInvalidTokenType` | Token type mismatch (using refresh token for access) | Use correct token type |
| `ErrTokenBlacklisted` | Token has been explicitly revoked (logout) | Force re-authentication |
| `ErrMissingToken` | No token provided in request | Redirect to login |
| `ErrInvalidCookie` | Cookie fails security constraints (HttpOnly, Secure, SameSite) | Redirect to login |
| `ErrInvalidClaims` | Token claims are missing required fields | Request new token pair |

### 3.2 State Transitions

```
IDLE -> Validating
    |- valid access token -> Authenticated
    |- invalid/expired access token -> check refresh token
    |- no access token -> check refresh token

Validating Refresh
    |- valid refresh token -> Generate new token pair
    |- invalid/expired refresh token -> Unauthenticated

Authenticated -> Token Refresh
    |- access token near expiration -> Rotate tokens

Authenticated -> Logout
    |- user logs out -> Blacklist tokens -> Unauthenticated
```

### 3.3 Token Lifecycle States

| State | Description | Next State |
|-------|-------------|------------|
| `Created` | Token pair just generated | `Active` |
| `Active` | Valid tokens in use | `RefreshNeeded`, `Revoked` |
| `RefreshNeeded` | Access token expired but refresh valid | `Active` |
| `Revoked` | Token explicitly invalidated (logout) | `Expired` |
| `Expired` | Token past expiration time | N/A |

## 4. Component Interfaces

### 4.1 Public Methods

```go
type JWTManager interface {
	// NewTokenPair creates a new access and refresh token pair
	NewTokenPair(ctx context.Context, payload TokenPayload) (*TokenPair, error)

	// ValidateAccessToken validates and parses an access token
	ValidateAccessToken(ctx context.Context, tokenString string) (*TokenClaims, error)

	// ValidateRefreshToken validates and parses a refresh token
	ValidateRefreshToken(ctx context.Context, tokenString string) (*TokenClaims, error)

	// RotateTokens invalidates the old refresh token and issues a new pair
	RotateTokens(ctx context.Context, currentRefreshToken string) (*TokenPair, error)

	// ExtractTokenFromRequest extracts token from Authorization header or cookie
	ExtractTokenFromRequest(ctx context.Context, r *http.Request, expectedType TokenType) (string, error)

	// BlacklistToken adds a token to the blacklist
	BlacklistToken(ctx context.Context, tokenString string) error

	// IsBlacklisted checks if a token is blacklisted
	IsBlacklisted(ctx context.Context, tokenString string) (bool, error)
}
```

### 4.2 Configuration Interface

```go
type JWTConfigProvider interface {
	GetAccessTokenSecret() string
	GetRefreshTokenSecret() string
	GetAccessTokenExpiry() time.Duration
	GetRefreshTokenExpiry() time.Duration
	GetIssuer() string
}
```

### 4.3 Storage Interface

```go
type TokenStore interface {
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Exists(ctx context.Context, key string) (bool, error)
	Delete(ctx context.Context, key string) error
}
```

### 4.4 Cookie Configuration

```go
func GetAccessCookieConfig(maxAge int, secure bool) fiber.Cookie {
	return fiber.Cookie{
		Name:     "access_token",
		Value:    "",
		Expires:  time.Now().Add(15 * time.Minute),
		HTTPOnly: true,
		Secure:   secure,
		SameSite: "Strict",
		Path:     "/",
		MaxAge:   maxAge,
	}
}

func GetRefreshCookieConfig(maxAge int, secure bool) fiber.Cookie {
	return fiber.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		HTTPOnly: true,
		Secure:   secure,
		SameSite: "Strict",
		Path:     "/api/auth/refresh",
		MaxAge:   maxAge,
	}
}
```
