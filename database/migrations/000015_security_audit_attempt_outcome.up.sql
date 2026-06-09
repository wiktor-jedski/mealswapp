-- Implements DESIGN-013 AuditLogger mutation attempt persistence repair.
ALTER TABLE security_audit_entries
    DROP CONSTRAINT IF EXISTS security_audit_entries_outcome_check;

ALTER TABLE security_audit_entries
    ADD CONSTRAINT security_audit_entries_outcome_check
        CHECK (outcome IN ('attempt', 'success', 'failure'));

INSERT INTO schema_migrations (version)
VALUES (15)
ON CONFLICT (version) DO NOTHING;
