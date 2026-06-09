-- Implements DESIGN-015 DataRetentionPolicy request query.
WITH existing AS (
    SELECT id, user_id, status, requested_at, completed_at, coalesce(failure_reason, '') AS failure_reason,
           coalesce(failure_category, '') AS failure_category, retry_count, next_attempt_at, receipt_id, receipt_issued_at
    FROM data_deletion_requests
    WHERE user_id = $1 AND status IN ('pending', 'processing')
    LIMIT 1
), inserted AS (
    INSERT INTO data_deletion_requests (user_id, status)
    SELECT $1, 'pending'
    WHERE NOT EXISTS (SELECT 1 FROM existing)
    RETURNING id, user_id, status, requested_at, completed_at, coalesce(failure_reason, '') AS failure_reason,
              coalesce(failure_category, '') AS failure_category, retry_count, next_attempt_at, receipt_id, receipt_issued_at
), selected AS (
    SELECT * FROM inserted
    UNION ALL
    SELECT * FROM existing
    LIMIT 1
), audit AS (
    INSERT INTO data_deletion_audit_entries (request_id, from_status, to_status, note)
    SELECT id, NULL, status, 'deletion requested' FROM inserted
)
SELECT id, user_id, status, requested_at, completed_at, failure_reason,
       failure_category, retry_count, next_attempt_at, receipt_id, receipt_issued_at
FROM selected;
