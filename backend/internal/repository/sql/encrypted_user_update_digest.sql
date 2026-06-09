-- Implements DESIGN-013 EncryptionService lookup digest reindex query.
UPDATE users
SET normalized_email_lookup_key_version = $2,
    normalized_email_digest = $3,
    updated_at = now()
WHERE id = $1
  AND email_ciphertext IS NOT NULL
RETURNING id;
