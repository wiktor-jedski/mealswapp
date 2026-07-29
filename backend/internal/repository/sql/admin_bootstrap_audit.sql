-- Implements DESIGN-009 AdminController truthful operator-origin bootstrap audit.
INSERT INTO admin_audit_entries (
    admin_user_id, actor_kind, action, entity_type, entity_id, request_id
)
VALUES (NULL, 'operator', 'bootstrap_administrator', 'user', $1, $2)
RETURNING id, created_at;
