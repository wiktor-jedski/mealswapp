-- Implements DESIGN-013 EncryptionService encrypted PII persistence.
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS email_key_version text,
    ADD COLUMN IF NOT EXISTS email_nonce bytea,
    ADD COLUMN IF NOT EXISTS email_ciphertext bytea,
    ADD COLUMN IF NOT EXISTS normalized_email_lookup_key_version text,
    ADD COLUMN IF NOT EXISTS normalized_email_digest text;

CREATE UNIQUE INDEX IF NOT EXISTS users_normalized_email_digest_idx
    ON users (normalized_email_lookup_key_version, normalized_email_digest)
    WHERE normalized_email_digest IS NOT NULL;

ALTER TABLE oauth_identities
    ADD COLUMN IF NOT EXISTS provider_user_id_key_version text,
    ADD COLUMN IF NOT EXISTS provider_user_id_nonce bytea,
    ADD COLUMN IF NOT EXISTS provider_user_id_ciphertext bytea,
    ADD COLUMN IF NOT EXISTS provider_user_id_lookup_key_version text,
    ADD COLUMN IF NOT EXISTS provider_user_id_digest text,
    ADD COLUMN IF NOT EXISTS email_key_version text,
    ADD COLUMN IF NOT EXISTS email_nonce bytea,
    ADD COLUMN IF NOT EXISTS email_ciphertext bytea;

CREATE UNIQUE INDEX IF NOT EXISTS oauth_identities_provider_digest_idx
    ON oauth_identities (provider, provider_user_id_lookup_key_version, provider_user_id_digest)
    WHERE provider_user_id_digest IS NOT NULL;

ALTER TABLE user_profiles
    ADD COLUMN IF NOT EXISTS display_name_key_version text,
    ADD COLUMN IF NOT EXISTS display_name_nonce bytea,
    ADD COLUMN IF NOT EXISTS display_name_ciphertext bytea;

ALTER TABLE search_history
    ADD COLUMN IF NOT EXISTS query_key_version text,
    ADD COLUMN IF NOT EXISTS query_nonce bytea,
    ADD COLUMN IF NOT EXISTS query_ciphertext bytea;

INSERT INTO schema_migrations (version)
VALUES (13)
ON CONFLICT (version) DO NOTHING;
