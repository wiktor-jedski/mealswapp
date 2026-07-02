# Phase 06 Stripe CLI Sandbox Verification

Implements DESIGN-007 StripeWebhookHandler for task 171.

## Scope

This document records the local Stripe sandbox verification path for the webhook endpoint:

- endpoint: `POST /api/v1/billing/stripe/webhook`
- design source: `docs/design/DESIGN-007.md`, `StripeWebhookHandler`
- fixture user: `11111111-1111-4111-8111-111111111171`
- fixture customer: `cus_test_task171`
- fixture subscription: `sub_test_task171`

No real Stripe keys, real customer identifiers, email addresses, or card data are used by the committed script or this document.

## Local Command Sequence

Start the local backend and dependencies:

```bash
bash scripts/start-services.sh
cd backend
GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go run ./cmd/migrate up
MEALSWAPP_STRIPE_WEBHOOK_SECRET=whsec_local_fixture GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go run ./cmd/api
```

In another shell, run the safe generated-fixture verifier:

```bash
python3 scripts/verify-stripe-cli-sandbox.py --webhook-secret whsec_local_fixture
```

When `psql` is available and the local database is reachable, include the database assertion. The verifier prepares only its deterministic `task171` fixture user/events:

```bash
MEALSWAPP_DATABASE_URL='postgres://mealswapp:mealswapp@localhost:5432/mealswapp?sslmode=disable' \
python3 scripts/verify-stripe-cli-sandbox.py --webhook-secret whsec_local_fixture
```

## Stripe CLI Forwarding

For real Stripe sandbox forwarding, start the backend with the webhook secret printed by `stripe listen`, then trigger sandbox fixtures:

```bash
stripe listen --forward-to http://127.0.0.1:8080/api/v1/billing/stripe/webhook
export MEALSWAPP_STRIPE_WEBHOOK_SECRET='<whsec value printed by stripe listen>'
stripe trigger checkout.session.completed
stripe trigger invoice.payment_failed
stripe trigger customer.subscription.deleted
```

The `stripe listen` secret is local-only runtime configuration. Do not commit it.

## Expected Evidence

`python3 scripts/verify-stripe-cli-sandbox.py --commands-only` prints the Stripe CLI sequence and a deterministic signed `curl` fixture without sending data.

Against a running backend, the verifier records:

- valid signed `checkout.session.completed` fixture returns `200`
- latest local entitlement becomes `paid:active`
- invalid `Stripe-Signature` returns `400`
- repeated provider event ID returns `200` with `duplicate=true`
- entitlement history count remains unchanged after duplicate delivery
- `invoice.payment_failed` fixture returns `200` and projects `past_due`
- `customer.subscription.deleted` fixture returns `200` and projects `cancelled`
- database assertion confirms latest entitlement is `paid:cancelled:cus_test_task171:sub_test_task171`
- controlled local entitlement write failure returns `500` for Stripe retry and persists sanitized dead-letter metadata with a payload hash

## Recorded Safe Fixture Evidence

Local static verification run on 2026-07-02:

```text
$ python3 scripts/verify-stripe-cli-sandbox.py --commands-only
Stripe CLI command sequence:
  stripe listen --forward-to http://127.0.0.1:8080/api/v1/billing/stripe/webhook
  export MEALSWAPP_STRIPE_WEBHOOK_SECRET='<whsec value printed by stripe listen>'
  stripe trigger checkout.session.completed
  stripe trigger invoice.payment_failed
  stripe trigger customer.subscription.deleted

Use the generated webhook secret only for the local process. Do not commit it.

Deterministic signed fixture example:
  curl -i -X POST http://127.0.0.1:8080/api/v1/billing/stripe/webhook \
    -H 'Content-Type: application/json' \
    -H 'Stripe-Signature: t=<generated timestamp>,v1=<generated hmac>' \
    --data '{"data":{"object":{"client_reference_id":"11111111-1111-4111-8111-111111111171","customer":"cus_test_task171","id":"cs_test_task171","metadata":{"user_id":"11111111-1111-4111-8111-111111111171"},"status":"","subscription":"sub_test_task171"}},"id":"evt_task171_checkout_completed","type":"checkout.session.completed"}'
```

Focused backend behavior already covered by task 164 and task 165 tests:

```bash
cd backend
GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/httpapi ./internal/subscription
```

Recorded focused result on 2026-07-02:

```text
ok  	github.com/wiktor-jedski/mealswapp/backend/internal/httpapi	(cached)
ok  	github.com/wiktor-jedski/mealswapp/backend/internal/subscription	0.006s
```

Those tests cover valid signature acceptance, invalid signature rejection, duplicate event idempotency, retry-producing store failures, `past_due` mapping for failed payments, `cancelled` mapping for deleted subscriptions, and sanitized dead-letter metadata.

## Recorded Backend Verification Evidence

The Stripe CLI was not installed on the verification host on 2026-07-02, so `stripe listen` could not be executed directly. The deterministic verifier was run against a clean migrated local backend/database instead.

Setup run on 2026-07-02:

```bash
docker compose up -d --force-recreate postgres
psql 'postgres://mealswapp:mealswapp@localhost:5432/postgres?sslmode=disable' -c 'DROP DATABASE IF EXISTS mealswapp_task171'
psql 'postgres://mealswapp:mealswapp@localhost:5432/postgres?sslmode=disable' -c 'CREATE DATABASE mealswapp_task171'
cd backend
MEALSWAPP_DATABASE_URL='postgres://mealswapp:mealswapp@localhost:5432/mealswapp_task171?sslmode=disable' \
GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go run ./cmd/migrate up
MEALSWAPP_HTTP_PORT=18082 \
MEALSWAPP_DATABASE_URL='postgres://mealswapp:mealswapp@localhost:5432/mealswapp_task171?sslmode=disable' \
MEALSWAPP_STRIPE_WEBHOOK_SECRET=whsec_local_fixture \
GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go run ./cmd/api
```

Verifier run on 2026-07-02:

```bash
MEALSWAPP_DATABASE_URL='postgres://mealswapp:mealswapp@localhost:5432/mealswapp_task171?sslmode=disable' \
python3 scripts/verify-stripe-cli-sandbox.py \
  --webhook-url http://127.0.0.1:18082/api/v1/billing/stripe/webhook \
  --webhook-secret whsec_local_fixture
```

Sanitized verifier output:

```text
PASS local database prepared with deterministic task-171 fixture user
PASS valid signed checkout event accepted
PASS local database latest entitlement is paid:active after checkout
PASS invalid signature rejected
PASS duplicate provider event accepted without duplicate side effects
PASS duplicate delivery left entitlement history count unchanged
PASS payment failure event accepted for past_due projection
PASS local database latest entitlement is paid:past_due after failed payment
PASS subscription deletion event accepted for cancelled projection
PASS local database latest entitlement is paid:cancelled after failed/cancelled fixtures
PASS forced entitlement write failure returned 500 for Stripe retry
PASS sanitized dead-letter metadata persisted for retry-producing failure
Stripe sandbox webhook verification passed.
```
