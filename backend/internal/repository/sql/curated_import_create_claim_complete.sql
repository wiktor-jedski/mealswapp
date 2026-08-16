-- Implements DESIGN-009 DataImporter immutable idempotency response completion.
UPDATE mutation_idempotency_keys
SET status_code = 201, response_body = $4, updated_at = now()
WHERE user_id = $1 AND method = 'POST' AND route = '/admin/imports' AND key = $2 AND body_hash = $3
RETURNING status_code, response_body;
