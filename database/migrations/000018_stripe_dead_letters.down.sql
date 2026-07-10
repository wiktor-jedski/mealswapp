-- Implements DESIGN-007 StripeWebhookHandler dead-letter persistence.
DROP INDEX IF EXISTS stripe_dead_letters_event_idx;
DROP INDEX IF EXISTS stripe_dead_letters_created_idx;
DROP TABLE IF EXISTS stripe_dead_letters;

DELETE FROM schema_migrations WHERE version = 18;
