-- Implements DESIGN-009 DataImporter idempotency replay lookup.
SELECT body_hash, status_code, response_body
FROM mutation_idempotency_keys
WHERE user_id = $1 AND method = 'POST' AND route = '/admin/imports' AND key = $2
FOR UPDATE;
