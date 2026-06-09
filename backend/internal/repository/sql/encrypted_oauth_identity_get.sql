-- Implements DESIGN-006 OAuthHandler and DESIGN-013 EncryptionService encrypted lookup query.
SELECT id,
       user_id,
       provider,
       provider_user_id_key_version,
       provider_user_id_nonce,
       provider_user_id_ciphertext,
       provider_user_id_lookup_key_version,
       provider_user_id_digest,
       email_key_version,
       email_nonce,
       email_ciphertext,
       created_at
FROM oauth_identities
WHERE provider = btrim($1)
  AND provider_user_id_lookup_key_version = $2
  AND provider_user_id_digest = $3;
