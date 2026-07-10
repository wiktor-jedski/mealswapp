-- Implements DESIGN-015 DataRetentionPolicy hardened deletion workflow metadata rollback.
DROP INDEX IF EXISTS data_deletion_requests_receipt_idx;
DROP INDEX IF EXISTS data_deletion_requests_claim_idx;

DO $$
BEGIN
    IF to_regclass('public.data_deletion_requests') IS NOT NULL THEN
        ALTER TABLE data_deletion_requests
            DROP CONSTRAINT IF EXISTS data_deletion_requests_user_id_fkey;

        DELETE FROM data_deletion_requests
        WHERE user_id IS NULL;

        ALTER TABLE data_deletion_requests
            ALTER COLUMN user_id SET NOT NULL;

        ALTER TABLE data_deletion_requests
            ADD CONSTRAINT data_deletion_requests_user_id_fkey
                FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

        ALTER TABLE data_deletion_requests
            DROP COLUMN IF EXISTS receipt_issued_at,
            DROP COLUMN IF EXISTS receipt_id,
            DROP COLUMN IF EXISTS next_attempt_at,
            DROP COLUMN IF EXISTS retry_count,
            DROP COLUMN IF EXISTS failure_category;
    END IF;
END $$;

DELETE FROM schema_migrations
WHERE version = 14;
