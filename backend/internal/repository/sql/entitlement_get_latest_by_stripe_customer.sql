-- Implements DESIGN-007 EntitlementManager latest-state by Stripe Customer query.
SELECT user_id, tier, status, search_limit_per_24h, allowed_modes,
       expires_at, coalesce(stripe_customer_id, ''), coalesce(stripe_subscription_id, ''),
       created_at, updated_at
FROM entitlements
WHERE stripe_customer_id = $1
ORDER BY created_at DESC, id DESC
LIMIT 1;
