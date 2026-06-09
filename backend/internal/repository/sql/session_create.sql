-- Implements DESIGN-006 AuthController session create query.
INSERT INTO user_sessions (
    user_id,
    refresh_token_hash,
    refresh_family_id,
    access_expires_at,
    refresh_expires_at
)
VALUES ($1, $2, $3, $4, $5)
RETURNING id;
