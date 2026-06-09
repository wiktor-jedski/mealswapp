-- Implements DESIGN-006 AuthUser and DESIGN-013 EncryptionService encrypted lookup query.
SELECT id,
       email_key_version,
       email_nonce,
       email_ciphertext,
       normalized_email_lookup_key_version,
       normalized_email_digest,
       email_verified,
       role,
       password_hash,
       password_salt,
       created_at,
       updated_at
FROM users
WHERE normalized_email_lookup_key_version = $1
  AND normalized_email_digest = $2
  AND NOT EXISTS (
      SELECT 1
      FROM data_deletion_requests d
      WHERE d.user_id = users.id
        AND d.status IN ('pending', 'processing')
  );
