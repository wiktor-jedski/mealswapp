-- Implements DESIGN-015 DataRetentionPolicy categorized failure query.
UPDATE data_deletion_requests
SET status = 'failed',
    failure_category = $3,
    failure_reason = nullif(btrim($4), ''),
    retry_count = CASE WHEN $3 = 'transient' THEN retry_count + 1 ELSE retry_count END,
    next_attempt_at = CASE WHEN $3 = 'transient' AND retry_count + 1 < 3 THEN $5::timestamptz ELSE NULL END
WHERE id = $1
  AND status = 'processing'
  AND next_attempt_at = $2::timestamptz
RETURNING id;
