-- Implements DESIGN-009 AdminController audit insert query.
INSERT INTO admin_audit_entries (
    admin_user_id, action, entity_type, entity_id, before_snapshot, after_snapshot, request_id
)
VALUES ($1, btrim($2), btrim($3), $4, $5, $6, nullif(btrim($7), ''))
RETURNING id;
