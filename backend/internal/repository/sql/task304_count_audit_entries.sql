-- Implements DESIGN-009 AdminController acceptance proof.
SELECT count(*)
FROM admin_audit_entries
WHERE request_id = $1;
