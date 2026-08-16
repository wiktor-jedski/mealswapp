-- Implements DESIGN-015 DataRetentionPolicy pseudonymous receipt query.
UPDATE data_deletion_requests
SET status = 'completed',
    user_id = NULL,
    completed_at = $4,
    failure_category = NULL,
    failure_reason = NULL,
    next_attempt_at = NULL,
    receipt_id = $3,
    receipt_issued_at = $4
WHERE id = $1
  AND status = 'processing'
  AND next_attempt_at = $2::timestamptz
RETURNING id;
