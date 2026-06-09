-- Implements DESIGN-015 DataRetentionPolicy categorized failure query.
UPDATE data_deletion_requests
SET status = 'failed',
    failure_category = $2,
    failure_reason = nullif(btrim($3), ''),
    retry_count = CASE WHEN $2 = 'transient' THEN retry_count + 1 ELSE retry_count END,
    next_attempt_at = CASE WHEN $2 = 'transient' AND retry_count + 1 < 3 THEN $4::timestamptz ELSE NULL END
WHERE id = $1
  AND status = 'processing'
RETURNING id;
