-- Implements DESIGN-015 DataRetentionPolicy pseudonymous receipt query.
UPDATE data_deletion_requests
SET status = 'completed',
    user_id = NULL,
    completed_at = $3,
    failure_category = NULL,
    failure_reason = NULL,
    next_attempt_at = NULL,
    receipt_id = $2,
    receipt_issued_at = $3
WHERE id = $1
  AND status = 'processing'
RETURNING id;
