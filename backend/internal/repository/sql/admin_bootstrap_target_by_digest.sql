-- Implements DESIGN-009 AdminController encrypted-email bootstrap lookup.
SELECT u.id,
       u.role,
       u.email_verified,
       u.password_hash,
       u.password_salt,
       EXISTS (
           SELECT 1
           FROM oauth_identities oi
           WHERE oi.user_id = u.id
             AND oi.provider_user_id_key_version IS NOT NULL
             AND octet_length(oi.provider_user_id_nonce) > 0
             AND octet_length(oi.provider_user_id_ciphertext) > 0
             AND oi.provider_user_id_lookup_key_version IS NOT NULL
             AND btrim(oi.provider_user_id_digest) <> ''
       ) AS has_oauth_credential
FROM users u
WHERE u.normalized_email_lookup_key_version = $1
  AND u.normalized_email_digest = $2
FOR UPDATE;
