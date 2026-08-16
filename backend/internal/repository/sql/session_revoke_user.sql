-- Implements DESIGN-006 AuthController and DESIGN-008 AccountDeleter retry-safe session revocation.
WITH revoked AS (
    UPDATE user_sessions
    SET revoked_at = coalesce(revoked_at, now())
    WHERE user_id = $1
    RETURNING id
)
SELECT coalesce((SELECT id FROM revoked LIMIT 1), $1::uuid);
