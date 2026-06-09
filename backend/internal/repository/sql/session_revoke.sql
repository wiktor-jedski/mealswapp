-- Implements DESIGN-006 AuthController session revoke query.
UPDATE user_sessions
SET revoked_at = coalesce(revoked_at, now())
WHERE id = $1
RETURNING id;
