-- Implements DESIGN-006 AuthController password-reset session-family revocation query.
UPDATE user_sessions
SET revoked_at = coalesce(revoked_at, now())
WHERE user_id = $1
RETURNING id;
