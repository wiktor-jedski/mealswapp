-- Implements DESIGN-007 StripeWebhookHandler idempotency query.
INSERT INTO processed_stripe_events (event_id, event_type, outcome, payload)
VALUES (btrim($1), btrim($2), $3, $4)
ON CONFLICT (event_id) DO NOTHING;
