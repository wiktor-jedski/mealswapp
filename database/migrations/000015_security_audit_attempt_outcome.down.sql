-- Implements DESIGN-013 AuditLogger mutation attempt persistence repair rollback.
DO $$
BEGIN
    IF to_regclass('public.security_audit_entries') IS NOT NULL THEN
        ALTER TABLE security_audit_entries
            DROP CONSTRAINT IF EXISTS security_audit_entries_outcome_check;

        UPDATE security_audit_entries
        SET outcome = 'failure'
        WHERE outcome = 'attempt';

        ALTER TABLE security_audit_entries
            ADD CONSTRAINT security_audit_entries_outcome_check
                CHECK (outcome IN ('success', 'failure'));
    END IF;
END $$;

DELETE FROM schema_migrations
WHERE version = 15;
