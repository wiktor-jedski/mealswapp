-- Implements DESIGN-009 ItemCurator durable manual global-item create claim.
INSERT INTO mutation_idempotency_keys (user_id, method, route, key, body_hash, status_code, response_body)
VALUES ($1, 'POST', '/admin/items', $2, $3, 102, '{}'::jsonb)
ON CONFLICT (user_id, method, route, key) DO NOTHING
RETURNING user_id, method, route, key, body_hash, status_code, response_body, created_at, updated_at;
