-- Implements DESIGN-008 ProfileController durable custom-item create claim and AccountDeleter write lockout.
WITH active_owner AS MATERIALIZED (
    SELECT id
    FROM users
    WHERE id = $1 AND deletion_requested_at IS NULL
    FOR NO KEY UPDATE
)
INSERT INTO mutation_idempotency_keys (user_id, method, route, key, body_hash, status_code, response_body)
SELECT id, 'POST', '/custom-items', $2, $3, 102, '{}'::jsonb
FROM active_owner
ON CONFLICT (user_id, method, route, key) DO NOTHING
RETURNING user_id, method, route, key, body_hash, status_code, response_body, created_at, updated_at;
