-- Implements DESIGN-006 AuthController session lookup query.
SELECT id,
       user_id,
       refresh_token_hash,
       refresh_family_id,
       access_expires_at,
       refresh_expires_at,
       revoked_at,
       created_at
FROM user_sessions
WHERE refresh_token_hash = $1;
