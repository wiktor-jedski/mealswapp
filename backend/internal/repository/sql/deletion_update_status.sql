-- Implements DESIGN-015 DataRetentionPolicy status-update query.
UPDATE data_deletion_requests
SET status = $2,
    completed_at = CASE WHEN $2 = 'completed' THEN now() ELSE completed_at END,
    failure_reason = CASE WHEN $2 = 'failed' THEN nullif(btrim($3), '') ELSE NULL END
WHERE id = $1;
