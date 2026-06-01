-- Implements DESIGN-007 EntitlementManager append query.
INSERT INTO entitlements (
    user_id, tier, status, search_limit_per_24h, allowed_modes,
    expires_at, stripe_customer_id, stripe_subscription_id
)
VALUES ($1, $2, $3, $4, $5, $6, nullif(btrim($7), ''), nullif(btrim($8), ''));
