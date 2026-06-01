-- Implements DESIGN-007 TrialTracker expired-trials query.
SELECT user_id, tier, status, search_limit_per_24h, allowed_modes,
       expires_at, coalesce(stripe_customer_id, ''), coalesce(stripe_subscription_id, ''),
       created_at, updated_at
FROM (
    SELECT DISTINCT ON (user_id)
           user_id, tier, status, search_limit_per_24h, allowed_modes,
           expires_at, stripe_customer_id, stripe_subscription_id, created_at, updated_at
    FROM entitlements
    ORDER BY user_id, created_at DESC, id DESC
) latest
WHERE tier = 'trial' AND status = 'active' AND expires_at <= $1
ORDER BY expires_at, user_id;
