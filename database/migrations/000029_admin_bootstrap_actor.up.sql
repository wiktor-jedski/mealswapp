-- Implements DESIGN-009 AdminController truthful operator-origin bootstrap audit.
ALTER TABLE admin_audit_entries
    ADD COLUMN IF NOT EXISTS actor_kind text NOT NULL DEFAULT 'administrator'
        CHECK (actor_kind IN ('administrator', 'operator')),
    ALTER COLUMN admin_user_id DROP NOT NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'admin_audit_actor_identity'
    ) THEN
        ALTER TABLE admin_audit_entries
            ADD CONSTRAINT admin_audit_actor_identity CHECK (
                (actor_kind = 'administrator' AND admin_user_id IS NOT NULL)
                OR (actor_kind = 'operator' AND admin_user_id IS NULL)
            );
    END IF;
END $$;

INSERT INTO schema_migrations (version)
VALUES (29)
ON CONFLICT (version) DO NOTHING;
