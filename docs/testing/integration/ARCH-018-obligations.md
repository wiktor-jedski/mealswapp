# ARCH-018 Integration Verification Obligations

## Purpose

This document defines the SWE.5 integration verification obligations for architecture component ARCH-018, the Frontend Authentication Session Module.

The goal is to verify that AuthView, RegisterView, LoginView, AuthSessionStore, AuthApiClient, ConsentGate, OAuthEntryPoint, AuthenticatedActionGuard, SearchView, SidebarComponent, SubscriptionBilling, generated API clients, and safe error mapping collaborate according to the architecture.

## Component Information

| Field | Value |
| --- | --- |
| Architecture Component | ARCH-018 |
| Name | Frontend Authentication Session Module |
| Source Documents | `docs/architecture/ARCH-018.md`, `docs/design/DESIGN-018.md`, `docs/design/DESIGN-001.md`, `docs/design/DESIGN-006.md`, `docs/design/DESIGN-007.md`, `docs/design/DESIGN-010.md`, `docs/design/DESIGN-015.md`, `docs/design/DESIGN-017.md` |
| Related Units | AuthView, RegisterView, LoginView, AuthSessionStore, AuthApiClient, ConsentGate, OAuthEntryPoint, AuthenticatedActionGuard, SearchView, SidebarComponent, SubscriptionBilling, generated API clients |
| Collaborating Architecture | ARCH-001, ARCH-006, ARCH-007, ARCH-010, ARCH-015, ARCH-017 |
| Related Requirements | SW-REQ-044, SW-REQ-046, SW-REQ-058, SW-REQ-060, SW-REQ-061, SW-REQ-062, SW-REQ-063, SW-REQ-064, SW-REQ-065, SW-REQ-066, SW-REQ-070, SW-REQ-074 |

## IT-ARCH-018-001 Registration, Login, Logout, and Anonymous Search Fallback

### Intent

Verify that the composed frontend auth session module supports email registration, email login, logout, and anonymous Catalog Search fallback while preserving SearchView state and using generated API contracts.

### System Under Test

ARCH-018 Frontend Authentication Session Module, centered on AuthView, RegisterView, LoginView, AuthSessionStore, and AuthApiClient.

### Real Components

- AuthView
- RegisterView
- LoginView
- AuthSessionStore
- AuthApiClient
- SearchView
- SidebarComponent
- generated frontend API types

### Allowed Test Doubles

- Playwright route interception may stand in for ARCH-006, ARCH-010, ARCH-015 consent, and ARCH-007 HTTP responses while preserving generated frontend response types.

### Trigger / Stimulus

Anonymous user opens guarded subscription UI, registers, logs out, performs anonymous Catalog Search, then logs in.

### Expected Integrated Behavior

1. Anonymous subscription access renders sign-in/register guidance and does not call checkout.
2. Registration uses CSRF and generated-contract payloads with consent versions before setting authenticated session projection.
3. Logout clears frontend-safe session state and preserves anonymous Catalog Search behavior.
4. Anonymous Catalog Search remains usable after logout.
5. Login uses CSRF and generated-contract payloads, refreshes session and entitlement state, and restores authenticated UI.

### Required Evidence

- Test verifies registration, logout, anonymous search, login, auth state display, preserved search state, and no checkout call while anonymous.
- Test traceability comment references `IT-ARCH-018-001`, `ARCH-018`, `DESIGN-018`, and related SW requirements.

### Requirement Traceability

- SW-REQ-058
- SW-REQ-060
- SW-REQ-061
- SW-REQ-064
- SW-REQ-070

### Verification Status

Implemented by:

- `frontend/tests/auth-session.spec.ts::registration, login, logout, anonymous search fallback, sidebar navigation, keyboard flow, and axe checks work together`
- `frontend/src/lib/stores/auth-session.test.ts::loginWithEmail and registerWithEmail use CSRF, store authenticated state, and refresh entitlements`
- `frontend/src/lib/stores/auth-session.test.ts::logout clears authenticated state while preserving anonymous Catalog Search state`
- `frontend/src/lib/api/auth-client.test.ts::registerWithEmail uses generated DTOs, CSRF header, credentialed POST, and clears caller password`
- `frontend/src/lib/api/auth-client.test.ts::loginWithEmail uses generated LoginRequest, maps session envelope, and clears caller password`

Status: PASS.

## IT-ARCH-018-002 Authenticated Search and Subscription Navigation

### Intent

Verify that ARCH-018 session projection collaborates with ARCH-001 SearchView and SidebarComponent so authenticated users can move between Search and Subscription views without losing search state.

### System Under Test

ARCH-018 Frontend Authentication Session Module, centered on AuthSessionStore and AuthenticatedActionGuard as consumed by SidebarComponent.

### Real Components

- AuthSessionStore
- SidebarComponent
- SearchView
- SubscriptionBilling
- entitlement store
- generated API types

### Allowed Test Doubles

- Playwright route interception may stand in for backend HTTP responses while preserving generated frontend response types.

### Trigger / Stimulus

Authenticated user performs a Catalog Search, opens Subscription navigation, then returns to Search with pointer and keyboard flows.

### Expected Integrated Behavior

1. Authenticated session projection exposes account navigation.
2. Subscription navigation opens billing UI and hides Search-specific controls within the subscription surface.
3. Search navigation returns to SearchView with prior query and result state intact.
4. Keyboard focus reaches Search and Subscription account navigation controls.
5. Mobile sidebar closes predictably after subscription selection.

### Required Evidence

- Test verifies authenticated Search/Subscription navigation, keyboard activation, mobile behavior, preserved query, and preserved result cards.
- Test traceability comment references `IT-ARCH-018-002`, `ARCH-018`, `DESIGN-018`, `ARCH-001`, and related SW requirements.

### Requirement Traceability

- SW-REQ-044
- SW-REQ-058
- SW-REQ-061

### Verification Status

Implemented by:

- `frontend/tests/subscription-navigation.spec.ts::authenticated sidebar links separate Subscription from Search and preserve search state`
- `frontend/tests/subscription-navigation.spec.ts::keyboard focus reaches account links and Enter activation preserves search state`
- `frontend/tests/subscription-navigation.spec.ts::mobile sidebar navigation remains usable and closes after Subscription selection`
- `frontend/tests/auth-session.spec.ts::registration, login, logout, anonymous search fallback, sidebar navigation, keyboard flow, and axe checks work together`

Status: PASS.

## IT-ARCH-018-003 Protected Checkout Gating and Retry

### Intent

Verify that AuthenticatedActionGuard blocks protected checkout and entitlement actions for anonymous, unknown, expired, and unverified states, then allows authenticated users to reach subscription UI and retry protected checkout from a cookie-backed session.

### System Under Test

ARCH-018 Frontend Authentication Session Module, centered on AuthenticatedActionGuard.

### Real Components

- AuthenticatedActionGuard
- AuthSessionStore
- LoginView
- RegisterView
- SubscriptionBilling
- entitlement store
- generated billing client types

### Allowed Test Doubles

- Playwright route interception may stand in for ARCH-006, ARCH-007, and ARCH-010 responses while preserving generated frontend response types.

### Trigger / Stimulus

Users with unknown, anonymous, expired, unverified, and authenticated session states attempt subscription and checkout actions.

### Expected Integrated Behavior

1. Unknown and anonymous sessions do not automatically call protected entitlement or sidebar endpoints.
2. Anonymous subscription access renders auth guidance instead of checkout.
3. Expired sessions clear frontend-safe auth state before protected navigation.
4. Unverified sessions do not load protected sidebar activity.
5. Successful authentication opens the guarded subscription UI and checkout calls happen only after an authenticated user action.
6. Canceling auth clears queued subscription navigation.

### Required Evidence

- Test verifies no protected network calls before auth, guarded subscription UI, queued action retry behavior, checkout call count, and cancellation behavior.
- Test traceability comment references `IT-ARCH-018-003`, `ARCH-018`, `DESIGN-018`, `ARCH-007`, and related SW requirements.

### Requirement Traceability

- SW-REQ-044
- SW-REQ-058
- SW-REQ-061
- SW-REQ-066

### Verification Status

Implemented by:

- `frontend/tests/auth-guard.spec.ts::unknown sessions do not automatically call protected entitlement refresh`
- `frontend/tests/auth-guard.spec.ts::unknown sessions do not load protected sidebar activity`
- `frontend/tests/auth-guard.spec.ts::anonymous sessions do not automatically call protected entitlement refresh`
- `frontend/tests/auth-guard.spec.ts::anonymous sessions do not load protected sidebar activity`
- `frontend/tests/auth-guard.spec.ts::authenticated but unverified sessions do not load protected sidebar activity`
- `frontend/tests/auth-guard.spec.ts::verified authenticated sessions load protected sidebar activity`
- `frontend/tests/auth-guard.spec.ts::anonymous Catalog Search stays usable while Subscription navigation is guarded`
- `frontend/tests/auth-guard.spec.ts::successful registration retries queued Subscription navigation with the cookie-backed session`
- `frontend/tests/auth-guard.spec.ts::canceling auth clears queued Subscription navigation`
- `frontend/tests/login.spec.ts::successful login opens guarded Subscription view, creates checkout once, and preserves search state after closing`

Status: PASS.

## IT-ARCH-018-004 OAuth Return Refresh

### Intent

Verify that OAuth entry and return handling do not infer authentication from URL parameters, and instead refresh session and entitlement state through generated-contract backend calls.

### System Under Test

ARCH-018 Frontend Authentication Session Module, centered on OAuthEntryPoint, AuthApiClient, and AuthSessionStore.

### Real Components

- OAuthEntryPoint
- AuthApiClient
- AuthSessionStore
- entitlement store
- generated API types

### Allowed Test Doubles

- Unit-level dependency injection may stand in for backend refresh and entitlement responses while preserving generated frontend data structures.
- Playwright route interception may stand in for profile, session-refresh, and entitlement responses while rendering the real modal OAuth entry UI.

### Trigger / Stimulus

User opens auth surface with OAuth provider actions and the SPA processes OAuth-return URLs.

### Expected Integrated Behavior

1. Google and Apple OAuth entry actions use generated provider start URLs without provider secrets.
2. OAuth-return refresh calls the backend session refresh endpoint exactly once.
3. Success URL parameters alone do not mark the user authenticated.
4. Successful refresh stores frontend-safe session projection and refreshes entitlement state.
5. Expired OAuth-return refresh maps to expired session state.

### Required Evidence

- Tests verify OAuth provider actions, generated start URLs, server refresh call ordering, entitlement refresh, and expired-return behavior.
- Test traceability comment references `IT-ARCH-018-004`, `ARCH-018`, `DESIGN-018`, and related SW requirements.

### Requirement Traceability

- SW-REQ-046
- SW-REQ-058
- SW-REQ-061
- SW-REQ-070

### Verification Status

Implemented by:

- `frontend/tests/auth-surface.spec.ts::SearchShell modal is the sole auth surface and exposes Google sign-in`
- `frontend/tests/auth-surface.spec.ts::OAuth callback keeps the SearchShell mounted and refreshes the modal session`
- `frontend/src/lib/stores/auth-session.test.ts::OAuth-return refresh ignores URL parameters and trusts only server session refresh`
- `frontend/src/lib/stores/auth-session.test.ts::OAuth-return refresh stores authenticated projection and coordinates entitlement refresh`
- `frontend/src/lib/api/auth-client.test.ts::getOAuthStartUrl returns generated provider start URLs without provider secrets`
- `frontend/src/lib/api/auth-client.test.ts::refreshAuthStateAfterOAuthReturn coordinates session then entitlement refresh`

Status: PASS.

## IT-ARCH-018-005 Consent and Safe Failure Handling

### Intent

Verify that ConsentGate collaborates with ARCH-015 and ARCH-017 so registration is blocked until current consent is accepted, stale consent is recoverable, and auth errors remain safe.

### System Under Test

ARCH-018 Frontend Authentication Session Module, centered on ConsentGate, RegisterView, LoginView, and AuthApiClient error mapping.

### Real Components

- ConsentGate
- RegisterView
- LoginView
- AuthApiClient
- ErrorMessageMapper-facing auth error mapping

### Allowed Test Doubles

- Playwright route interception may stand in for ARCH-015 and ARCH-006 responses while preserving generated frontend response types.

### Trigger / Stimulus

User opens the modal auth surface, attempts registration without consent, receives stale consent, and receives invalid credential or rate-limit responses.

### Expected Integrated Behavior

1. Registration remains disabled until Privacy Policy and Terms of Service versions are accepted.
2. Stale consent clears acceptance and requires re-acceptance.
3. Invalid credentials use generic safe copy and do not enumerate accounts.
4. Rate-limit metadata is normalized and displayed only when safe.

### Required Evidence

- Tests verify consent blocking, stale consent recovery, generic login feedback, lockout retry timing, and password clearing.
- Test traceability comment references `IT-ARCH-018-005`, `ARCH-018`, `DESIGN-018`, `ARCH-015`, `ARCH-017`, and related SW requirements.

### Requirement Traceability

- SW-REQ-060
- SW-REQ-062
- SW-REQ-063
- SW-REQ-065
- SW-REQ-074

### Verification Status

Implemented by:

- `frontend/tests/register.spec.ts::registration cannot submit until current consent versions are checked`
- `frontend/tests/register.spec.ts::stale consent clears acceptance and requires re-acceptance`
- `frontend/tests/login.spec.ts::login form validates focus order, generic invalid credentials, lockout retry timing, duplicate submissions, and password clearing`
- `frontend/tests/login.spec.ts::login retry timing suppresses malformed values, clamps huge values, and supports HTTP-date metadata`
- `frontend/src/lib/api/auth-client.test.ts::auth client maps 400, 401, 403, 409, 429, and 503 envelopes to safe AppError values`

Status: PASS.

## IT-ARCH-018-006 Session Expiry Recovery

### Intent

Verify that expired or invalid server sessions are mapped through ARCH-017 to expired/anonymous frontend-safe state, protected actions request sign-in, and anonymous Catalog Search remains usable.

### System Under Test

ARCH-018 Frontend Authentication Session Module, centered on AuthSessionStore and AuthenticatedActionGuard.

### Real Components

- AuthSessionStore
- AuthenticatedActionGuard
- SearchView
- SubscriptionBilling
- AuthApiClient error mapping

### Allowed Test Doubles

- Unit-level dependency injection and Playwright route interception may stand in for backend responses while preserving generated frontend error envelopes.

### Trigger / Stimulus

Server profile/session probe returns anonymous, invalid, expired, locked, or unexpected failure semantics.

### Expected Integrated Behavior

1. Profile 401 anonymous/invalid responses map to anonymous session state.
2. Session-expired responses map to expired state and remove user fields from storage.
3. Lockout responses map to locked state.
4. Unexpected failures map to error state without exposing secrets.
5. Expired protected navigation displays sign-in guidance.

### Required Evidence

- Tests verify state transitions, storage clearing, protected navigation guidance, and Catalog Search fallback.
- Test traceability comment references `IT-ARCH-018-006`, `ARCH-018`, `DESIGN-018`, `ARCH-017`, and related SW requirements.

### Requirement Traceability

- SW-REQ-058
- SW-REQ-061
- SW-REQ-065
- SW-REQ-066

### Verification Status

Implemented by:

- `frontend/src/lib/stores/auth-session.test.ts::probeAuthSession maps 401 to anonymous and session-expired semantics to expired`
- `frontend/src/lib/stores/auth-session.test.ts::probeAuthSession maps lockout and unexpected failures to locked and error states`
- `frontend/src/lib/stores/auth-session.test.ts::clearAuthSession removes user fields for anonymous and expired transitions`
- `frontend/tests/auth-guard.spec.ts::expired sessions clear frontend-safe auth state before guarded Subscription navigation`

Status: PASS.

## IT-ARCH-018-007 No Token and No Card Data in Frontend State

### Intent

Verify that ARCH-018 keeps access tokens, refresh tokens, raw CSRF secrets, passwords, provider secrets, and raw payment-card data out of JavaScript-visible persistent frontend state and UI.

### System Under Test

ARCH-018 Frontend Authentication Session Module, centered on AuthSessionStore, AuthApiClient, RegisterView, LoginView, and AuthenticatedActionGuard.

### Real Components

- AuthSessionStore
- AuthApiClient
- RegisterView
- LoginView
- SubscriptionBilling
- generated API types

### Allowed Test Doubles

- Unit-level dependency injection may inject oversized response objects to verify stripping.
- Playwright route interception may stand in for backend and subscription responses while rendering real UI.

### Trigger / Stimulus

Generated frontend clients receive token-like response fields, login/register failures occur, OAuth start URLs are built, and anonymous users open guarded subscription UI.

### Expected Integrated Behavior

1. Session/profile projection strips token-like fields before storage.
2. Password request fields are cleared after successful and failed auth mutations.
3. Browser storage does not contain passwords, emails from duplicate registration failures, raw CSRF tokens, access tokens, or refresh tokens.
4. OAuth start URLs do not expose provider secrets.
5. Subscription UI never renders raw card, PAN, CVC, CVV, or credit-card autocomplete inputs.
6. Anonymous checkout attempts do not call protected checkout endpoints.

### Required Evidence

- Tests verify storage snapshots, stripped projections, cleared request fields, OAuth URL safety, no card inputs, and checkout call suppression.
- Test traceability comment references `IT-ARCH-018-007`, `ARCH-018`, `DESIGN-018`, `ARCH-007`, `ARCH-010`, and related SW requirements.

### Requirement Traceability

- SW-REQ-044
- SW-REQ-058
- SW-REQ-061
- SW-REQ-064
- SW-REQ-066

### Verification Status

Implemented by:

- `frontend/src/lib/stores/auth-session.test.ts::probeAuthSession stores only frontend-safe projection fields when authenticated`
- `frontend/src/lib/stores/auth-session.test.ts::storage failures keep cookie-based auth usable without persisting tokens or passwords`
- `frontend/src/lib/api/auth-client.test.ts::session and profile decoding strip unexpected token strings from JavaScript-visible results`
- `frontend/src/lib/api/auth-client.test.ts::auth mutation helpers clear passwords after failed submissions too`
- `frontend/tests/register.spec.ts::duplicate email offers login mode without storing PII or passwords`
- `frontend/tests/register.spec.ts::successful registration creates an authenticated session projection`
- `frontend/tests/auth-guard.spec.ts::anonymous Catalog Search stays usable while Subscription navigation is guarded`

Status: PASS.
