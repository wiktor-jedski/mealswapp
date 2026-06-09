-- Implements DESIGN-006 OAuthHandler and DESIGN-013 EncryptionService encrypted upsert query.
INSERT INTO oauth_identities (
    user_id,
    provider,
    provider_user_id,
    provider_user_id_key_version,
    provider_user_id_nonce,
    provider_user_id_ciphertext,
    provider_user_id_lookup_key_version,
    provider_user_id_digest,
    email,
    email_key_version,
    email_nonce,
    email_ciphertext
)
VALUES (
    $1,
    btrim($2),
    'encrypted:' || $7,
    $3,
    $4,
    $5,
    $6,
    $7,
    'encrypted',
    $8,
    $9,
    $10
)
ON CONFLICT (provider, provider_user_id)
DO UPDATE SET
    user_id = EXCLUDED.user_id,
    provider_user_id_key_version = EXCLUDED.provider_user_id_key_version,
    provider_user_id_nonce = EXCLUDED.provider_user_id_nonce,
    provider_user_id_ciphertext = EXCLUDED.provider_user_id_ciphertext,
    provider_user_id_lookup_key_version = EXCLUDED.provider_user_id_lookup_key_version,
    provider_user_id_digest = EXCLUDED.provider_user_id_digest,
    email_key_version = EXCLUDED.email_key_version,
    email_nonce = EXCLUDED.email_nonce,
    email_ciphertext = EXCLUDED.email_ciphertext
RETURNING id;
