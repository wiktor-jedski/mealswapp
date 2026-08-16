-- Implements DESIGN-008 ProfileController durable custom-item create response completion.
UPDATE mutation_idempotency_keys
SET status_code = 201, response_body = $4, updated_at = NOW()
WHERE user_id = $1 AND method = 'POST' AND route = '/custom-items' AND key = $2 AND body_hash = $3
RETURNING user_id, method, route, key, body_hash, status_code, response_body, created_at, updated_at;
