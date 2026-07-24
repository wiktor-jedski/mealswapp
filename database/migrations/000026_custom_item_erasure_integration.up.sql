-- Implements DESIGN-008 AccountDeleter custom-item write lockout and retry-safe erasure.
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS deletion_requested_at timestamptz;

UPDATE users AS account
SET deletion_requested_at = coalesce(account.deletion_requested_at, request.requested_at, now()),
    updated_at = now()
FROM (
    SELECT user_id, min(requested_at) AS requested_at
    FROM data_deletion_requests
    WHERE user_id IS NOT NULL AND status <> 'completed'
    GROUP BY user_id
) AS request
WHERE account.id = request.user_id
  AND account.deletion_requested_at IS NULL;

ALTER TABLE data_deletion_requests
    DROP CONSTRAINT IF EXISTS data_deletion_requests_user_id_fkey;

DROP INDEX IF EXISTS data_deletion_requests_claim_idx;

CREATE INDEX data_deletion_requests_claim_idx
    ON data_deletion_requests (status, next_attempt_at, requested_at)
    WHERE status IN ('pending', 'processing', 'failed');

INSERT INTO schema_migrations (version)
VALUES (26)
ON CONFLICT (version) DO NOTHING;
