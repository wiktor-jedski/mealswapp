-- Implements DESIGN-009 AdminController audit insert query.
INSERT INTO admin_audit_entries (
    actor_kind, admin_user_id, action, entity_type, entity_id, before_snapshot, after_snapshot, request_id
)
VALUES ($1, $2, btrim($3), btrim($4), $5, $6, $7, nullif(btrim($8), ''))
RETURNING id;
