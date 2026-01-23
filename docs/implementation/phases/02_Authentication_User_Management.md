## Phase 2: Authentication & User Management

**Goal:** Complete authentication system with email/password and OAuth

### Components & Static Aspects

#### ARCH-006 - Authentication Module
| Static Aspect | Description | File |
|:--------------|:------------|:-----|
| **AuthController** | HTTP handlers for auth endpoints | `auth/controller.go` |
| **PasswordHasher** | Argon2 hashing with unique salts (golang.org/x/crypto/argon2) | `auth/password_hasher.go` |
| **JWTManager** | JWT issue, validate, refresh (15min access, 7-day refresh) | `auth/jwt_manager.go` |
| **OAuthHandler** | Google/Apple OAuth via github.com/markbates/goth | `auth/oauth_handler.go` |
| **SessionManager** | Fiber session middleware integration | `auth/session_manager.go` |
| **AccountLockoutTracker** | Track failed attempts, enforce 5-failure/15min lockout | `auth/lockout_tracker.go` |

#### ARCH-010 (partial) - API Gateway
| Static Aspect | Description | File |
|:--------------|:------------|:-----|
| **RouteHandler** | Fiber route definitions and grouping | `routes/router.go` |
| **RateLimiter** | Fiber built-in limiter (10 failed/IP/10min) | `middleware/rate_limiter.go` |
| **SecurityHeaderMiddleware** | CSP, X-Frame-Options, X-Content-Type-Options, etc. | `middleware/headers.go` |
| **CSRFValidator** | Fiber csrf middleware for state-changing requests | `middleware/csrf.go` |
| **CORSHandler** | CORS configuration for allowed origins | `middleware/cors.go` |

#### ARCH-005 (partial) - Data Repository (Users)
| Static Aspect | Description | File |
|:--------------|:------------|:-----|
| **UserEntity** | User model with email, password hash, role, verification status | `models/user.go` |
| **UserRepository** | User CRUD operations | `repository/user_repo.go` |

### Testing
- [ ] User registration with email validation
- [ ] Password hashing with unique salts (verify hash format)
- [ ] JWT token lifecycle (15min access, 7-day refresh)
- [ ] OAuth flow with Google/Apple mock
- [ ] Rate limiting: 10 failed attempts/IP/10min
- [ ] Account lockout: 5 failures -> 15min lock
- [ ] Security headers present on all responses
- [ ] CSRF tokens validate correctly
- [ ] Session timeout after 30min inactivity

---

