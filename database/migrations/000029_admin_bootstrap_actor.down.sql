-- Implements DESIGN-009 AdminController truthful operator-origin bootstrap audit rollback.
DELETE FROM admin_audit_entries WHERE actor_kind = 'operator';

ALTER TABLE admin_audit_entries
    DROP CONSTRAINT IF EXISTS admin_audit_actor_identity,
    ALTER COLUMN admin_user_id SET NOT NULL,
    DROP COLUMN IF EXISTS actor_kind;

DELETE FROM schema_migrations WHERE version = 29;
