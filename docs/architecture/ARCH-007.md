# [ARCH-007] - Subscription Module

**Description:** Service managing subscription tiers, payment processing via Stripe, entitlement enforcement, and trial period logic.

| Attribute | Value |
| :--- | :--- |
| **Type** | Service |
| **Static Aspects** | SubscriptionController, StripeWebhookHandler, EntitlementManager, TrialTracker, UsageLimiter |
| **Dependencies** | ARCH-006 (Authentication), ARCH-005 (Data Repository), Stripe API |
| **Traceability** | SW-REQ-042, SW-REQ-044, SW-REQ-045, SW-REQ-050, SW-REQ-051, SW-REQ-052, SW-REQ-053 |

**Dynamic Behavior:**

- **Tier Enforcement:** Checks user entitlement on each request. Free tier: 3 searches/24h, single-item only. Paid/Trial: unlimited, all features.
- **Payment Flow:** Client uses Stripe Elements (PCI-DSS compliant tokenization). Server creates Payment Intents, never handles raw card data.
- **Webhook Processing:** Asynchronously processes payment_intent.succeeded/failed events to update entitlement status reliably.
- **Trial Management:** Activates 7-day trial on first social login. Tracks expiration timestamp. Auto-downgrades to Free tier on expiry.

**Interface Definition:**

- `Input`: Subscription requests, Stripe webhook events, entitlement checks
- `Output`: Entitlement status, payment session URLs, feature access decisions

**Alternative Analysis (BP6):**

- *Chosen Approach:* Stripe with server-side webhook processing for entitlement sync
- *Alternative Considered:* Client-side payment confirmation with polling
- *Trade-off:* Webhook-based sync (SW-REQ-045) ensures reliable entitlement updates even if user closes browser during payment. Polling would miss events and create inconsistent states. Stripe Elements ensure PCI-DSS scope reduction (SW-REQ-044) by tokenizing at client.

### Webhook Handling

**Idempotency:**
- Store `event.id` in `processed_events` table before processing
- On duplicate webhook delivery, return 200 OK without reprocessing
- Prevents double-crediting or duplicate entitlement updates

**Retry Policy Awareness:**
- Stripe retries failed webhooks for up to 3 days with exponential backoff
- Handler must be idempotent to safely handle retries
- Return 2xx status only after successful processing; 4xx/5xx triggers retry

**Signature Verification:**
- Validate `Stripe-Signature` header using webhook signing secret
- Reject webhooks with invalid or missing signatures (return 400)
- Prevents spoofed webhook attacks

### Partial Failure Recovery

**Scenario:** Payment succeeds at Stripe, but local entitlement database write fails.

**Solution:**
1. Webhook handler wraps entitlement update in database transaction
2. On transaction failure, log event to dead-letter queue with full payload
3. Return 500 to Stripe (triggers automatic retry)
4. Reconciliation job runs hourly: queries Stripe API for active subscriptions, compares with local entitlements, fixes discrepancies

**User-Facing Behavior:**
- During payment processing, UI shows "Payment processing..." state
- Entitlement confirmed only after webhook successfully processed
- If webhook fails repeatedly, reconciliation job catches within 1 hour

### Payment Flow Diagram

```
┌──────────┐     ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  Client  │────>│ Stripe      │────>│ ARCH-007    │────>│ ARCH-005    │
│          │     │ Checkout    │     │ Webhook     │     │ Repository  │
└──────────┘     └─────────────┘     └─────────────┘     └─────────────┘
     │                  │                   │                   │
     │  1. Redirect     │                   │                   │
     │─────────────────>│                   │                   │
     │                  │                   │                   │
     │  2. User pays    │                   │                   │
     │                  │                   │                   │
     │                  │ 3. payment_intent │                   │
     │                  │    .succeeded     │                   │
     │                  │──────────────────>│                   │
     │                  │                   │                   │
     │                  │                   │ 4. Verify         │
     │                  │                   │    signature      │
     │                  │                   │                   │
     │                  │                   │ 5. Check          │
     │                  │                   │    idempotency    │
     │                  │                   │                   │
     │                  │                   │ 6. BEGIN TXN      │
     │                  │                   │──────────────────>│
     │                  │                   │                   │
     │                  │                   │ 7. Update         │
     │                  │                   │    entitlement    │
     │                  │                   │<──────────────────│
     │                  │                   │                   │
     │                  │                   │ 8. Log event      │
     │                  │                   │──────────────────>│
     │                  │                   │                   │
     │                  │                   │ 9. COMMIT         │
     │                  │                   │                   │
     │                  │   10. HTTP 200    │                   │
     │                  │<──────────────────│                   │
     │                  │                   │                   │
     │ 11. Return URL   │                   │                   │
     │<─────────────────│                   │                   │
     │                  │                   │                   │
     │ 12. Fetch        │                   │                   │
     │    entitlement   │                   │                   │
     │─────────────────────────────────────>│                   │
     │                  │                   │                   │
     │ 13. Confirmed    │                   │                   │
     │<─────────────────────────────────────│                   │
```
