-- Implements DESIGN-008 ProfileController daily-diet idempotency.
SELECT user_id, method, route, key, body_hash, status_code, response_body, created_at, updated_at
FROM checkout_idempotency_keys
WHERE user_id = $1 AND method = $2 AND route = $3 AND key = $4
FOR UPDATE;
