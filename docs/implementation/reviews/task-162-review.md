# Task 162 Review

Task ID: 162

Evidence path: `docs/implementation/reviews/task-162-review.md`

Recommended status: PASSED

Checklist summary:

- PASS: Task 162 is `PREPARED`; dependencies 157 and 158 are `PREPARED`.
- PASS: Authenticated checkout creation is exposed at `POST /api/v1/billing/checkout` with auth and CSRF route hooks.
- PASS: HTTP tests verify user scope comes from JWT cookies and is passed to checkout service input.
- PASS: Missing `Idempotency-Key` is rejected, and reused idempotency keys with different normalized body hashes are rejected.
- PASS: Exact retries return the stored checkout response and do not create another gateway checkout session.
- PASS: Stripe checkout/session creation is behind an injectable `CheckoutGateway`; production wiring injects `StripeCheckoutGateway`.
- PASS: Stripe unavailable maps to `ErrStripeUnavailable` at service level and HTTP 503 at controller level.
- PASS: Repair evidence is now direct: service-level test asserts Stripe failure leaves entitlement fixture unchanged, performs no entitlement append, and stores no checkout idempotency response.
- PASS: Monthly and annual price choices are validated and mapped to configured price IDs and amounts.
- PASS: Request validation rejects raw payment-card fields, gateway tests assert no raw card fields are sent to Stripe, and response mapping contains only hosted checkout/session fields.
- PASS: Traceability comments are present for Task 162 implementation surfaces and `python3 scripts/validate-traceability.py` passes.

Commands run/results:

- `rg -n "\| 162 \||\| 157 \||\| 158 \|" docs/implementation/02_TASK_LIST.md` -> PASS; Task 162, 157, and 158 are all `PREPARED`.
- `git status --short` -> observed prepared/repair changes and existing review files; no task-list status changes made by this review.
- `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/httpapi ./internal/subscription ./internal/repository` from `backend/` -> PASS, cached.
- `python3 scripts/validate-traceability.py` -> PASS; `Traceability validation passed.`
- `python3 scripts/validate-task-list.py` -> PASS; `Task-list validation passed: 175 sequential tasks with ordered dependencies.`
- `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 ./internal/httpapi ./internal/subscription ./internal/repository` from `backend/` -> PASS; httpapi 1.267s, subscription 0.016s, repository 13.709s.

Files inspected:

- `docs/implementation/02_TASK_LIST.md`
- `backend/internal/httpapi/subscription_controller.go`
- `backend/internal/httpapi/subscription_controller_test.go`
- `backend/internal/httpapi/billing_validation.go`
- `backend/internal/httpapi/billing_validation_test.go`
- `backend/internal/subscription/checkout.go`
- `backend/internal/subscription/checkout_test.go`
- `backend/internal/repository/checkout_idempotency_repository.go`
- `backend/internal/repository/checkout_idempotency_repository_test.go`
- `backend/internal/repository/types.go`
- `backend/internal/repository/repository_test.go`
- `backend/internal/repository/sql/checkout_idempotency_get.sql`
- `backend/internal/repository/sql/checkout_idempotency_store.sql`
- `backend/internal/httpapi/router.go`
- `backend/internal/app/app.go`
- `database/migrations/000017_checkout_idempotency.up.sql`
- `database/migrations/000017_checkout_idempotency.down.sql`

Decision reason:

Task 162 now directly satisfies every verification criterion. The repaired `TestCheckoutServiceStripeUnavailableLeavesEntitlementUnchanged` verifies the previously missing Stripe-failure invariant by asserting `ErrStripeUnavailable`, unchanged entitlement fixture, zero entitlement append calls, and zero idempotency-store writes. Controller tests cover authenticated JWT-cookie scope, missing/conflicting idempotency handling, exact replay, 503 mapping, and raw card-field rejection. Service and gateway tests cover monthly/annual price mapping, injectable Stripe checkout-session creation, and absence of raw payment-card fields in Stripe requests and API responses.

Repair instructions if rejected:

None.
