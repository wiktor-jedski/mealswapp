## FILE: DESIGN-007.md
**Traceability:** ARCH-007

**Static aspects covered:** SubscriptionController, StripeWebhookHandler, EntitlementManager, TrialTracker, UsageLimiter.

### 0. Static Aspect Responsibilities
- `SubscriptionController`: owns checkout creation, entitlement lookup, billing-state endpoints, and user-facing payment state.
- `StripeWebhookHandler`: owns signature verification, idempotency, event dispatch, and retry-aware responses.
- `EntitlementManager`: owns tier/status persistence and feature access decisions.
- `TrialTracker`: owns one-time 7-day trial creation, expiry detection, and downgrade.
- `UsageLimiter`: owns 24-hour free-tier usage windows and feature-specific usage recording.

### 1. Data Structures & Types
- `type SubscriptionTier = "free" | "trial" | "paid"`
- `interface Entitlement { userId: UUID; tier: SubscriptionTier; status: "active" | "expired" | "past_due" | "cancelled"; searchLimitPer24h: number; allowedModes: SearchMode[]; expiresAt?: time.Time; stripeCustomerId?: string; stripeSubscriptionId?: string }`
- `interface UsageWindow { userId: UUID; feature: string; startedAt: time.Time; searchCount: number }`
- `interface PaymentIntentRequest { userId: UUID; priceId: string; successUrl: string; cancelUrl: string }`
- `interface StripeWebhookEvent { id: string; type: string; payload: []byte; signature: string; receivedAt: time.Time }`
- `interface ProcessedEvent { eventId: string; eventType: string; processedAt: time.Time; outcome: "success" | "duplicate" | "failed"; payload: []byte }`

### 2. Logic & Algorithms (Step-by-Step)
1. Entitlement checks load the user's active tier from ARCH-005 before protected feature execution.
2. Free users are limited to 3 searches per 24-hour rolling window and `single` mode only.
3. Trial and paid users receive unlimited searches and all modes while `status = active`.
4. Payment setup creates a Stripe payment or checkout session; raw card data remains in Stripe Elements.
5. Webhook handler verifies `Stripe-Signature` before parsing the event.
6. Store `event.id` in `processed_events` inside a transaction; duplicates return 200 without repeating side effects.
7. For successful payment or subscription events, update entitlement and usage limits in the same transaction.
8. For failed or cancelled payment events, mark entitlement as `past_due` or `cancelled` without deleting history.
9. Trial tracker creates one 7-day trial on first social login and auto-downgrades expired trials to free.
10. Hourly reconciliation queries Stripe for active subscriptions and fixes local entitlement drift.

### 3. State Management & Error Handling
- `free_active`: limited usage and single-item mode only.
- `trial_active`: full access until `expiresAt`.
- `paid_active`: full access while Stripe subscription is active.
- `past_due`: block paid-only features and show billing recovery state.
- `expired`: downgrade to free.
- `webhook_duplicate`: return 200 and make no changes.
- `webhook_signature_invalid`: return 400 and log security event.
- `entitlement_write_failed`: return 500 to Stripe so webhook retries; also write dead-letter entry when possible.
- `stripe_unavailable`: return 503 for checkout creation and leave entitlement unchanged.

### 4. Component Interfaces
- `func (c *SubscriptionController) CreateCheckout(ctx *fiber.Ctx) error`
- `func (c *SubscriptionController) GetEntitlement(ctx *fiber.Ctx) error`
- `func (h *StripeWebhookHandler) Handle(ctx *fiber.Ctx) error`
- `func CheckEntitlement(ctx context.Context, userID UUID, feature string) (Decision, error)`
- `func RecordUsage(ctx context.Context, userID UUID, feature string) error`
- `func StartTrial(ctx context.Context, userID UUID) (Entitlement, error)`
- `func ExpireTrials(ctx context.Context, now time.Time) error`
- `func ReconcileStripeEntitlements(ctx context.Context) error`
- `type EntitlementRepository interface { AppendEntitlement(ctx context.Context, entitlement Entitlement) error; GetLatest(ctx context.Context, userID UUID) (Entitlement, error) }`
- `type UsageRepository interface { RecordUsage(ctx context.Context, userID UUID, feature string, occurredAt time.Time) (UsageWindow, error); GetUsageSince(ctx context.Context, userID UUID, feature string, since time.Time) (UsageWindow, error) }`
- `type TrialRepository interface { ListExpiredTrials(ctx context.Context, now time.Time) ([]Entitlement, error) }`
- `type StripeEventRepository interface { InsertProcessedStripeEvent(ctx context.Context, event ProcessedEvent) (bool, error) }`
