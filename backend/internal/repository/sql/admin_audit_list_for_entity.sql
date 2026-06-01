-- Implements DESIGN-009 AdminController audit-list query.
SELECT id, admin_user_id, action, entity_type, entity_id, coalesce(before_snapshot, '{}'::jsonb),
       coalesce(after_snapshot, '{}'::jsonb), coalesce(request_id, ''), created_at
FROM admin_audit_entries
WHERE entity_type = btrim($1) AND entity_id = $2
ORDER BY created_at DESC, id;
