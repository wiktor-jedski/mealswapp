-- Implements DESIGN-006 AuthUser.
DROP INDEX IF EXISTS password_reset_tokens_expiry_idx;
DROP INDEX IF EXISTS password_reset_tokens_user_idx;
DROP TABLE IF EXISTS password_reset_tokens;
DROP INDEX IF EXISTS user_sessions_refresh_hash_idx;
DROP INDEX IF EXISTS user_sessions_user_idx;
DROP TABLE IF EXISTS user_sessions;
DROP INDEX IF EXISTS oauth_identities_user_idx;
DROP TABLE IF EXISTS oauth_identities;
DROP TABLE IF EXISTS users;

DELETE FROM schema_migrations WHERE version = 6;
