-- Implements DESIGN-007 StripeWebhookHandler dead-letter persistence.
INSERT INTO stripe_dead_letters (
    event_id,
    event_type,
    failure_category,
    error_message,
    payload_sha256,
    stripe_customer_id,
    stripe_subscription_id,
    user_id
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);
