# Phase 06 UAT: Subscription and Entitlement Enforcement

<!-- Implements DESIGN-007 SubscriptionController, StripeWebhookHandler, EntitlementManager, TrialTracker, UsageLimiter. -->

## Scope

Phase 06 covers tasks `157`-`174`. Tasks `157`-`173` implement and verify the
subscription and entitlement surface: Stripe sandbox configuration,
free/trial/paid entitlement decisions, rolling free-tier usage limits, search gating,
trial activation and expiry, hosted checkout creation, entitlement status
responses, Stripe webhook processing, dead-letter persistence, reconciliation,
OpenAPI billing contracts, generated frontend billing types, frontend
entitlement state, search UI gating, subscription checkout UI, Stripe sandbox
verification, billing workflow integration, and the aggregate verification gate.
Task `174` is this acceptance document.

The implemented Phase 06 surface follows `docs/design/DESIGN-007.md`:

- `SubscriptionController`: authenticated checkout creation and entitlement
  status endpoints, idempotency-key handling, generated OpenAPI contracts, and
  frontend billing controls.
- `StripeWebhookHandler`: signature verification, provider event idempotency,
  successful/past-due/cancelled entitlement projection, retry-aware 500
  responses, and sanitized dead-letter persistence.
- `EntitlementManager`: free, trial, paid, expired, past-due, and cancelled
  access decisions for Catalog, single-input Substitution, multi-input
  Substitution, Daily Diet, and Daily Diet Alternative search modes.
- `TrialTracker`: one-time 7-day trial activation on first social login and
  idempotent trial expiry downgrade to free.
- `UsageLimiter`: PostgreSQL-backed rolling 24-hour free-tier usage windows,
  usage recording after allowed completed searches, and persisted concurrency
  protection.

Phase 06 uses Stripe-hosted Checkout as recorded in
`docs/implementation/04_OPEN.md`. Raw card data is not accepted by backend
checkout DTOs and no PAN/CVC fields are rendered by the application billing UI.
Before production billing launch, the project owner must confirm whether
`SW-REQ-044` should continue to allow Stripe Checkout or be rewritten to require
embedded Stripe Elements specifically.

## Automated Evidence

Run from the repository root unless noted. These commands were actually run
during Phase 06 task preparation and review:

```sh
python3 scripts/validate-task-list.py
python3 scripts/validate-traceability.py
python3 scripts/check.py
npx --no-install redocly lint api/openapi.yaml
python3 scripts/generate-api-types.py --check
python3 -m py_compile scripts/verify-stripe-cli-sandbox.py
python3 scripts/verify-stripe-cli-sandbox.py --commands-only
cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./...
cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/...
cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/... -coverprofile=coverage.out
cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go vet ./...
cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./...
cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...
cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/httpapi ./internal/subscription
cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/httpapi -run TestPhase06BillingWorkflowIntegrationGate -count=1
cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/auth -run TestCoreAuthServiceOAuthRealTrialTracker -count=1
cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/entitlement -run 'Test(TrialTracker|EntitlementManager|UsageLimiterDoesNotCap|UsageLimiterValidation|UsageLimiterAllows|UsageLimiterBlocks|UsageLimiterRejects|UsageLimiterUses|UsageLimiterDoes)' -count=1
cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/subscription -run 'Test(Checkout|StripeWebhook|Reconciliation|Webhook)' -count=1
cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./cmd/expire-trials -count=1
cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run generate:api-types
cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run check:api-types
cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test
cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test --coverage
cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run build
cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run test:e2e -- tests/search-workflow.spec.ts tests/subscription-billing.spec.ts
```

Observed results from task review evidence:

- Task `173` recorded `python3 scripts/check.py` passing end to end.
- `python3 scripts/validate-task-list.py` passed with `175` sequential tasks
  and ordered dependencies.
- `python3 scripts/validate-traceability.py` passed.
- Redocly OpenAPI lint passed.
- Generated billing/search frontend API type drift checks passed.
- Backend `go test ./...`, `go vet ./...`, `go test -race ./...`, and
  `govulncheck` passed; `govulncheck` reported no called vulnerabilities.
- Focused Stripe webhook tests passed for `backend/internal/httpapi` and
  `backend/internal/subscription`.
- Frontend unit tests passed with `262 pass`, `0 fail`.
- Frontend coverage reported `All files | 98.85 | 96.76`; accepted deviations
  are recorded in `docs/implementation/04_OPEN.md`.
- Backend internal coverage reported `95.7%`; accepted deviations for
  `backend/internal/entitlement` and `backend/internal/subscription` are
  recorded in `docs/implementation/04_OPEN.md`.
- Playwright billing/search and axe checks passed through the aggregate gate;
  task `173` recorded `123 passed`, `1 skipped`.
- The aggregate gate reused already-running local PostgreSQL and Redis
  listeners when Docker ports `5432` and `6379` were occupied.

## Stripe Sandbox Evidence

Detailed Stripe sandbox evidence is recorded in
`docs/implementation/stripe-cli-sandbox-verification.md` for task `171`.

The committed verification path includes:

- `stripe listen --forward-to
  http://127.0.0.1:8080/api/v1/billing/stripe/webhook`
- local-only `MEALSWAPP_STRIPE_WEBHOOK_SECRET` setup from the Stripe CLI
  printed secret
- `stripe trigger checkout.session.completed`
- `stripe trigger invoice.payment_failed`
- `stripe trigger customer.subscription.deleted`
- deterministic signed local webhook fixtures generated by
  `scripts/verify-stripe-cli-sandbox.py`

Recorded 2026-07-02 evidence states that the Stripe CLI was not installed on
the verification host, so live `stripe listen` forwarding was not executed.
Instead, the deterministic signed verifier was run against a clean migrated
local backend and database. It passed these checks:

- valid signed `checkout.session.completed` accepted and projected
  `paid:active`
- invalid `Stripe-Signature` rejected with `400`
- duplicate provider event accepted without duplicate entitlement history
- `invoice.payment_failed` accepted and projected `paid:past_due`
- `customer.subscription.deleted` accepted and projected `paid:cancelled`
- forced entitlement write failure returned `500` for Stripe retry
- sanitized dead-letter metadata persisted for the retry-producing failure

No real Stripe keys, real customer identifiers, card data, or customer email
addresses are committed in the verifier or evidence document.

## Project-Owner Checks

### Integration Checks

1. Start local dependencies with `bash scripts/start-services.sh`.
2. Run migrations from `backend/` with
   `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go run ./cmd/migrate up`.
3. Start the API with local Stripe sandbox configuration:
   `MEALSWAPP_STRIPE_WEBHOOK_SECRET=<local whsec> GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go run ./cmd/api`.
4. From `frontend/`, run `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run dev`.
5. Confirm `GET /api/v1/billing/entitlement` returns authenticated entitlement
   status with tier, status, allowed modes, usage remaining, trial expiry, and
   billing recovery state.
6. Confirm `POST /api/v1/billing/checkout` requires authentication, CSRF, and
   `Idempotency-Key`, returns only a hosted checkout URL/session response, and
   never receives PAN/CVC/card-number fields.
7. Confirm Stripe webhook delivery to
   `POST /api/v1/billing/stripe/webhook` updates local entitlement state even if
   the browser is closed after checkout.

### Functional Checks

1. **Free entitlement (SW-REQ-042, SW-REQ-053):** Sign in as a free user and
   run Catalog or single-input Substitution searches. Confirm at most `3`
   completed counted searches are allowed per rolling 24 hours, denied attempts
   are not counted, and multi-input Substitution, Daily Diet, and Daily Diet
   Alternative are blocked before search dispatch.
2. **Trial entitlement (SW-REQ-051, SW-REQ-052):** Create a new account through
   social login. Confirm exactly one active 7-day trial is created, repeated
   social login does not extend the trial, paid modes are unlocked while the
   trial is active, and the expiry command downgrades expired trials to free
   without deleting history.
3. **Paid entitlement (SW-REQ-045, SW-REQ-050, SW-REQ-052):** Start monthly and
   annual checkout flows. Confirm monthly maps to `$3.00`, annual maps to
   `$25.00`, successful webhook processing projects `paid:active`, failed
   payment projects `past_due`, cancelled subscription projects `cancelled`,
   and paid-only features are available only while the entitlement is active.
4. **Anonymous Catalog:** Confirm anonymous Catalog search remains available
   and does not create entitlement usage writes.
5. **Checkout idempotency:** Retry checkout creation with the same
   `Idempotency-Key` and identical normalized request body; confirm the stored
   response is returned. Retry with the same key and a different body; confirm a
   conflict response and no new Stripe checkout session.

### End-to-End Checks

1. Run the billing workflow integration gate:
   `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/httpapi -run TestPhase06BillingWorkflowIntegrationGate -count=1`.
2. Run frontend billing/search Playwright coverage:
   `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run test:e2e -- tests/search-workflow.spec.ts tests/subscription-billing.spec.ts`.
3. Confirm the covered flows include free limit exhaustion, trial unlock from
   social login, paid unlock after webhook, duplicate webhook non-reapplication,
   checkout idempotency retry, anonymous Catalog Search, blocked paid-mode UI
   with no network search side effects, billing success/cancel returns, and
   billing recovery state rendering.

### Security Checks

1. Confirm no backend checkout DTO or frontend billing form accepts card PAN,
   CVC, expiry, or card-number fields (SW-REQ-044).
2. Confirm Stripe webhook processing rejects missing or invalid signatures and
   logs a security event without applying entitlement side effects.
3. Confirm duplicate provider event IDs return `200` without duplicate
   entitlement history or usage side effects.
4. Confirm webhook processing failures return retryable `500` responses and
   persist only sanitized dead-letter metadata plus payload hashes.
5. Confirm entitlement and checkout routes derive user scope from authenticated
   server state, not client-supplied user IDs.
6. Run `go vet`, `govulncheck`, race tests, OpenAPI lint, generated-type drift
   checks, and frontend axe checks before production billing launch.

### Acceptance Decision

Phase 06 is ready for project-owner acceptance when:

- tasks `157`-`173` remain prepared or are promoted by the owner according to
  their review evidence;
- this task `174` document remains current with the latest validation evidence;
- `python3 scripts/check.py`, `python3 scripts/validate-task-list.py`, and
  `python3 scripts/validate-traceability.py` pass on the final worktree;
- the project owner accepts the documented coverage deviations in
  `docs/implementation/04_OPEN.md`;
- the project owner accepts the Stripe Checkout interpretation for
  `SW-REQ-044`, or updates the requirement/task plan before production billing
  launch;
- Stripe sandbox forwarding is performed with a locally installed Stripe CLI
  before production credentials are introduced.

## Traceability

Primary design source:

- `docs/design/DESIGN-007.md`: `SubscriptionController`,
  `StripeWebhookHandler`, `EntitlementManager`, `TrialTracker`, and
  `UsageLimiter`.

Related Phase 06 task IDs:

- `157` Phase 06 Billing Configuration.
- `158` Phase 06 Entitlement Decision Service.
- `159` Phase 06 Usage Limiter.
- `160` Phase 06 Search Entitlement Gate.
- `161` Phase 06 Trial Activation and Expiry.
- `162` Phase 06 Checkout Creation.
- `163` Phase 06 Entitlement Status API.
- `164` Phase 06 Stripe Webhook Verification and Idempotency.
- `165` Phase 06 Stripe Dead Letter and Reconciliation.
- `166` Phase 06 Billing OpenAPI Contract.
- `167` Phase 06 Frontend Billing Type Generation.
- `168` Phase 06 Frontend Entitlement Client.
- `169` Phase 06 Search UI Entitlement Gating.
- `170` Phase 06 Subscription UI and Checkout Flow.
- `171` Phase 06 Stripe CLI Sandbox Verification.
- `172` Phase 06 Billing Workflow Integration Gate.
- `173` Phase 06 Coverage and Aggregate Gate.
- `174` Phase 06 Acceptance Documentation.

Requirement coverage:

- `SW-REQ-042` Free Tier Search Limitation: tasks `159`, `160`, `169`, `172`.
- `SW-REQ-044` Secure Credential Tokenization: tasks `157`, `162`, `170`,
  `171`; implemented through Stripe-hosted Checkout with no raw payment fields
  in the application server or UI. Product wording still needs owner
  confirmation before production billing launch.
- `SW-REQ-045` Payment Status Synchronization: tasks `164`, `165`, `171`,
  `172`.
- `SW-REQ-050` Subscription Pricing Tiers: tasks `157`, `162`, `166`, `170`.
- `SW-REQ-051` Promotional Trial Logic: tasks `161`, `172`.
- `SW-REQ-052` Paid Tier Exclusive Features: tasks `158`, `160`, `163`,
  `168`, `169`, `172`.
- `SW-REQ-053` Free Tier Functional Scope: tasks `158`, `160`, `163`, `169`,
  `172`.

## Known Notes

- `docs/implementation/04_OPEN.md` records accepted Phase 06 coverage
  deviations: backend internal coverage is `95.7%`, with Phase 06 gaps in
  `backend/internal/entitlement` and `backend/internal/subscription`; frontend
  coverage is `All files | 98.85 | 96.76`, with remaining gaps in
  `entitlement-client.ts` and `search-entitlement.ts`.
- `docs/implementation/04_OPEN.md` records the production-launch action to
  confirm Stripe Checkout vs Stripe Elements wording for `SW-REQ-044`.
- `docs/implementation/stripe-cli-sandbox-verification.md` records that the
  Stripe CLI was not installed on the 2026-07-02 verification host. Deterministic
  signed webhook verification passed locally; live `stripe listen` should still
  be performed by the project owner or release engineer before production
  billing launch.
