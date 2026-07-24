-- Implements DESIGN-015 DataRetentionPolicy worker claim query.
WITH claimable AS (
    SELECT id
    FROM data_deletion_requests
    WHERE status = 'pending'
       OR (
           status = 'failed'
           AND failure_category = 'transient'
           AND retry_count < 3
           AND (next_attempt_at IS NULL OR next_attempt_at <= $1)
       )
       OR (
           status = 'processing'
           AND next_attempt_at IS NOT NULL
           AND next_attempt_at <= $1
       )
    ORDER BY requested_at, id
    LIMIT $2
    FOR UPDATE SKIP LOCKED
), updated AS (
    UPDATE data_deletion_requests d
    SET status = 'processing',
        next_attempt_at = $1::timestamptz + ($3::bigint * interval '1 millisecond')
    FROM claimable
    WHERE d.id = claimable.id
    RETURNING d.id, d.user_id, d.status, d.requested_at, d.completed_at, coalesce(d.failure_reason, ''),
              coalesce(d.failure_category, ''), d.retry_count, d.next_attempt_at, d.receipt_id, d.receipt_issued_at
)
SELECT *
FROM updated;
