-- Implements DESIGN-006 AuthController session-family revoke query.
UPDATE user_sessions
SET revoked_at = coalesce(revoked_at, now())
WHERE refresh_family_id = $1
RETURNING id;
