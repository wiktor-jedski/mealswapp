-- Implements DESIGN-009 ItemCurator durable manual global-item create claim read.
SELECT user_id, method, route, key, body_hash, status_code, response_body, created_at, updated_at
FROM mutation_idempotency_keys
WHERE user_id = $1 AND method = 'POST' AND route = '/admin/items' AND key = $2
FOR UPDATE;
