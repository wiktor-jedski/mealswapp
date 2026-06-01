-- Implements DESIGN-015 DataRetentionPolicy audit-list query.
SELECT id, request_id, coalesce(from_status, ''), to_status, coalesce(note, ''), created_at
FROM data_deletion_audit_entries
WHERE request_id = $1
ORDER BY created_at, id;
