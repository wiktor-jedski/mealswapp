-- Implements DESIGN-006 AuthController password-reset token consume query.
UPDATE password_reset_tokens
SET used_at = $2
WHERE token_hash = $1
  AND used_at IS NULL
  AND expires_at > $2
RETURNING token_hash, user_id, expires_at, used_at, created_at;
