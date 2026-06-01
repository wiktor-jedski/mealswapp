-- Implements DESIGN-007 EntitlementManager.
-- Implements DESIGN-007 TrialTracker.
-- Implements DESIGN-007 UsageLimiter.
DROP TABLE IF EXISTS processed_stripe_events;
DROP INDEX IF EXISTS usage_windows_user_started_idx;
DROP TABLE IF EXISTS usage_windows;
DROP INDEX IF EXISTS entitlements_trial_expiry_idx;
DROP INDEX IF EXISTS entitlements_user_created_idx;
DROP TABLE IF EXISTS entitlements;

DELETE FROM schema_migrations WHERE version = 8;
