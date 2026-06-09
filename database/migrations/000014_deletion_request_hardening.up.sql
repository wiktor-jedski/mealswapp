-- Implements DESIGN-015 DataRetentionPolicy hardened deletion workflow metadata.
ALTER TABLE data_deletion_requests
    DROP CONSTRAINT IF EXISTS data_deletion_requests_user_id_fkey;

ALTER TABLE data_deletion_requests
    ALTER COLUMN user_id DROP NOT NULL;

ALTER TABLE data_deletion_requests
    ADD CONSTRAINT data_deletion_requests_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE data_deletion_requests
    ADD COLUMN IF NOT EXISTS failure_category text CHECK (failure_category IS NULL OR failure_category IN ('transient', 'permanent', 'unknown')),
    ADD COLUMN IF NOT EXISTS retry_count integer NOT NULL DEFAULT 0 CHECK (retry_count >= 0),
    ADD COLUMN IF NOT EXISTS next_attempt_at timestamptz,
    ADD COLUMN IF NOT EXISTS receipt_id uuid,
    ADD COLUMN IF NOT EXISTS receipt_issued_at timestamptz;

CREATE INDEX IF NOT EXISTS data_deletion_requests_claim_idx
    ON data_deletion_requests (status, next_attempt_at, requested_at)
    WHERE status IN ('pending', 'failed');

CREATE UNIQUE INDEX IF NOT EXISTS data_deletion_requests_receipt_idx
    ON data_deletion_requests (receipt_id)
    WHERE receipt_id IS NOT NULL;

INSERT INTO schema_migrations (version)
VALUES (14)
ON CONFLICT (version) DO NOTHING;
