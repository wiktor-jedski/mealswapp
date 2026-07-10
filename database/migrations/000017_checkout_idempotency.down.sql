-- Implements DESIGN-007 SubscriptionController checkout idempotency.
DROP INDEX IF EXISTS checkout_idempotency_user_created_idx;
DROP TABLE IF EXISTS checkout_idempotency_keys;
DROP TABLE IF EXISTS billing_idempotency;

DELETE FROM schema_migrations WHERE version = 17;
