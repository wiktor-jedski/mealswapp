# [ARCH-018] - Frontend Authentication Session Module

**Description:** Client-side authentication surface for the Svelte SPA. It presents sign-in, registration, consent, disclaimer, OAuth entry, logout, and authenticated-action gating while relying on ARCH-006 and ARCH-010 for all credential validation, token issuance, CSRF validation, and HttpOnly cookie session management.

| Attribute | Value |
| :--- | :--- |
| **Type** | Module |
| **Static Aspects** | AuthView, RegisterView, LoginView, AuthSessionStore, AuthApiClient, ConsentGate, DisclaimerPanel, OAuthEntryPoint, AuthenticatedActionGuard |
| **Dependencies** | ARCH-001 (Web Application), ARCH-006 (Authentication), ARCH-007 (Subscription), ARCH-010 (API Gateway), ARCH-015 (Compliance), ARCH-017 (Error Handling), TanStack Query |
| **Traceability** | SW-REQ-044, SW-REQ-046, SW-REQ-058, SW-REQ-060, SW-REQ-061, SW-REQ-062, SW-REQ-063, SW-REQ-064, SW-REQ-065, SW-REQ-066, SW-REQ-070, SW-REQ-071, SW-REQ-074 |

**Dynamic Behavior:**

- **Session bootstrap:** On SPA startup and after auth mutations, probes authenticated state through generated API contracts and stores only frontend-safe session projection data. Access and refresh tokens remain unavailable to JavaScript because ARCH-006 stores them in HttpOnly cookies.
- **Registration:** Collects email, password, current Privacy Policy and ToS consent, retrieves CSRF state from ARCH-010, submits registration to ARCH-006, and displays verified-login restrictions from the backend session/profile projection.
- **Login:** Collects email and password, retrieves CSRF state, submits login to ARCH-006, handles generic invalid-credential, lockout, and rate-limit errors without revealing whether an email exists, then refreshes entitlement and user-facing session state.
- **Social login:** Presents Google and Apple OAuth entry actions that navigate to ARCH-006 provider start endpoints. Callback completion is handled through the backend session cookie result and a subsequent session/entitlement refresh.
- **Compliance display:** Loads login-screen disclaimer content from ARCH-015 and blocks registration completion until explicit consent versions are selected.
- **Authenticated action gating:** When anonymous users attempt authenticated-only actions such as Stripe-hosted Checkout, renders sign-in/register guidance instead of calling protected subscription endpoints. After authentication succeeds, refreshes entitlement state and allows the user to retry the action.
- **Logout and expiry:** Calls ARCH-006 logout, clears frontend-safe session state, preserves anonymous Catalog Search behavior, and maps expired sessions to sign-in guidance through ARCH-017.

**Interface Definition:**

- `Input`: User credentials, consent selections, OAuth provider selection, CSRF token responses, auth/profile/disclaimer/entitlement API responses, browser redirects, session-cookie presence inferred by server responses
- `Output`: Credentialed HTTPS requests with `credentials: include`, OAuth redirects, frontend-safe authenticated/anonymous session state, sign-in/register UI states, authenticated-action gating decisions

**Alternative Analysis (BP6):**

- *Chosen Approach:* First-party SPA authentication surface backed by HttpOnly cookie sessions and generated API clients
- *Alternative Considered:* Local development auth shortcut or manual cookie setup for subscription UAT
- *Trade-off:* A real frontend auth surface satisfies SW-REQ-058, SW-REQ-061, SW-REQ-071, and SW-REQ-074 while allowing Phase 06 checkout UAT to start from the webapp. Manual cookies or local shortcuts would unblock isolated developer testing but would not verify the user-facing account flow, consent capture, disclaimer display, CSRF behavior, or authenticated checkout precondition.

**Resource Goals (optional):**

- *CPU:* Auth UI state updates must not block search input responsiveness.
- *Memory:* Frontend session state stores only user-safe projection data and no token strings.
- *Network:* Session probes and entitlement refreshes should be deduplicated through TanStack Query cache keys to avoid repeated protected-route calls during startup and checkout retry flows.
