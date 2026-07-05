## FILE: DESIGN-018.md
**Traceability:** ARCH-018

**Static aspects covered:** AuthView, RegisterView, LoginView, AuthSessionStore, AuthApiClient, ConsentGate, DisclaimerPanel, OAuthEntryPoint, AuthenticatedActionGuard.

### 0. Static Aspect Responsibilities
- `AuthView`: owns the sign-in/register surface, mode switching, authenticated summary, and handoff back to the invoking workflow.
- `RegisterView`: owns email/password registration inputs, password-policy feedback, consent-gated submission, and duplicate-email feedback.
- `LoginView`: owns email/password login inputs, generic invalid-credential feedback, lockout/rate-limit feedback, and successful-session handoff.
- `AuthSessionStore`: owns frontend-safe authenticated/anonymous/session-expired state derived from server responses; it never stores access tokens, refresh tokens, password text, or raw CSRF secrets beyond the current request boundary.
- `AuthApiClient`: owns generated-contract calls for CSRF retrieval, registration, login, logout, refresh/session recovery, profile/session probing, disclaimer retrieval, and OAuth start URLs.
- `ConsentGate`: owns Privacy Policy and ToS version selection state and blocks registration submission until current versions are explicitly accepted.
- `DisclaimerPanel`: owns login-screen medical disclaimer loading, fallback rendering, and unavailable-state feedback.
- `OAuthEntryPoint`: owns Google and Apple login entry actions and callback-return state refresh.
- `AuthenticatedActionGuard`: owns anonymous/authenticated decisioning for protected UI actions such as Stripe-hosted Checkout and authenticated entitlement refresh.

### 1. Data Structures & Types
- `type AuthMode = "login" | "register"`
- `type AuthStatus = "unknown" | "anonymous" | "authenticating" | "authenticated" | "expired" | "locked" | "error"`
- `type OAuthProvider = "google" | "apple"`
- `type ProtectedActionKind = "checkout" | "entitlement_refresh" | "profile" | "saved_data" | "account"`
- `interface AuthSessionProjection { status: AuthStatus; userId?: string; email?: string; displayName?: string; hasVerifiedLoginMethod?: boolean; role?: "user" | "admin"; lastCheckedAt?: string; error?: AppError }`
- `interface LoginFormState { email: string; password: string; submitting: boolean; error?: AppError; retryAfterSeconds?: number }`
- `interface RegisterFormState { email: string; password: string; confirmPassword: string; privacyPolicyVersion?: string; termsVersion?: string; privacyAccepted: boolean; termsAccepted: boolean; submitting: boolean; error?: AppError }`
- `interface ConsentVersions { privacyPolicyVersion: string; termsVersion: string; effectiveAt: string }`
- `interface DisclaimerViewModel { version: string; bodyMarkdown: string; effectiveAt: string; unavailable: boolean }`
- `interface AuthRequestContext { csrfToken: string; requestId?: string }`
- `interface AuthenticatedActionRequest { kind: ProtectedActionKind; label: string; continueAfterAuth: () => Promise<void> }`
- `interface AuthGuardDecision { allowed: boolean; reason?: "anonymous" | "expired" | "unverified" | "locked"; signInAction?: AuthenticatedActionRequest }`

### 2. Logic & Algorithms (Step-by-Step)
1. On SPA startup, initialize `AuthSessionStore` to `unknown`, then run a generated-contract session/profile probe with `credentials: "include"`.
2. If the probe returns authenticated profile/session data, store only `AuthSessionProjection`; if it returns 401, set `anonymous`; if it returns session-expired semantics, set `expired`.
3. Load login-screen disclaimer content through `AuthApiClient.getDisclaimer("login")`; if unavailable, render bundled fallback content and mark `DisclaimerViewModel.unavailable = true`.
4. When `AuthView` opens, choose `login` mode unless the invoking action explicitly requests registration. Keep the current search state intact while the auth surface is visible.
5. On registration submission, validate local form completeness, password confirmation, current consent checkbox acceptance, and consent version presence before requesting CSRF.
6. For registration, fetch a fresh CSRF token, submit generated registration payload with consent versions and `credentials: "include"`, map duplicate-email, stale-consent, validation, lockout, and network failures through ARCH-017, then refresh session/profile and entitlement state on success.
7. On login submission, validate local form completeness before requesting CSRF.
8. For login, fetch a fresh CSRF token, submit generated login payload with `credentials: "include"`, map invalid credentials to a generic user-facing message, surface lockout/rate-limit retry timing when the server provides safe metadata, then refresh session/profile and entitlement state on success.
9. For OAuth login, navigate to the generated provider start URL. On return to the SPA, run session/profile and entitlement refresh; do not infer success from URL parameters alone.
10. For logout, fetch a fresh CSRF token, submit logout with `credentials: "include"`, clear frontend-safe session state to `anonymous`, and preserve anonymous Catalog Search state.
11. `AuthenticatedActionGuard` checks `AuthSessionProjection` before protected actions. Anonymous or expired users see sign-in/register guidance and the pending action is retained as a retry callback.
12. After successful authentication, run the retained protected-action callback exactly once and clear it, so checkout can be retried from a real HttpOnly-cookie browser session.
13. Registration through email/password may create an unverified login method. The UI must display verified-login restriction feedback from server state and must not locally override paid-feature eligibility.
14. Never store or expose access tokens, refresh tokens, raw password values after submission, provider secrets, Stripe customer IDs, or payment-card fields in frontend state.

### 3. State Management & Error Handling
- `unknown`: session probe has not completed; protected actions wait or show neutral loading state.
- `anonymous`: no authenticated server session; Catalog Search remains usable and protected actions request sign-in/register.
- `authenticating`: login, registration, logout, or OAuth-return session refresh is in progress; duplicate submissions are disabled.
- `authenticated`: server-derived session projection exists; entitlement and checkout actions may proceed through their own authorization checks.
- `expired`: server reports expired or invalid cookies; clear frontend-safe session projection and request sign-in before protected actions.
- `locked`: server reports account or IP lockout; show safe retry timing if present and keep password field cleared.
- `invalid_credentials`: show generic login failure text without revealing whether the email exists.
- `duplicate_email`: registration shows duplicate-account feedback and offers login mode.
- `consent_missing`: registration submit remains disabled until current Privacy Policy and ToS versions are accepted.
- `consent_stale`: registration refreshes current consent versions and asks the user to accept the latest versions.
- `disclaimer_unavailable`: display bundled fallback disclaimer and continue to allow login/register.
- `csrf_unavailable`: block auth mutation, show retryable network/security error, and do not send credentials.
- `oauth_unavailable`: provider entry action is disabled or maps to fail-closed feedback when backend provider configuration is unavailable.
- `protected_action_pending`: store one pending protected action and clear it after successful retry or explicit cancellation.
- `session_storage_forbidden`: if localStorage/sessionStorage is unavailable, keep auth state in memory only; do not degrade cookie-based auth.

### 4. Component Interfaces
- `function createInitialAuthSession(): AuthSessionProjection`
- `async function probeAuthSession(): Promise<AuthSessionProjection>`
- `function setAuthSession(session: AuthSessionProjection): void`
- `function clearAuthSession(reason: "logout" | "expired" | "anonymous"): void`
- `async function fetchCsrfToken(): Promise<AuthRequestContext>`
- `async function registerWithEmail(state: RegisterFormState): Promise<AuthSessionProjection>`
- `async function loginWithEmail(state: LoginFormState): Promise<AuthSessionProjection>`
- `async function logoutCurrentSession(): Promise<void>`
- `async function refreshAuthSessionAfterOAuthReturn(): Promise<AuthSessionProjection>`
- `function getOAuthStartUrl(provider: OAuthProvider): string`
- `async function loadLoginDisclaimer(): Promise<DisclaimerViewModel>`
- `function canSubmitRegistration(state: RegisterFormState, consent: ConsentVersions): boolean`
- `function canSubmitLogin(state: LoginFormState): boolean`
- `function buildAuthGuardDecision(session: AuthSessionProjection, action: AuthenticatedActionRequest): AuthGuardDecision`
- `function queueProtectedAction(action: AuthenticatedActionRequest): void`
- `async function runQueuedProtectedActionAfterAuth(): Promise<void>`
- `function mapAuthError(error: AppError): AuthStatus`
