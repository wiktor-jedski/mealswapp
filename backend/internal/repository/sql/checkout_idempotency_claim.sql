-- Implements DESIGN-008 ProfileController daily-diet idempotency.
INSERT INTO checkout_idempotency_keys (user_id, method, route, key, body_hash, status_code, response_body)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (user_id, method, route, key) DO NOTHING
RETURNING user_id, method, route, key, body_hash, status_code, response_body, created_at, updated_at;
