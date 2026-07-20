-- Implements DESIGN-004 JobStatusTracker publication acknowledgement update.
UPDATE mutation_idempotency_keys
SET status_code = $6, response_body = $7, updated_at = NOW()
WHERE user_id = $1 AND method = $2 AND route = $3 AND key = $4 AND body_hash = $5;
