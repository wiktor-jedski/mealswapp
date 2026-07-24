-- Implements DESIGN-008 AccountDeleter custom-item erasure rollback.
ALTER TABLE data_deletion_requests
    DROP CONSTRAINT IF EXISTS data_deletion_requests_user_id_fkey;

UPDATE data_deletion_requests AS request
SET user_id = NULL
WHERE user_id IS NOT NULL
  AND NOT EXISTS (SELECT 1 FROM users WHERE id = request.user_id);

ALTER TABLE data_deletion_requests
    ADD CONSTRAINT data_deletion_requests_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL;

DROP INDEX IF EXISTS data_deletion_requests_claim_idx;

CREATE INDEX data_deletion_requests_claim_idx
    ON data_deletion_requests (status, next_attempt_at, requested_at)
    WHERE status IN ('pending', 'failed');

ALTER TABLE users
    DROP COLUMN IF EXISTS deletion_requested_at;

DELETE FROM schema_migrations WHERE version = 26;
