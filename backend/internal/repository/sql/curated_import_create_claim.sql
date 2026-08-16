-- Implements DESIGN-009 DataImporter idempotency claim when natural identity is absent.
INSERT INTO mutation_idempotency_keys (user_id, method, route, key, body_hash, status_code, response_body)
VALUES ($1, 'POST', '/admin/imports', $2, $3, 102, '{}'::jsonb)
ON CONFLICT (user_id, method, route, key) DO NOTHING
RETURNING user_id;
