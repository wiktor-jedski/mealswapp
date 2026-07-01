# ARCH-007 Integration Verification Obligations

## Purpose

This document defines the SWE.5 integration verification obligations for architecture component ARCH-007, the Subscription Module.

The goal is to verify that SubscriptionController, StripeWebhookHandler, EntitlementManager, TrialTracker, UsageLimiter, authentication, search routing, repository persistence, Stripe gateway adapters, and frontend billing/search gating collaborate according to the architecture.

## Component Information

| Field | Value |
| --- | --- |
| Architecture Component | ARCH-007 |
| Name | Subscription Module |
| Source Documents | `docs/architecture/ARCH-007.md`, `docs/design/DESIGN-007.md`, `docs/implementation/01_PLAN.md` |
| Related Units | SubscriptionController, StripeWebhookHandler, EntitlementManager, TrialTracker, UsageLimiter, checkout idempotency store, Stripe gateway, entitlement repositories, search controller, frontend entitlement client, billing UI |
| Collaborating Architecture | ARCH-001, ARCH-002, ARCH-005, ARCH-006, ARCH-010, ARCH-013, ARCH-014, ARCH-017 |
| Related Requirements | SW-REQ-042, SW-REQ-044, SW-REQ-045, SW-REQ-050, SW-REQ-051, SW-REQ-052, SW-REQ-053 |

## IT-ARCH-007-001 Search Gateway Entitlement and Usage Enforcement

### Intent

Verify that authenticated search requests pass through the API gateway into entitlement and usage checks before the Search Module dispatches paid-mode work, while anonymous Catalog Search remains available.

### System Under Test

ARCH-007 Subscription Module, centered on EntitlementManager and UsageLimiter as used by the SearchController boundary.

### Real Components

- SearchController
- EntitlementManager
- UsageLimiter
- UsageRepository
- EntitlementRepository
- Fiber gateway routing and authenticated-user derivation
- Search dispatcher/service boundary

### Allowed Test Doubles

- Search service test doubles may be used to prove allowed/denied dispatch behavior.
- Repository test doubles may be used only when a PostgreSQL integration fixture is impractical for the specific assertion.

### Trigger / Stimulus

Anonymous, free authenticated, trial authenticated, and paid authenticated users submit Catalog, single-input Substitution, multi-input Substitution, Daily Diet, and Daily Diet Alternative search requests.

### Expected Integrated Behavior

1. Anonymous Catalog Search dispatches without entitlement persistence side effects.
2. Free authenticated users can dispatch Catalog and single-input Substitution while under the rolling 24-hour limit.
3. The fourth counted free-tier search in a rolling 24-hour window returns a stable entitlement error before search dispatch.
4. Free users cannot dispatch multi-input Substitution, Daily Diet, or Daily Diet Alternative.
5. Trial and paid active users can dispatch paid modes without the free usage cap.
6. Denied requests do not write search history, cache entries, or usage records.

### Required Evidence

- Test verifies HTTP response status/body, allowed dispatch, denied no-dispatch, usage count behavior, and absence of side effects for denied requests.
- Test traceability comment references `IT-ARCH-007-001`, `ARCH-007`, `ARCH-002`, and related SW requirements.

### Requirement Traceability

- SW-REQ-042
- SW-REQ-052
- SW-REQ-053

### Verification Status

Planned for Phase 06.

## IT-ARCH-007-002 OAuth First-Login Trial Activation and Expiry

### Intent

Verify that Authentication Module social login collaborates with TrialTracker to create exactly one 7-day trial for first social login and later downgrade expired trials without corrupting paid entitlement history.

### System Under Test

ARCH-007 Subscription Module, centered on TrialTracker integrated with ARCH-006 OAuth login.

### Real Components

- OAuth authentication service
- TrialTracker
- EntitlementManager
- EntitlementRepository
- TrialRepository
- Expiry command or worker entrypoint

### Allowed Test Doubles

- OAuth provider gateway test doubles may provide verified social identity claims.
- Time source test doubles may control trial creation and expiry timestamps.

### Trigger / Stimulus

A new user completes social login, repeats social login, then trial expiry processing runs after 168 hours; a paid user with expired trial history is processed as a separate case.

### Expected Integrated Behavior

1. First social login appends one active trial entitlement expiring 168 hours after creation.
2. Repeated social login does not append another trial or extend expiry.
3. Expiry processing downgrades expired active trial users to free.
4. Expiry processing does not downgrade users whose latest entitlement is paid active.
5. Repeated expiry runs are idempotent.

### Required Evidence

- Test verifies OAuth-to-trial collaboration, entitlement history, expiry downgrade, paid-user protection, and idempotent repeated processing.
- Test traceability comment references `IT-ARCH-007-002`, `ARCH-007`, `ARCH-006`, and related SW requirements.

### Requirement Traceability

- SW-REQ-046
- SW-REQ-051
- SW-REQ-052

### Verification Status

Planned for Phase 06.

## IT-ARCH-007-003 Checkout Creation and Idempotency Flow

### Intent

Verify that authenticated checkout creation combines SubscriptionController, EntitlementManager, Stripe gateway, idempotency persistence, and gateway error mapping without ever accepting raw payment-card data.

### System Under Test

ARCH-007 Subscription Module, centered on SubscriptionController checkout creation.

### Real Components

- SubscriptionController
- Authenticated route middleware
- EntitlementManager
- Checkout idempotency store
- Stripe gateway adapter boundary
- AppError mapping

### Allowed Test Doubles

- Stripe gateway test double may stand in for Stripe sandbox API while preserving request and response semantics.

### Trigger / Stimulus

An authenticated user requests monthly and annual checkout sessions, repeats an exact request with the same `Idempotency-Key`, reuses the key with a different body, and exercises Stripe gateway failure.

### Expected Integrated Behavior

1. User identity is derived from validated authentication context only.
2. Monthly and annual plans map to configured Stripe price IDs for SW-REQ-050.
3. Exact idempotent retries return the stored checkout response without creating another Stripe session.
4. Key reuse with a different normalized body is rejected.
5. Stripe unavailability returns 503 and leaves entitlement unchanged.
6. Backend request DTOs reject raw PAN/CVC/card-number fields.

### Required Evidence

- Test verifies authentication, price mapping, Stripe call count, idempotent replay, key-conflict rejection, failure mapping, and no raw-card fields accepted or logged.
- Test traceability comment references `IT-ARCH-007-003`, `ARCH-007`, `ARCH-010`, and related SW requirements.

### Requirement Traceability

- SW-REQ-044
- SW-REQ-050

### Verification Status

Planned for Phase 06.

## IT-ARCH-007-004 Stripe Webhook Idempotency, Failure, and Retry Behavior

### Intent

Verify that StripeWebhookHandler verifies signatures, records provider event IDs, updates entitlements transactionally, returns retry-aware status codes, and ignores duplicate deliveries.

### System Under Test

ARCH-007 Subscription Module, centered on StripeWebhookHandler.

### Real Components

- StripeWebhookHandler
- Stripe signature verifier
- StripeEventRepository
- EntitlementRepository
- Dead-letter persistence
- Security audit logging
- Fiber gateway route

### Allowed Test Doubles

- Stripe CLI sandbox fixtures or deterministic signed fixture payloads may stand in for live Stripe delivery.
- Database failure injection may be used to verify retry-aware failure status.

### Trigger / Stimulus

Valid signed success, failure, cancellation, duplicate, invalid-signature, and persistence-failure webhook events are delivered to the webhook route.

### Expected Integrated Behavior

1. Invalid or missing signatures return 400 and record a security event without entitlement changes.
2. New valid provider event IDs are recorded before repeatable side effects where possible.
3. Successful checkout/subscription events append paid active entitlement state.
4. Failed payment events append past_due state without deleting entitlement history.
5. Cancelled subscription events append cancelled state.
6. Duplicate event delivery returns 200 without duplicate entitlement history or usage effects.
7. Entitlement write failure returns 500 so Stripe retries and stores sanitized dead-letter metadata when possible.

### Required Evidence

- Test verifies route status, signature handling, processed event idempotency, entitlement history rows, duplicate non-reapplication, dead-letter metadata, and retry-aware 500 behavior.
- Test traceability comment references `IT-ARCH-007-004`, `ARCH-007`, `ARCH-013`, and related SW requirements.

### Requirement Traceability

- SW-REQ-045
- SW-REQ-052

### Verification Status

Planned for Phase 06.

## IT-ARCH-007-005 Reconciliation Repairs Entitlement Drift

### Intent

Verify that the reconciliation job compares Stripe subscription state to local entitlement history and repairs missing or stale local records without duplicating existing entitlement state.

### System Under Test

ARCH-007 Subscription Module, centered on reconciliation behavior.

### Real Components

- Reconciliation job or command
- Stripe gateway adapter boundary
- EntitlementManager
- EntitlementRepository
- Observability warning path

### Allowed Test Doubles

- Stripe gateway test double may provide active, past_due, cancelled, and unavailable subscription states.

### Trigger / Stimulus

The reconciliation job runs against local users whose latest entitlement differs from Stripe sandbox fixtures, then runs again with the same fixtures.

### Expected Integrated Behavior

1. Missing paid active entitlement state is appended when Stripe shows an active subscription.
2. Local active paid state is repaired to past_due or cancelled when Stripe reports those states.
3. Re-running reconciliation with the same Stripe state does not duplicate entitlement rows.
4. Stripe gateway failure leaves local entitlement unchanged and emits observable warning metadata.

### Required Evidence

- Test verifies local-before/local-after entitlement state, idempotent second run, Stripe failure no-op, and observability warning.
- Test traceability comment references `IT-ARCH-007-005`, `ARCH-007`, and related SW requirements.

### Requirement Traceability

- SW-REQ-045
- SW-REQ-052

### Verification Status

Planned for Phase 06.

## IT-ARCH-007-006 Frontend Billing and Search Gating Collaboration

### Intent

Verify that the Web Application Module consumes generated entitlement contracts, presents free/trial/paid billing state, blocks paid-mode UI execution for free users, and starts checkout without collecting raw card data.

### System Under Test

ARCH-007 Subscription Module as consumed by ARCH-001 frontend billing and search-gating workflows.

### Real Components

- Frontend entitlement client
- TanStack Query billing state
- SearchView mode controls
- Search UI entitlement feedback
- Billing controls
- Generated OpenAPI frontend types

### Allowed Test Doubles

- Playwright route interception may stand in for API responses while preserving generated frontend response shapes.
- Stripe redirect URL may be a sandbox/test URL returned by the backend fixture.

### Trigger / Stimulus

Users load the SPA as anonymous, free, trial, paid, past_due, and cancelled states; free users attempt paid-mode searches; subscribed users start checkout and return from success/cancel routes.

### Expected Integrated Behavior

1. Anonymous Catalog Search remains usable.
2. Free users see remaining usage and cannot trigger blocked paid-mode search network calls.
3. Trial and paid users can execute paid-mode search requests.
4. Past_due and cancelled states show billing recovery actions.
5. Monthly and annual checkout actions call the generated checkout client and follow the server-provided redirect URL.
6. The application UI does not render fields for raw card number, CVC, or PAN capture.
7. Keyboard and axe accessibility gates remain passing for billing and gated search flows.

### Required Evidence

- Playwright tests verify rendered state, network requests or absence of requests, checkout redirect handling, accessibility checks, and no raw-card input fields.
- Test traceability comment references `IT-ARCH-007-006`, `ARCH-007`, `ARCH-001`, and related SW requirements.

### Requirement Traceability

- SW-REQ-042
- SW-REQ-044
- SW-REQ-050
- SW-REQ-052
- SW-REQ-053

### Verification Status

Planned for Phase 06.
