-- Implements DESIGN-007 StripeWebhookHandler dead-letter persistence verification.
SELECT
    event_id,
    event_type,
    failure_category,
    error_message,
    payload_sha256,
    stripe_customer_id,
    stripe_subscription_id,
    user_id
FROM stripe_dead_letters
WHERE event_id = $1;
