-- Implements DESIGN-009 UserAdminPanel locked legal deletion retry transition.
WITH eligible AS (
    SELECT request.id, request.failure_category, request.retry_count
    FROM data_deletion_requests AS request
    WHERE request.id = $1
      AND request.user_id = $2
      AND request.status = 'failed'
      AND (
          request.failure_category IN ('permanent', 'unknown')
          OR (request.failure_category = 'transient' AND request.retry_count >= 3)
      )
    FOR UPDATE
), retried AS (
    UPDATE data_deletion_requests AS request
    SET status = 'pending',
        failure_reason = NULL,
        failure_category = NULL,
        retry_count = 0,
        next_attempt_at = NULL
    FROM eligible
    WHERE request.id = eligible.id
    RETURNING request.id, eligible.failure_category, eligible.retry_count
), deletion_audit AS (
    INSERT INTO data_deletion_audit_entries (request_id, from_status, to_status, note)
    SELECT id, 'failed', 'pending', 'admin_retry'
    FROM retried
    RETURNING request_id
)
SELECT retried.id, retried.failure_category, retried.retry_count
FROM retried
JOIN deletion_audit ON deletion_audit.request_id = retried.id;
