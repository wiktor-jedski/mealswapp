-- Implements DESIGN-008 ProfileController durable custom-item create claim read.
SELECT user_id, method, route, key, body_hash, status_code, response_body, created_at, updated_at
FROM mutation_idempotency_keys
WHERE user_id = $1 AND method = 'POST' AND route = '/custom-items' AND key = $2
FOR UPDATE;
