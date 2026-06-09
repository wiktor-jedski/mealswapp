-- Implements DESIGN-013 EncryptionService encrypted PII persistence rollback.
DROP INDEX IF EXISTS oauth_identities_provider_digest_idx;
DROP INDEX IF EXISTS users_normalized_email_digest_idx;

ALTER TABLE search_history
    DROP COLUMN IF EXISTS query_ciphertext,
    DROP COLUMN IF EXISTS query_nonce,
    DROP COLUMN IF EXISTS query_key_version;

ALTER TABLE user_profiles
    DROP COLUMN IF EXISTS display_name_ciphertext,
    DROP COLUMN IF EXISTS display_name_nonce,
    DROP COLUMN IF EXISTS display_name_key_version;

ALTER TABLE oauth_identities
    DROP COLUMN IF EXISTS email_ciphertext,
    DROP COLUMN IF EXISTS email_nonce,
    DROP COLUMN IF EXISTS email_key_version,
    DROP COLUMN IF EXISTS provider_user_id_digest,
    DROP COLUMN IF EXISTS provider_user_id_lookup_key_version,
    DROP COLUMN IF EXISTS provider_user_id_ciphertext,
    DROP COLUMN IF EXISTS provider_user_id_nonce,
    DROP COLUMN IF EXISTS provider_user_id_key_version;

ALTER TABLE users
    DROP COLUMN IF EXISTS normalized_email_digest,
    DROP COLUMN IF EXISTS normalized_email_lookup_key_version,
    DROP COLUMN IF EXISTS email_ciphertext,
    DROP COLUMN IF EXISTS email_nonce,
    DROP COLUMN IF EXISTS email_key_version;

DELETE FROM schema_migrations WHERE version = 13;
