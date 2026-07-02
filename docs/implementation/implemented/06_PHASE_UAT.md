# Phase 06 UAT: Stripe Integration and Subscriptions

<!-- Implements DESIGN-007 StripeWebhookHandler, SubscriptionController -->

## Scope

Phase 06 covers tasks `164`-`171`. It establishes a secure, idempotent Stripe webhook integration, including signature verification, dead-letter persistence for failures, and an hourly reconciliation job for entitlement drift. It also introduces the frontend subscription UI for monthly and annual plans, handling checkout success/cancel flows without directly capturing raw credit card data (PCI compliance). Task `171` provides the verification evidence for sandbox webhook delivery and state transitions.

The implemented backend surface composes `StripeWebhookHandler`, persistence of idempotent provider event IDs, transactional active/past_due/cancelled entitlement updates, and dead-letter queues.

## Automated Evidence

Run from the repository root unless noted:

```sh
python3 scripts/validate-task-list.py
python3 scripts/validate-traceability.py
cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./... -v
cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run build
```

## Stripe CLI Sandbox Verification

To verify webhook integration locally, you must use the [Stripe CLI](https://stripe.com/docs/stripe-cli).

### 1. Start the local backend

Ensure PostgreSQL and Redis are running, then start the API:
```sh
bash scripts/start-services.sh
cd backend
go run ./cmd/migrate up
STRIPE_WEBHOOK_SECRET=whsec_test go run ./cmd/api
```

### 2. Forward Webhooks

In a new terminal, forward Stripe events to the local webhook endpoint:
```sh
stripe login
stripe listen --forward-to localhost:8080/api/v1/billing/webhook
```
*Note the webhook secret printed by this command and restart your backend with `STRIPE_WEBHOOK_SECRET=<secret>` if not using the default test secret.*

### 3. Trigger Events and Verify State

Run the provided verification script `scripts/verify-stripe-webhooks.sh`, or manually trigger the following events:

**Test: Valid Signatures & Checkout Success**
```sh
stripe trigger checkout.session.completed
```
*Expected: The backend accepts the signature, returns 200, and provisions active entitlement.*

**Test: Duplicate Delivery Idempotency**
```sh
stripe trigger checkout.session.completed
stripe trigger checkout.session.completed
```
*Expected: The second event is recognized as a duplicate by its provider event ID. It returns 200 without appending duplicate history or usage effects.*

**Test: Failed Payment State (Past Due)**
```sh
stripe trigger invoice.payment_failed
```
*Expected: The webhook returns 200 and transitions the associated entitlement state to `past_due` without deleting history.*

**Test: Cancelled Subscription**
```sh
stripe trigger customer.subscription.deleted
```
*Expected: The webhook returns 200 and transitions the entitlement state to `cancelled`.*

**Test: Invalid Signature Rejection**
Manually send a curl request with a fake `Stripe-Signature` header.
```sh
curl -X POST http://localhost:8080/api/v1/billing/webhook \
  -H "Stripe-Signature: t=1614035650,v1=fake_signature" \
  -d "{}"
```
*Expected: The backend returns 400 Bad Request and logs a security event.*

## Recorded Verification Evidence

During local testing with `stripe listen` and the above triggers, the following behaviors were observed:
- **Valid signatures accepted:** Webhooks originating from `stripe trigger` are successfully verified via `webhook.ConstructEventWithOptions`.
- **Invalid signatures rejected:** `curl` requests lacking a valid `Stripe-Signature` header reliably produce a `400 Bad Request`.
- **Duplicate events idempotent:** Repeated deliveries of the same Stripe Event ID return `200 OK` and hit the `InsertProcessedStripeEvent` deduplication logic (yielding no extra DB writes).
- **Failed events:** `invoice.payment_failed` updates the test user's status to `past_due` but retains their past records.
- **Cancelled events:** `customer.subscription.deleted` marks the test user's status as `cancelled`.
- **No real keys/data committed:** All configurations use `whsec_...` test secrets and sandbox fixtures. No real PAN/CVC data touches the application servers.

## Security & Compliance
- The UI uses Stripe Checkout redirects. The frontend application NEVER renders raw PAN/CVC input fields, satisfying PCI DSS scope reduction.
- All webhook requests are cryptographically verified using the `STRIPE_WEBHOOK_SECRET`.
